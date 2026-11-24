//! 6 位配对码：生命周期管理 + 基于 SPAKE2 的配对握手。
//!
//! 码只作为 SPAKE2 口令，不上网传输；被动窃听者无法离线爆破。
//! 在线猜测由「单码最多 3 次握手、5 分钟有效、成功即作废」限制。

use std::sync::Mutex;
use std::time::{Duration, Instant};

use anyhow::{bail, Result};
use hmac::{Hmac, Mac};
use rand::Rng;
use sha2::Sha256;
use spake2::{Ed25519Group, Identity, Password, Spake2};
use tokio::io::{AsyncRead, AsyncWrite};

use super::protocol::{
    derive_key, direction_keys, read_plain, write_plain, DeviceMeta, SecureReader, SecureWriter,
};

pub const CODE_TTL: Duration = Duration::from_secs(5 * 60);
pub const MAX_ATTEMPTS: u8 = 3;
const PAKE_IDENTITY: &[u8] = b"ecopaste-sync-v1";

type HmacSha256 = Hmac<Sha256>;

struct ActiveCode {
    code: String,
    expires_at: Instant,
    expires_at_ms: i64,
    attempts: u8,
}

/// 当前展示中的配对码（至多一个）。
#[derive(Default)]
pub struct Pairing {
    active: Mutex<Option<ActiveCode>>,
}

/// 展示给前端的配对码快照。
#[derive(Debug, Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PairingCode {
    pub code: String,
    /// Unix 毫秒。
    pub expires_at: i64,
}

impl Pairing {
    /// 生成新的 6 位码（覆盖旧码）。
    pub fn start(&self) -> PairingCode {
        let code = format!("{:06}", rand::thread_rng().gen_range(0..1_000_000u32));
        let expires_at_ms = chrono::Utc::now().timestamp_millis() + CODE_TTL.as_millis() as i64;
        *self.active.lock().expect("pairing poisoned") = Some(ActiveCode {
            code: code.clone(),
            expires_at: Instant::now() + CODE_TTL,
            expires_at_ms,
            attempts: 0,
        });
        PairingCode {
            code,
            expires_at: expires_at_ms,
        }
    }

    pub fn cancel(&self) {
        *self.active.lock().expect("pairing poisoned") = None;
    }

    /// 码是否仍可用（存在、未过期、未超次数）。过期的码在此处顺带清掉。
    pub fn is_active(&self) -> bool {
        let mut guard = self.active.lock().expect("pairing poisoned");
        let alive = guard
            .as_ref()
            .is_some_and(|c| c.expires_at > Instant::now() && c.attempts < MAX_ATTEMPTS);
        if !alive {
            *guard = None;
        }
        alive
    }

    pub fn snapshot(&self) -> Option<PairingCode> {
        if !self.is_active() {
            return None;
        }
        self.active
            .lock()
            .expect("pairing poisoned")
            .as_ref()
            .map(|c| PairingCode {
                code: c.code.clone(),
                expires_at: c.expires_at_ms,
            })
    }

    /// 开始一次握手尝试：占用一次机会并返回口令；码不可用返回 `None`。
    pub fn begin_attempt(&self) -> Option<String> {
        let mut guard = self.active.lock().expect("pairing poisoned");
        let active = guard.as_mut()?;
        if active.expires_at <= Instant::now() || active.attempts >= MAX_ATTEMPTS {
            *guard = None;
            return None;
        }
        active.attempts += 1;
        Some(active.code.clone())
    }

    /// 握手失败后调用：次数用尽则让码作废。
    pub fn finish_failed(&self) {
        let mut guard = self.active.lock().expect("pairing poisoned");
        if guard.as_ref().is_some_and(|c| c.attempts >= MAX_ATTEMPTS) {
            *guard = None;
        }
    }
}

/// 配对握手产物。
pub struct PairOutcome {
    pub peer_id: String,
    pub peer_name: String,
    pub peer_platform: String,
    pub peer_port: u16,
    pub long_key: [u8; 32],
}

fn confirm_tag(key: &[u8], from: &str, to: &str) -> Vec<u8> {
    let mut mac = HmacSha256::new_from_slice(key).expect("hmac accepts any key length");
    mac.update(b"ecopaste-sync/pair-confirm");
    mac.update(from.as_bytes());
    mac.update(b"\0");
    mac.update(to.as_bytes());
    mac.finalize().into_bytes().to_vec()
}

