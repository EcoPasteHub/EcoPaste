use futures_util::StreamExt;
use hostname::get;
use keyring::Entry;
use quick_xml::events::Event;
use quick_xml::Reader;
use reqwest::header::{HeaderMap, HeaderValue};
use reqwest::{Client, Method, StatusCode};
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use tauri::command;
use tokio::fs::File;
use tokio::io::AsyncWriteExt;
use tokio_util::io::ReaderStream;
use url::Url;

static UPLOAD_CANCELLED: AtomicBool = AtomicBool::new(false);

const SERVICE_NAME: &str = "EcoPaste.WebDAV";
const ACCOUNT_NAME: &str = "default";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebdavConfig {
    pub address: String,
    pub username: String,
    pub password: String,
    pub path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebdavBackupFile {
    pub file_name: String,
    pub size: Option<u64>,
    pub modified: Option<String>,
}

fn entry() -> Result<Entry, String> {
    Entry::new(SERVICE_NAME, ACCOUNT_NAME).map_err(|error| error.to_string())
}

fn build_base_url(config: &WebdavConfig) -> Result<Url, String> {
    let mut base = config.address.trim().to_string();
    if !base.ends_with('/') {
        base.push('/');
    }
    let base_url = Url::parse(&base).map_err(|error| error.to_string())?;
    let mut path = config.path.trim().to_string();
    if path.starts_with('/') {
        path = path[1..].to_string();
    }
    let mut url = base_url;
    if !path.is_empty() {
        url = url.join(&format!("{path}/")).map_err(|error| error.to_string())?;
    }
    Ok(url)
}

fn build_client() -> Result<Client, String> {
    Client::builder()
        .user_agent("EcoPaste")
        .build()
        .map_err(|error| error.to_string())
}

async fn request_propfind(
    client: &Client,
    config: &WebdavConfig,
    depth: &str,
) -> Result<String, String> {
    let url = build_base_url(config)?;
    let mut headers = HeaderMap::new();
    headers.insert("Depth", HeaderValue::from_str(depth).map_err(|e| e.to_string())?);
    headers.insert(
        "Content-Type",
        HeaderValue::from_static("application/xml"),
    );
    let body = r#"<?xml version="1.0" encoding="utf-8" ?><d:propfind xmlns:d="DAV:"><d:allprop/></d:propfind>"#;
    let response = client
        .request(Method::from_bytes(b"PROPFIND").map_err(|e| e.to_string())?, url)
        .basic_auth(&config.username, Some(&config.password))
        .headers(headers)
        .body(body)
        .send()
        .await
        .map_err(|error| error.to_string())?;
    let status = response.status();
    let text = response.text().await.map_err(|error| error.to_string())?;
    if status != StatusCode::OK && status != StatusCode::MULTI_STATUS {
        return Err(format!("status: {status}, body: {text}"));
    }
    Ok(text)
}

fn parse_propfind(xml: &str) -> Vec<WebdavBackupFile> {
    let mut reader = Reader::from_str(xml);
    reader.trim_text(true);
    let mut buf = Vec::new();
    let mut files = Vec::new();
    let mut current: Option<WebdavBackupFile> = None;
    let mut current_field: Option<String> = None;
    let mut is_dir = false;
    let mut in_resourcetype = false;

    loop {
        match reader.read_event_into(&mut buf) {
            Ok(Event::Start(event)) => {
                let name = event.name().as_ref().to_owned();
                let name = local_name(&name);
                match name {
                    b"response" => {
                        current = Some(WebdavBackupFile {
                            file_name: String::new(),
                            size: None,
                            modified: None,
                        });
                        is_dir = false;
                    }
                    b"href" | b"displayname" | b"getcontentlength" | b"getlastmodified" => {
                        current_field = Some(String::from_utf8_lossy(name).to_string());
                    }
                    b"resourcetype" => {
                        in_resourcetype = true;
                    }
                    b"collection" => {
                        if in_resourcetype {
                            is_dir = true;
                        }
                    }
                    _ => {}
                }
            }
            Ok(Event::Text(event)) => {
                if let (Some(field), Some(item)) = (&current_field, current.as_mut()) {
                    let text = event.unescape().unwrap_or_default().to_string();
                    match field.as_str() {
                        "href" | "displayname" => {
                            if !text.is_empty() {
                                item.file_name = text;
                            }
                        }
                        "getcontentlength" => {
                            if let Ok(size) = text.parse::<u64>() {
                                item.size = Some(size);
                            }
                        }
                        "getlastmodified" => {
                            if !text.is_empty() {
                                item.modified = Some(text);
                            }
                        }
                        _ => {}
                    }
                }
            }
            Ok(Event::End(event)) => {
                let name = event.name().as_ref().to_owned();
                let name = local_name(&name);
                match name {
                    b"response" => {
                        if let Some(item) = current.take() {
                            if !is_dir {
                                files.push(item);
                            }
                        }
                        current_field = None;
                        in_resourcetype = false;
                    }
                    b"resourcetype" => {
                        in_resourcetype = false;
                    }
                    _ => {
                        current_field = None;
                    }
                }
            }
            Ok(Event::Eof) => break,
            Err(_) => break,
            _ => {}
        }
        buf.clear();
    }

    files
}

fn local_name(name: &[u8]) -> &[u8] {
    match name.iter().rposition(|b| *b == b':') {
        Some(pos) => &name[pos + 1..],
        None => name,
    }
}

fn normalize_file_name(raw: &str) -> String {
    let trimmed = raw.trim();
    if trimmed.ends_with('/') {
        return String::new();
    }
    let decoded = match Url::parse(trimmed) {
        Ok(url) => url.path().to_string(),
        Err(_) => trimmed.to_string(),
    };
    decoded
        .split('/')
        .filter(|part| !part.is_empty())
        .last()
        .unwrap_or("")
        .to_string()
}

#[command]
pub async fn set_config(config: WebdavConfig) -> Result<(), String> {
    let value = serde_json::to_string(&config).map_err(|error| error.to_string())?;
    let entry = entry()?;
    entry.set_password(&value).map_err(|error| error.to_string())
}

#[command]
pub async fn get_config() -> Result<Option<WebdavConfig>, String> {
    let entry = entry()?;
    match entry.get_password() {
        Ok(value) => {
            let config =
                serde_json::from_str::<WebdavConfig>(&value).map_err(|e| e.to_string())?;
            Ok(Some(config))
        }
        Err(error) => {
            if error.to_string().contains("NoEntry") || error.to_string().contains("not found") {
                return Ok(None);
            }
            Ok(None)
        }
    }
}

#[command]
pub fn get_computer_name() -> Result<String, String> {
    let name = get().map_err(|error| error.to_string())?;
    let value = name.to_string_lossy().to_string();
    if value.trim().is_empty() {
        return Ok("Unknown".to_string());
    }
    Ok(value)
}

async fn ensure_remote_dir(client: &Client, config: &WebdavConfig) -> Result<(), String> {
    let base = config.address.trim().trim_end_matches('/').to_string();
    let path = config.path.trim().trim_matches('/').to_string();
    if path.is_empty() {
        return Ok(());
    }
    let mut current = base;
    for part in path.split('/') {
        if part.is_empty() {
            continue;
        }
        current = format!("{current}/{part}");
        let url = Url::parse(&format!("{current}/")).map_err(|e| e.to_string())?;
        let response = client
            .request(Method::from_bytes(b"MKCOL").map_err(|e| e.to_string())?, url)
            .basic_auth(&config.username, Some(&config.password))
            .send()
            .await
            .map_err(|e| e.to_string())?;
        let status = response.status();
        if status == StatusCode::CREATED || status == StatusCode::METHOD_NOT_ALLOWED {
            continue;
        }
        if status.is_success() {
            continue;
        }
        let text = response.text().await.unwrap_or_default();
        return Err(format!("status: {status}, body: {text}"));
    }
    Ok(())
}

#[command]
pub async fn test_config(config: WebdavConfig) -> Result<(), String> {
    let client = build_client()?;
    ensure_remote_dir(&client, &config).await?;
    request_propfind(&client, &config, "0").await?;
    Ok(())
}

#[command]
pub async fn list_backups() -> Result<Vec<WebdavBackupFile>, String> {
    let config = get_config().await?.ok_or("missing config")?;
    let client = build_client()?;
    ensure_remote_dir(&client, &config).await?;
    let xml = request_propfind(&client, &config, "1").await?;
    let mut files = parse_propfind(&xml);
    let base_url = build_base_url(&config)?;

    files = files
        .into_iter()
        .filter_map(|mut item| {
            let file_name = normalize_file_name(&item.file_name);
            if file_name.is_empty() {
                return None;
            }
            if let Ok(url) = base_url.join(&file_name) {
                item.file_name = normalize_file_name(url.as_str());
            } else {
                item.file_name = file_name;
            }
            Some(item)
        })
        .collect();

    Ok(files)
}

#[command]
pub async fn upload_backup(file_path: String, file_name: String) -> Result<(), String> {
    UPLOAD_CANCELLED.store(false, Ordering::SeqCst);
    let config = get_config().await?.ok_or("missing config")?;
    let url = build_base_url(&config)?.join(&file_name).map_err(|e| e.to_string())?;
    let client = build_client()?;
    ensure_remote_dir(&client, &config).await?;
    let file = File::open(file_path).await.map_err(|e| e.to_string())?;
    let stream = ReaderStream::new(file).take_while(|_| {
        let cancelled = UPLOAD_CANCELLED.load(Ordering::SeqCst);
        futures_util::future::ready(!cancelled)
    });
    let body = reqwest::Body::wrap_stream(stream);
    let response = client
        .put(url)
        .basic_auth(&config.username, Some(&config.password))
        .body(body)
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if UPLOAD_CANCELLED.load(Ordering::SeqCst) {
        return Err("cancelled".to_string());
    }

    if !response.status().is_success() {
        let status = response.status();
        let text = response.text().await.unwrap_or_default();
        return Err(format!("status: {status}, body: {text}"));
    }

    Ok(())
}

#[command]
pub async fn cancel_upload() -> Result<(), String> {
    UPLOAD_CANCELLED.store(true, Ordering::SeqCst);
    Ok(())
}

#[command]
pub async fn download_backup(file_name: String) -> Result<String, String> {
    let config = get_config().await?.ok_or("missing config")?;
    let url = build_base_url(&config)?.join(&file_name).map_err(|e| e.to_string())?;
    let client = build_client()?;
    ensure_remote_dir(&client, &config).await?;
    let response = client
        .get(url)
        .basic_auth(&config.username, Some(&config.password))
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !response.status().is_success() {
        let status = response.status();
        let text = response.text().await.unwrap_or_default();
        return Err(format!("status: {status}, body: {text}"));
    }

    let mut temp_path = std::env::temp_dir();
    let safe_name = Path::new(&file_name)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("backup.zip");
    temp_path.push(safe_name);
    let mut file = File::create(&temp_path).await.map_err(|e| e.to_string())?;
    let mut stream = response.bytes_stream();

    while let Some(chunk) = stream.next().await {
        let data = chunk.map_err(|e| e.to_string())?;
        file.write_all(&data).await.map_err(|e| e.to_string())?;
    }

    Ok(temp_path.to_string_lossy().to_string())
}

#[command]
pub async fn delete_backup(file_name: String) -> Result<(), String> {
    let config = get_config().await?.ok_or("missing config")?;
    let url = build_base_url(&config)?.join(&file_name).map_err(|e| e.to_string())?;
    let client = build_client()?;
    ensure_remote_dir(&client, &config).await?;
    let response = client
        .delete(url)
        .basic_auth(&config.username, Some(&config.password))
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !response.status().is_success() {
        let status = response.status();
        let text = response.text().await.unwrap_or_default();
        return Err(format!("status: {status}, body: {text}"));
    }

    Ok(())
}

#[command]
pub async fn create_slim_database(source_db_path: String, target_db_path: String) -> Result<(), String> {
    let source = Connection::open(&source_db_path).map_err(|e| e.to_string())?;
    if let Some(parent) = PathBuf::from(&target_db_path).parent() {
        fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    let _ = fs::remove_file(&target_db_path);
    let mut target = Connection::open(&target_db_path).map_err(|e| e.to_string())?;
    target
        .execute_batch(
            "CREATE TABLE IF NOT EXISTS history (
                id TEXT PRIMARY KEY,
                type TEXT,
                \"group\" TEXT,
                value TEXT,
                search TEXT,
                count INTEGER,
                width INTEGER,
                height INTEGER,
                favorite INTEGER DEFAULT 0,
                createTime TEXT,
                note TEXT,
                subtype TEXT
            );",
        )
        .map_err(|e| e.to_string())?;

    let mut stmt = source
        .prepare(
            "SELECT id, type, \"group\", value, search, count, width, height, favorite, createTime, note, subtype
             FROM history WHERE \"group\" = 'text'",
        )
        .map_err(|e| e.to_string())?;

    let rows = stmt
        .query_map([], |row| {
            Ok((
                row.get::<_, String>(0)?,
                row.get::<_, String>(1)?,
                row.get::<_, String>(2)?,
                row.get::<_, String>(3)?,
                row.get::<_, String>(4)?,
                row.get::<_, Option<i64>>(5)?,
                row.get::<_, Option<i64>>(6)?,
                row.get::<_, Option<i64>>(7)?,
                row.get::<_, Option<i64>>(8)?,
                row.get::<_, String>(9)?,
                row.get::<_, Option<String>>(10)?,
                row.get::<_, Option<String>>(11)?,
            ))
        })
        .map_err(|e| e.to_string())?;

    let tx = target.transaction().map_err(|e| e.to_string())?;
    for row in rows {
        let (
            id,
            r#type,
            group,
            value,
            search,
            count,
            width,
            height,
            favorite,
            create_time,
            note,
            subtype,
        ) = row.map_err(|e| e.to_string())?;
        tx.execute(
            "INSERT INTO history (id, type, \"group\", value, search, count, width, height, favorite, createTime, note, subtype)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)",
            params![
                id,
                r#type,
                group,
                value,
                search,
                count,
                width,
                height,
                favorite,
                create_time,
                note,
                subtype
            ],
        )
        .map_err(|e| e.to_string())?;
    }
    tx.commit().map_err(|e| e.to_string())?;
    Ok(())
}
