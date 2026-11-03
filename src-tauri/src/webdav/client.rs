//! WebDAV HTTP 客户端，封装底层 WebDAV 请求交互。

use std::time::Duration;

use anyhow::{anyhow, Context};
use base64::engine::general_purpose::STANDARD as BASE64;
use base64::Engine;
use reqwest::header::{HeaderMap, HeaderValue, AUTHORIZATION, CONTENT_TYPE};
use reqwest::{Client, Method, StatusCode};
use url::Url;

use crate::core::{AppError, Result};

#[derive(Clone)]
pub struct WebDavClient {
    client: Client,
    base_url: Url,
    auth_header: String,
}

impl WebDavClient {
    /// 根据配置初始化 WebDAV 客户端。
    pub fn new(endpoint: &str, username: &str, password: &str) -> Result<Self> {
        let trimmed = endpoint.trim();
        if trimmed.is_empty() {
            return Err(AppError::Other(anyhow!("WebDAV 服务器地址不能为空")));
        }

        let formatted = if !trimmed.starts_with("http://") && !trimmed.starts_with("https://") {
            format!("https://{trimmed}")
        } else {
            trimmed.to_owned()
        };

        let mut base_url = Url::parse(&formatted)
            .map_err(|err| AppError::Other(anyhow!("无效的 WebDAV 服务器地址: {err}")))?;

        // 保证基础 URL 以 `/` 结尾，以便正确拼接子路径
        if !base_url.path().ends_with('/') {
            let path = format!("{}/", base_url.path());
            base_url.set_path(&path);
        }

        let credentials = format!("{username}:{password}");
        let auth_header = format!("Basic {}", BASE64.encode(credentials.as_bytes()));

        let client = Client::builder()
            .timeout(Duration::from_secs(30))
            .build()
            .context("failed to build reqwest client")?;

        Ok(Self {
            client,
            base_url,
            auth_header,
        })
    }

    /// 构建带认证头的默认请求头。
    fn headers(&self) -> HeaderMap {
        let mut headers = HeaderMap::new();
        if let Ok(val) = HeaderValue::from_str(&self.auth_header) {
            headers.insert(AUTHORIZATION, val);
        }
        headers
    }

    /// 拼接完整路径 URL。
    fn build_url(&self, relative_path: &str) -> Result<Url> {
        let clean = relative_path.trim_start_matches('/');
        self.base_url
            .join(clean)
            .map_err(|err| AppError::Other(anyhow!("构建 WebDAV URL 失败: {err}")))
    }