fn verify_confirm(key: &[u8], from: &str, to: &str, tag: &[u8]) -> bool {
    let mut mac = HmacSha256::new_from_slice(key).expect("hmac accepts any key length");
    mac.update(b"ecopaste-sync/pair-confirm");
    mac.update(from.as_bytes());
    mac.update(b"\0");
    mac.update(to.as_bytes());
    mac.verify_slice(tag).is_ok()
}

/// 配对握手（两端对称）：SPAKE2 → 互相确认 → 加密通道内交换设备信息 → 派生长期密钥。
///
/// `initiator` 仅用于区分加密方向，不影响 SPAKE2。
pub async fn run_handshake<R, W>(
    reader: R,
    writer: W,
    code: &str,
    my_id: &str,
    peer_id: &str,
    my_meta: DeviceMeta,
    initiator: bool,
) -> Result<PairOutcome>
where
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let mut reader = reader;
    let mut writer = writer;

    let (state, my_msg) = Spake2::<Ed25519Group>::start_symmetric(
        &Password::new(code.as_bytes()),
        &Identity::new(PAKE_IDENTITY),
    );
    write_plain(&mut writer, &my_msg).await?;
    let their_msg: Vec<u8> = read_plain(&mut reader).await?;
    let Ok(shared) = state.finish(&their_msg) else {
        bail!("pairing key agreement failed");
    };

    write_plain(&mut writer, &confirm_tag(&shared, my_id, peer_id)).await?;
    let their_tag: Vec<u8> = read_plain(&mut reader).await?;
    if !verify_confirm(&shared, peer_id, my_id, &their_tag) {
        bail!("pairing code mismatch");
    }

    let mut ids = [my_id, peer_id];
    ids.sort_unstable();
    let salt = format!("{}|{}", ids[0], ids[1]);
    let base = derive_key(&shared, salt.as_bytes(), b"ecopaste-sync/pair-channel");
    let (send_key, recv_key) = direction_keys(&base, initiator);
    let mut secure_w = SecureWriter::new(writer, &send_key);
    let mut secure_r = SecureReader::new(reader, &recv_key);

    secure_w.send(&my_meta).await?;
    let meta: DeviceMeta = secure_r.recv().await?;

    Ok(PairOutcome {
        peer_id: peer_id.to_owned(),
        peer_name: meta.name,
        peer_platform: meta.platform,
        peer_port: meta.port,
        long_key: derive_key(&shared, salt.as_bytes(), b"ecopaste-sync/long-term"),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn meta(name: &str) -> DeviceMeta {
        DeviceMeta {
            name: name.into(),
            platform: "windows".into(),
            port: 1,
        }
    }

    async fn handshake(code_a: &str, code_b: &str) -> (Result<PairOutcome>, Result<PairOutcome>) {
        let (a, b) = tokio::io::duplex(1 << 16);
        let (ar, aw) = tokio::io::split(a);
        let (br, bw) = tokio::io::split(b);
        tokio::join!(
            run_handshake(ar, aw, code_a, "dev-a", "dev-b", meta("A"), false),
            run_handshake(br, bw, code_b, "dev-b", "dev-a", meta("B"), true),
        )
    }

    #[tokio::test]
    async fn matching_codes_pair_and_agree_on_key() {
        let (a, b) = handshake("123456", "123456").await;
        let (a, b) = (a.unwrap(), b.unwrap());
        assert_eq!(a.long_key, b.long_key);
        assert_eq!(a.peer_name, "B");
        assert_eq!(b.peer_name, "A");
    }

    #[tokio::test]
    async fn wrong_code_fails_on_both_sides() {
        let (a, b) = handshake("123456", "654321").await;
        assert!(a.is_err());
        assert!(b.is_err());
    }

    #[test]
    fn code_is_six_digits_and_single_use_limits_attempts() {
        let pairing = Pairing::default();
        let code = pairing.start();
        assert_eq!(code.code.len(), 6);
        assert!(code.code.chars().all(|c| c.is_ascii_digit()));

        for _ in 0..MAX_ATTEMPTS {
            assert!(pairing.begin_attempt().is_some());
            pairing.finish_failed();
        }
        assert!(pairing.begin_attempt().is_none());
        assert!(!pairing.is_active());
    }

    #[test]
    fn cancel_invalidates_code() {
        let pairing = Pairing::default();
        pairing.start();
        pairing.cancel();
        assert!(pairing.begin_attempt().is_none());
    }
}
