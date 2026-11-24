//! 同步线协议：消息定义、长度前缀帧、AEAD 加密通道。
//!
//! 握手前的帧是明文（仅含身份 id、SPAKE2 公开消息、随机数、HMAC 标签），
//! 握手完成后全部走 [`SecureWriter`] / [`SecureReader`]（ChaCha20-Poly1305，按方向分密钥、计数器作 nonce）。

use anyhow::{anyhow, bail, Context, Result};
use chacha20poly1305::aead::{Aead, KeyInit};
use chacha20poly1305::{ChaCha20Poly1305, Key, Nonce};
use hkdf::Hkdf;
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

/// 协议版本；不兼容变更时递增，握手阶段不匹配直接拒绝。
pub const PROTOCOL_VERSION: u8 = 3;
/// 优先监听的固定端口，便于手动输入 IP 连接；被占用时回落到随机端口。
pub const DEFAULT_PORT: u16 = 52526;
/// 单帧（密文）上限。大负载由上层分块。
pub const MAX_FRAME: usize = 16 * 1024 * 1024;
/// 图片 / 文件分块大小。
pub const CHUNK_SIZE: usize = 1024 * 1024;

/// 连接建立后的第一帧，声明本连接用途。
#[derive(Debug, Serialize, Deserialize)]
pub enum Intro {
    /// 一次性配对（SPAKE2）。
    Pair { version: u8, device_id: String },
    /// 已配对设备的会话。
    /// `port` 为发起方自身的监听端口，供对端记录地址、后续回连。
    Session {
        version: u8,
        device_id: String,
        port: u16,
    },
}

/// 握手阶段监听方对 [`Intro`] 的应答。
#[derive(Debug, Serialize, Deserialize)]
pub struct Hello {
    pub device_id: String,
}