    /// 测试与 WebDAV 服务器的连通性与认证。
    pub async fn test_connection(&self) -> Result<()> {
        let mkcol_method = Method::from_bytes(b"PROPFIND")
            .map_err(|err| AppError::Other(anyhow!("无效 HTTP 方法: {err}")))?;

        let mut headers = self.headers();
        headers.insert("Depth", HeaderValue::from_static("0"));

        let response = self
            .client
            .request(mkcol_method, self.base_url.clone())
            .headers(headers)
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("无法连接到 WebDAV 服务器: {err}")))?;

        let status = response.status();
        if status == StatusCode::UNAUTHORIZED || status == StatusCode::FORBIDDEN {
            return Err(AppError::Other(anyhow!(
                "WebDAV 认证失败，请检查用户名或密码"
            )));
        }

        // PROPFIND 返回 200, 207 (Multi-Status), 404, 405 均表示连接与凭证有效
        if status.is_success()
            || status == StatusCode::MULTI_STATUS
            || status == StatusCode::NOT_FOUND
            || status == StatusCode::METHOD_NOT_ALLOWED
        {
            return Ok(());
        }

        // 兜底发一次 GET 请求测试
        let get_resp = self
            .client
            .get(self.base_url.clone())
            .headers(self.headers())
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("无法连接到 WebDAV 服务器: {err}")))?;

        let get_status = get_resp.status();
        if get_status == StatusCode::UNAUTHORIZED || get_status == StatusCode::FORBIDDEN {
            return Err(AppError::Other(anyhow!(
                "WebDAV 认证失败，请检查用户名或密码"
            )));
        }

        Ok(())
    }

    /// 确保远程目录存在（若不存在则递归逐级创建）。
    pub async fn ensure_dir(&self, dir_path: &str) -> Result<()> {
        let trimmed = dir_path.trim().trim_matches('/');
        if trimmed.is_empty() {
            return Ok(());
        }

        let parts: Vec<&str> = trimmed.split('/').filter(|s| !s.is_empty()).collect();
        let mut current = String::new();

        let mkcol_method = Method::from_bytes(b"MKCOL")
            .map_err(|err| AppError::Other(anyhow!("无效 HTTP 方法: {err}")))?;

        for part in parts {
            if !current.is_empty() {
                current.push('/');
            }
            current.push_str(part);
            let dir_url = format!("{current}/");
            let url = self.build_url(&dir_url)?;

            let resp = self
                .client
                .request(mkcol_method.clone(), url)
                .headers(self.headers())
                .send()
                .await
                .map_err(|err| AppError::Other(anyhow!("创建 WebDAV 目录失败: {err}")))?;

            let status = resp.status();
            // 201 Created, 200 OK, 204 No Content, 405 Method Not Allowed (已存在) 均为合理状态
            if status != StatusCode::CREATED
                && status != StatusCode::OK
                && status != StatusCode::NO_CONTENT
                && status != StatusCode::METHOD_NOT_ALLOWED
            {
                log::warn!("MKCOL for {current} returned status {status}");
            }
        }

        Ok(())
    }

    /// 上传文件到 WebDAV 路径。
    pub async fn put(&self, path: &str, data: Vec<u8>, content_type: &str) -> Result<()> {
        let url = self.build_url(path)?;
        let mut headers = self.headers();
        if let Ok(val) = HeaderValue::from_str(content_type) {
            headers.insert(CONTENT_TYPE, val);
        }

        let resp = self
            .client
            .put(url)
            .headers(headers)
            .body(data)
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("WebDAV 上传失败: {err}")))?;

        let status = resp.status();
        if !status.is_success() && status != StatusCode::CREATED && status != StatusCode::NO_CONTENT
        {
            let error_text = resp.text().await.unwrap_or_default();
            return Err(AppError::Other(anyhow!(
                "WebDAV 上传文件失败 (HTTP {status}): {error_text}"
            )));
        }

        Ok(())
    }

    /// 下载 WebDAV 文件。若文件不存在返回 `Ok(None)`。
    pub async fn get(&self, path: &str) -> Result<Option<Vec<u8>>> {
        let url = self.build_url(path)?;
        let resp = self
            .client
            .get(url)
            .headers(self.headers())
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("WebDAV 下载失败: {err}")))?;

        let status = resp.status();
        if status == StatusCode::NOT_FOUND {
            return Ok(None);
        }

        if !status.is_success() {
            let error_text = resp.text().await.unwrap_or_default();
            return Err(AppError::Other(anyhow!(
                "WebDAV 下载文件失败 (HTTP {status}): {error_text}"
            )));
        }

        let bytes = resp
            .bytes()
            .await
            .map_err(|err| AppError::Other(anyhow!("读取 WebDAV 响应数据失败: {err}")))?;

        Ok(Some(bytes.to_vec()))
    }

    /// 检查远程文件是否存在。
    pub async fn exists(&self, path: &str) -> Result<bool> {
        let url = self.build_url(path)?;
        let resp = self
            .client
            .head(url)
            .headers(self.headers())
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("检查 WebDAV 文件状态失败: {err}")))?;

        let status = resp.status();
        Ok(status.is_success())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_client_url_construction() {
        let client = WebDavClient::new("https://dav.example.com/remote/", "user", "pass")
            .expect("create client");
        let url = client
            .build_url("EcoPaste/items/batch_1.json")
            .expect("build url");
        assert_eq!(
            url.as_str(),
            "https://dav.example.com/remote/EcoPaste/items/batch_1.json"
        );
    }

    #[test]
    fn test_client_url_without_trailing_slash() {
        let client = WebDavClient::new("https://dav.example.com/remote", "user", "pass")
            .expect("create client");
        let url = client
            .build_url("/EcoPaste/manifest.json")
            .expect("build url");
        assert_eq!(
            url.as_str(),
            "https://dav.example.com/remote/EcoPaste/manifest.json"
        );
    }

    #[test]
    fn test_empty_endpoint_returns_error() {
        let client = WebDavClient::new("   ", "user", "pass");
        assert!(client.is_err());
    }
}
