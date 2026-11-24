//! 已配对设备间的会话握手：用长期密钥做双向挑战-应答认证，再派生会话密钥。

use anyhow::{bail, Result};
use hmac::{Hmac, Mac};
use rand::RngCore;
use sha2::Sha256;
use tokio::io::{AsyncRead, AsyncWrite};

use super::protocol::{
    derive_key, direction_keys, read_plain, write_plain, SecureReader, SecureWriter,
};

type HmacSha256 = Hmac<Sha256>;

fn tag(
    key: &[u8; 32],
    role: &[u8],
    nonce_s: &[u8],
    nonce_c: &[u8],
    client: &str,
    server: &str,
) -> HmacSha256 {
    let mut mac = HmacSha256::new_from_slice(key).expect("hmac accepts any key length");
    mac.update(b"ecopaste-sync/session");
    mac.update(role);
    mac.update(nonce_s);
    mac.update(nonce_c);
    mac.update(client.as_bytes());
    mac.update(b"\0");
    mac.update(server.as_bytes());
    mac
}

fn session_secure<R, W>(
    reader: R,
    writer: W,
    key: &[u8; 32],
    nonce_s: &[u8],
    nonce_c: &[u8],
    initiator: bool,
) -> (SecureReader<R>, SecureWriter<W>)
where
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let salt = [nonce_s, nonce_c].concat();
    let base = derive_key(key, &salt, b"ecopaste-sync/session-keys");
    let (send_key, recv_key) = direction_keys(&base, initiator);
    (
        SecureReader::new(reader, &recv_key),
        SecureWriter::new(writer, &send_key),
    )
}

/// 发起方：已发送 `Intro` 并确认对端 `Hello` 后调用。
pub async fn client_handshake<R, W>(
    mut reader: R,
    mut writer: W,
    key: &[u8; 32],
    my_id: &str,
    peer_id: &str,
) -> Result<(SecureReader<R>, SecureWriter<W>)>
where
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let nonce_s: Vec<u8> = read_plain(&mut reader).await?;
    let mut nonce_c = vec![0u8; 32];
    rand::thread_rng().fill_bytes(&mut nonce_c);

    let client_tag = tag(key, b"c", &nonce_s, &nonce_c, my_id, peer_id)
        .finalize()
        .into_bytes()
        .to_vec();
    write_plain(&mut writer, &(nonce_c.clone(), client_tag)).await?;

    let server_tag: Vec<u8> = read_plain(&mut reader).await?;
    if tag(key, b"s", &nonce_s, &nonce_c, my_id, peer_id)
        .verify_slice(&server_tag)
        .is_err()
    {
        bail!("peer failed authentication");
    }

    Ok(session_secure(
        reader, writer, key, &nonce_s, &nonce_c, true,
    ))
}

/// 监听方：已读取 `Intro`、查到对端密钥并回复 `Hello` 后调用。
pub async fn server_handshake<R, W>(
    mut reader: R,
    mut writer: W,
    key: &[u8; 32],
    my_id: &str,
    peer_id: &str,
) -> Result<(SecureReader<R>, SecureWriter<W>)>
where
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let mut nonce_s = vec![0u8; 32];
    rand::thread_rng().fill_bytes(&mut nonce_s);
    write_plain(&mut writer, &nonce_s).await?;

    let (nonce_c, client_tag): (Vec<u8>, Vec<u8>) = read_plain(&mut reader).await?;
    if nonce_c.len() != 32
        || tag(key, b"c", &nonce_s, &nonce_c, peer_id, my_id)
            .verify_slice(&client_tag)
            .is_err()
    {
        bail!("peer failed authentication");
    }

    let server_tag = tag(key, b"s", &nonce_s, &nonce_c, peer_id, my_id)
        .finalize()
        .into_bytes()
        .to_vec();
    write_plain(&mut writer, &server_tag).await?;

    Ok(session_secure(
        reader, writer, key, &nonce_s, &nonce_c, false,
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sync::protocol::Message;

    async fn run(client_key: [u8; 32], server_key: [u8; 32]) -> (Result<()>, Result<()>) {
        let (a, b) = tokio::io::duplex(1 << 16);
        let (ar, aw) = tokio::io::split(a);
        let (br, bw) = tokio::io::split(b);

        let client = async move {
            let (mut r, mut w) = client_handshake(ar, aw, &client_key, "client", "server").await?;
            w.send(&Message::Ping).await?;
            assert_eq!(r.recv::<Message>().await?, Message::Pong);
            Ok(())
        };
        let server = async move {
            let (mut r, mut w) = server_handshake(br, bw, &server_key, "server", "client").await?;
            assert_eq!(r.recv::<Message>().await?, Message::Ping);
            w.send(&Message::Pong).await?;
            Ok(())
        };
        tokio::join!(client, server)
    }

    #[tokio::test]
    async fn shared_key_authenticates_and_carries_traffic() {
        let (c, s) = run([5u8; 32], [5u8; 32]).await;
        c.unwrap();
        s.unwrap();
    }

    #[tokio::test]
    async fn different_keys_are_rejected() {
        let (c, s) = run([5u8; 32], [6u8; 32]).await;
        assert!(c.is_err());
        assert!(s.is_err());
    }
}
