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
        let propfind_method = Method::from_bytes(b"PROPFIND")
            .map_err(|err| AppError::Other(anyhow!("无效 HTTP 方法: {err}")))?;

        for part in parts {
            if !current.is_empty() {
                current.push('/');
            }
            current.push_str(part);
            let dir_url = format!("{current}/");
            let url = self.build_url(&dir_url)?;

            // 优先通过 PROPFIND 检查目录是否已存在
            let mut prop_headers = self.headers();
            prop_headers.insert("Depth", HeaderValue::from_static("0"));
            if let Ok(check_resp) = self
                .client
                .request(propfind_method.clone(), url.clone())
                .headers(prop_headers)
                .send()
                .await
            {
                let check_status = check_resp.status();
                if check_status.is_success() || check_status == StatusCode::MULTI_STATUS {
                    continue;
                }
            }

            let resp = self
                .client
                .request(mkcol_method.clone(), url.clone())
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
                let error_text = resp.text().await.unwrap_or_default();
                log::warn!("MKCOL for {url} returned status {status}: {error_text}");

                if status == StatusCode::FORBIDDEN
                    || status == StatusCode::UNAUTHORIZED
                    || status == StatusCode::INTERNAL_SERVER_ERROR
                {
                    return Err(AppError::Other(anyhow!(
                        "创建 WebDAV 目录 '{current}' 失败: {}",
                        format_http_error(status, &error_text)
                    )));
                }
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
            .put(url.clone())
            .headers(headers)
            .body(data)
            .send()
            .await
            .map_err(|err| AppError::Other(anyhow!("WebDAV 上传失败: {err}")))?;

        let status = resp.status();
        if !status.is_success() && status != StatusCode::CREATED && status != StatusCode::NO_CONTENT
        {
            let error_text = resp.text().await.unwrap_or_default();
            log::error!("WebDAV PUT {url} failed: HTTP {status}, body: {error_text}");
            return Err(AppError::Other(anyhow!(
                "WebDAV 上传文件失败: {}",
                format_http_error(status, &error_text)
            )));
        }

        Ok(())
    }

    /// 下载 WebDAV 文件。若文件不存在返回 `Ok(None)`。
    pub async fn get(&self, path: &str) -> Result<Option<Vec<u8>>> {
        let url = self.build_url(path)?;
        let resp = self
            .client
            .get(url.clone())
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
            log::error!("WebDAV GET {url} failed: HTTP {status}, body: {error_text}");
            return Err(AppError::Other(anyhow!(
                "WebDAV 下载文件失败: {}",
                format_http_error(status, &error_text)
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

/// 格式化 HTTP 错误响应，避免直接向用户界面抛出大段未转义的 HTML 报错页面。
fn format_http_error(status: StatusCode, body: &str) -> String {
    let trimmed = body.trim();
    if trimmed.is_empty() {
        return format!("HTTP {status}");
    }

    if trimmed.contains("<html") || trimmed.contains("<!DOCTYPE") || trimmed.contains("<body") {
        if status == StatusCode::INTERNAL_SERVER_ERROR {
            return "HTTP 500 (服务端内部错误。若为自建 Apache/Nextcloud/WebDAV 服务，通常为目录写入权限不足、只读挂载或服务端 DavLockDB 锁数据库未配置)".to_string();
        }
        if status == StatusCode::FORBIDDEN {
            return "HTTP 403 (权限不足或禁止访问)".to_string();
        }
        if status == StatusCode::NOT_FOUND {
            return "HTTP 404 (远端路径未找到)".to_string();
        }
        if let Some(start) = trimmed.find("<title>") {
            if let Some(end) = trimmed[start..].find("</title>") {
                let title = &trimmed[start + 7..start + end];
                return format!("HTTP {status}: {}", title.trim());
            }
        }
        return format!("HTTP {status} (服务端返回错误页面)");
    }

    format!("HTTP {status}: {trimmed}")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_http_error_html_stripping() {
        let html = "<!DOCTYPE HTML><html><head><title>500 Internal Server Error</title></head><body><h1>Error</h1></body></html>";
        let formatted = format_http_error(StatusCode::INTERNAL_SERVER_ERROR, html);
        assert!(!formatted.contains("<html"));
        assert!(formatted.contains("HTTP 500"));
    }

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
