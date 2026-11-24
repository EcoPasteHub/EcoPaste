//! HTTPS client for the EcoPasteServer REST API (login, items, blobs, history) with certificate pinning.

use std::time::Duration;

use anyhow::{anyhow, bail, Context, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use super::pin;

pub const DEFAULT_PORT: u16 = 52530;

/// Failures the UI wants to tell apart.
#[derive(Debug, thiserror::Error)]
pub enum CloudError {
    #[error("cannot reach the server")]
    Unreachable,
    #[error("certificate does not match the pinned fingerprint")]
    PinMismatch,
    #[error("invalid username or password")]
    BadCredentials,
    #[error("too many failed attempts")]
    Locked,
    #[error("not signed in")]
    NotSignedIn,
    #[error("server address is invalid")]
    BadAddress,
}

/// Accepts `host`, `host:port` or `https://host[:port]` and returns `(base_url, host)`.
pub fn normalize_server_url(input: &str) -> Result<String, CloudError> {
    let trimmed = input.trim().trim_end_matches('/');
    let trimmed = trimmed
        .strip_prefix("https://")
        .or_else(|| trimmed.strip_prefix("wss://"))
        .unwrap_or(trimmed);
    if trimmed.is_empty() || trimmed.contains('/') || trimmed.contains(' ') || trimmed.contains('@')
    {
        return Err(CloudError::BadAddress);
    }
    let has_port = match trimmed.rsplit_once(':') {
        Some((host, port)) if !host.ends_with(']') || host.contains(']') => {
            port.parse::<u16>().is_ok()
        }
        _ => false,
    };
    let base = if has_port {
        format!("https://{trimmed}")
    } else {
        format!("https://{trimmed}:{DEFAULT_PORT}")
    };
    url::Url::parse(&base).map_err(|_| CloudError::BadAddress)?;
    Ok(base)
}

/// 是否为小写十六进制 SHA-256。
pub fn is_hash(s: &str) -> bool {
    s.len() == 64
        && s.bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

pub fn ws_url(base: &str, path_and_query: &str) -> String {
    format!(
        "wss://{}{}",
        base.trim_start_matches("https://"),
        path_and_query
    )
}

#[derive(Clone)]
pub struct Api {
    pub base: String,
    http: reqwest::Client,
    token: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LoginResult {
    pub token: String,
    pub device_id: i64,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Me {
    pub latest_seq: i64,
}

/// One synced entry as stored by the server.
#[derive(Debug, Clone, Deserialize)]
pub struct ServerItem {
    pub seq: i64,
    pub kind: String,
    pub origin: i64,
    pub payload: serde_json::Value,
}

#[derive(Debug, Serialize)]
pub struct NewItem<'a> {
    pub hash: &'a str,
    pub kind: &'a str,
    pub size: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub blob: Option<&'a str>,
    pub payload: serde_json::Value,
}

fn classify(err: reqwest::Error) -> anyhow::Error {
    let text = format!("{err:?}");
    if text.contains("fingerprint does not match") {
        anyhow::Error::new(CloudError::PinMismatch)
    } else if err.is_connect() || err.is_timeout() || err.is_request() {
        anyhow::Error::new(CloudError::Unreachable)
    } else {
        anyhow!(err)
    }
}

impl Api {
    /// `pin = None` builds a probing client that accepts any certificate (used only to read its fingerprint).
    pub fn new(
        base: &str,
        pin: Option<String>,
        token: Option<String>,
    ) -> Result<(Self, std::sync::Arc<pin::PinVerifier>)> {
        let (config, verifier) = pin::client_config(pin);
        let http = reqwest::Client::builder()
            .use_preconfigured_tls((*config).clone())
            .connect_timeout(Duration::from_secs(8))
            .timeout(Duration::from_secs(60))
            .build()
            .context("build http client")?;
        Ok((
            Self {
                base: base.to_owned(),
                http,
                token,
            },
            verifier,
        ))
    }

    fn url(&self, path: &str) -> String {
        format!("{}{path}", self.base)
    }

    fn authed(&self, req: reqwest::RequestBuilder) -> reqwest::RequestBuilder {
        match &self.token {
            Some(token) => req.bearer_auth(token),
            None => req,
        }
    }

    async fn check(resp: reqwest::Response) -> Result<reqwest::Response> {
        let status = resp.status();
        if status.is_success() {
            return Ok(resp);
        }
        match status.as_u16() {
            401 => Err(CloudError::NotSignedIn.into()),
            429 => Err(CloudError::Locked.into()),
            _ => {
                let body = resp.text().await.unwrap_or_default();
                bail!("server answered {status}: {body}")
            }
        }
    }

    /// Fingerprint of the certificate the server presents, without trusting it.
    pub async fn probe(base: &str) -> Result<String> {
        let (api, verifier) = Self::new(base, None, None)?;
        // Any answer (including 401) completes the TLS handshake we care about.
        api.http
            .get(api.url("/api/me"))
            .send()
            .await
            .map_err(classify)?;
        verifier
            .seen()
            .ok_or_else(|| CloudError::Unreachable.into())
    }

    pub async fn login(
        &self,
        username: &str,
        password: &str,
        device_name: &str,
        platform: &str,
    ) -> Result<LoginResult> {
        let resp = self
            .http
            .post(self.url("/api/login"))
            .json(&serde_json::json!({
                "username": username, "password": password,
                "deviceName": device_name, "platform": platform,
            }))
            .send()
            .await
            .map_err(classify)?;
        match resp.status().as_u16() {
            401 => Err(CloudError::BadCredentials.into()),
            429 => Err(CloudError::Locked.into()),
            _ => Ok(Self::check(resp).await?.json().await?),
        }
    }

    /// 吊销本设备的令牌（登出）。
    pub async fn logout(&self) -> Result<()> {
        let resp = self
            .authed(self.http.post(self.url("/api/logout")))
            .send()
            .await
            .map_err(classify)?;
        Self::check(resp).await?;
        Ok(())
    }

    pub async fn me(&self) -> Result<Me> {
        let resp = self
            .authed(self.http.get(self.url("/api/me")))
            .send()
            .await
            .map_err(classify)?;
        Ok(Self::check(resp).await?.json().await?)
    }

    pub async fn history(&self, limit: u32) -> Result<Vec<ServerItem>> {
        #[derive(Deserialize)]
        struct Page {
            items: Vec<ServerItem>,
        }
        let resp = self
            .authed(
                self.http
                    .get(self.url(&format!("/api/history?limit={limit}"))),
            )
            .send()
            .await
            .map_err(classify)?;
        let page: Page = Self::check(resp).await?.json().await?;
        Ok(page.items)
    }

    pub async fn post_item(&self, item: &NewItem<'_>) -> Result<i64> {
        let resp = self
            .authed(self.http.post(self.url("/api/items")))
            .json(item)
            .send()
            .await
            .map_err(classify)?;
        let value: serde_json::Value = Self::check(resp).await?.json().await?;
        Ok(value.get("seq").and_then(|s| s.as_i64()).unwrap_or(0))
    }

    /// Uploads `bytes` under its SHA-256 and returns that hash.
    pub async fn put_blob(&self, bytes: Vec<u8>) -> Result<String> {
        let hash: String = Sha256::digest(&bytes)
            .iter()
            .map(|b| format!("{b:02x}"))
            .collect();
        let resp = self
            .authed(self.http.put(self.url(&format!("/api/blobs/{hash}"))))
            .body(bytes)
            .send()
            .await
            .map_err(classify)?;
        Self::check(resp).await?;
        Ok(hash)
    }

    pub async fn get_blob(&self, hash: &str) -> Result<Vec<u8>> {
        let resp = self
            .authed(self.http.get(self.url(&format!("/api/blobs/{hash}"))))
            .send()
            .await
            .map_err(classify)?;
        Ok(Self::check(resp).await?.bytes().await?.to_vec())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn server_url_accepts_common_forms() {
        assert_eq!(
            normalize_server_url("example.com").unwrap(),
            "https://example.com:52530"
        );
        assert_eq!(
            normalize_server_url(" https://example.com:9000/ ").unwrap(),
            "https://example.com:9000"
        );
        assert_eq!(
            normalize_server_url("203.0.113.5:1234").unwrap(),
            "https://203.0.113.5:1234"
        );
        assert!(normalize_server_url("").is_err());
        assert!(normalize_server_url("a b").is_err());
        assert!(normalize_server_url("host/path").is_err());
    }

    #[test]
    fn ws_url_swaps_scheme() {
        assert_eq!(
            ws_url("https://h:1", "/api/ws?since=0"),
            "wss://h:1/api/ws?since=0"
        );
    }
}