/// 配对成功后在加密通道内交换的设备信息。
#[derive(Debug, Serialize, Deserialize)]
pub struct DeviceMeta {
    pub name: String,
    pub platform: String,
    /// 本机监听端口。
    pub port: u16,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct FileEntry {
    /// 以 `/` 分隔的相对路径（顶层条目即文件名）。
    pub rel_path: String,
    pub is_dir: bool,
    pub size: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum Message {
    Ping,
    Pong,
    /// 对端解除配对。
    Unpair,
    /// `ts` 为复制发生时的逻辑时间戳（毫秒），接收方只采用比本机已知内容更新的一条。
    ClipText {
        ts: u64,
        plain: String,
        html: Option<String>,
        rtf: Option<String>,
    },
    ImageStart {
        ts: u64,
        id: u64,
        total: u64,
    },
    ImageChunk {
        id: u64,
        data: Vec<u8>,
    },
    ImageEnd {
        id: u64,
    },
    /// 文件清单；内容不预传，对端粘贴时再 [`Message::FetchFile`]。
    ClipFiles {
        ts: u64,
        offer: String,
        entries: Vec<FileEntry>,
    },
    FetchFile {
        req: u64,
        offer: String,
        index: u32,
        offset: u64,
        len: u32,
    },
    /// 空 `data` 表示已到文件末尾。
    FileData {
        req: u64,
        data: Vec<u8>,
    },
    FileError {
        req: u64,
        message: String,
    },
}

pub async fn write_frame<W: AsyncWrite + Unpin>(w: &mut W, bytes: &[u8]) -> Result<()> {
    if bytes.len() > MAX_FRAME {
        bail!("frame too large: {}", bytes.len());
    }
    let mut framed = Vec::with_capacity(4 + bytes.len());
    framed.extend_from_slice(&(bytes.len() as u32).to_be_bytes());
    framed.extend_from_slice(bytes);
    w.write_all(&framed).await?;
    w.flush().await?;
    Ok(())
}

pub async fn read_frame<R: AsyncRead + Unpin>(r: &mut R) -> Result<Vec<u8>> {
    let mut len = [0u8; 4];
    r.read_exact(&mut len).await.context("connection closed")?;
    let len = u32::from_be_bytes(len) as usize;
    if len > MAX_FRAME {
        bail!("frame too large: {len}");
    }
    let mut buf = vec![0u8; len];
    r.read_exact(&mut buf).await.context("connection closed")?;
    Ok(buf)
}

/// 写一个明文（握手阶段）帧。
pub async fn write_plain<W: AsyncWrite + Unpin, T: Serialize>(w: &mut W, value: &T) -> Result<()> {
    let bytes = postcard::to_stdvec(value)?;
    write_frame(w, &bytes).await
}

/// 读一个明文（握手阶段）帧。
pub async fn read_plain<R: AsyncRead + Unpin, T: DeserializeOwned>(r: &mut R) -> Result<T> {
    let bytes = read_frame(r).await?;
    postcard::from_bytes(&bytes).map_err(|err| anyhow!("malformed frame: {err}"))
}

/// 用 HKDF-SHA256 从 `secret` 派生 32 字节子密钥。
pub fn derive_key(secret: &[u8], salt: &[u8], info: &[u8]) -> [u8; 32] {
    let hk = Hkdf::<Sha256>::new(Some(salt), secret);
    let mut out = [0u8; 32];
    hk.expand(info, &mut out)
        .expect("32 bytes is a valid hkdf output length");
    out
}

/// 一对方向密钥：`(本端发送, 本端接收)`。
pub fn direction_keys(base: &[u8; 32], initiator: bool) -> ([u8; 32], [u8; 32]) {
    let c2s = derive_key(base, b"", b"ecopaste-sync/c2s");
    let s2c = derive_key(base, b"", b"ecopaste-sync/s2c");
    if initiator {
        (c2s, s2c)
    } else {
        (s2c, c2s)
    }
}

fn nonce_for(counter: u64) -> Nonce {
    let mut n = [0u8; 12];
    n[4..].copy_from_slice(&counter.to_be_bytes());
    *Nonce::from_slice(&n)
}

pub struct SecureWriter<W> {
    inner: W,
    cipher: ChaCha20Poly1305,
    counter: u64,
}

pub struct SecureReader<R> {
    inner: R,
    cipher: ChaCha20Poly1305,
    counter: u64,
}

impl<W: AsyncWrite + Unpin> SecureWriter<W> {
    pub fn new(inner: W, key: &[u8; 32]) -> Self {
        Self {
            inner,
            cipher: ChaCha20Poly1305::new(Key::from_slice(key)),
            counter: 0,
        }
    }

    pub async fn send<T: Serialize>(&mut self, value: &T) -> Result<()> {
        let plain = postcard::to_stdvec(value)?;
        let sealed = self
            .cipher
            .encrypt(&nonce_for(self.counter), plain.as_slice())
            .map_err(|_| anyhow!("encrypt failed"))?;
        self.counter += 1;
        write_frame(&mut self.inner, &sealed).await
    }
}

impl<R: AsyncRead + Unpin> SecureReader<R> {
    pub fn new(inner: R, key: &[u8; 32]) -> Self {
        Self {
            inner,
            cipher: ChaCha20Poly1305::new(Key::from_slice(key)),
            counter: 0,
        }
    }

    pub async fn recv<T: DeserializeOwned>(&mut self) -> Result<T> {
        let sealed = read_frame(&mut self.inner).await?;
        let plain = self
            .cipher
            .decrypt(&nonce_for(self.counter), sealed.as_slice())
            .map_err(|_| anyhow!("authentication failed"))?;
        self.counter += 1;
        postcard::from_bytes(&plain).map_err(|err| anyhow!("malformed message: {err}"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn secure_channel_round_trips_messages() {
        let (a, b) = tokio::io::duplex(1 << 20);
        let key = [7u8; 32];
        let mut writer = SecureWriter::new(a, &key);
        let mut reader = SecureReader::new(b, &key);

        let msg = Message::ClipText {
            ts: 1,
            plain: "hello".into(),
            html: Some("<b>hello</b>".into()),
            rtf: None,
        };
        writer.send(&msg).await.unwrap();
        writer.send(&Message::Ping).await.unwrap();

        assert_eq!(reader.recv::<Message>().await.unwrap(), msg);
        assert_eq!(reader.recv::<Message>().await.unwrap(), Message::Ping);
    }

    #[tokio::test]
    async fn secure_channel_rejects_wrong_key() {
        let (a, b) = tokio::io::duplex(1 << 20);
        let mut writer = SecureWriter::new(a, &[1u8; 32]);
        let mut reader = SecureReader::new(b, &[2u8; 32]);

        writer.send(&Message::Ping).await.unwrap();
        assert!(reader.recv::<Message>().await.is_err());
    }

    #[tokio::test]
    async fn secure_channel_rejects_replayed_frame() {
        let (mut a, b) = tokio::io::duplex(1 << 20);
        let key = [3u8; 32];
        let cipher = ChaCha20Poly1305::new(Key::from_slice(&key));
        let plain = postcard::to_stdvec(&Message::Ping).unwrap();
        let sealed = cipher.encrypt(&nonce_for(0), plain.as_slice()).unwrap();
        write_frame(&mut a, &sealed).await.unwrap();
        write_frame(&mut a, &sealed).await.unwrap();

        let mut reader = SecureReader::new(b, &key);
        assert!(reader.recv::<Message>().await.is_ok());
        assert!(reader.recv::<Message>().await.is_err());
    }

    #[tokio::test]
    async fn oversized_frame_is_rejected() {
        let (mut a, mut b) = tokio::io::duplex(64);
        a.write_all(&(u32::MAX).to_be_bytes()).await.unwrap();
        assert!(read_frame(&mut b).await.is_err());
    }
}
