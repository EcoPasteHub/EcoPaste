//! TLS certificate pinning: the cloud server uses a self-signed certificate, so trust is established by
//! comparing its SHA-256 fingerprint (confirmed by the user on first connect) instead of a CA chain.

use std::sync::{Arc, Mutex};

use rustls::client::danger::{HandshakeSignatureValid, ServerCertVerified, ServerCertVerifier};
use rustls::crypto::{ring, CryptoProvider};
use rustls::pki_types::{CertificateDer, ServerName, UnixTime};
use rustls::{ClientConfig, DigitallySignedStruct, Error, SignatureScheme};
use sha2::{Digest, Sha256};

/// Formats a certificate as an uppercase, colon-separated SHA-256 fingerprint (same format as the server shows).
pub fn fingerprint(der: &[u8]) -> String {
    Sha256::digest(der)
        .iter()
        .map(|b| format!("{b:02X}"))
        .collect::<Vec<_>>()
        .join(":")
}

/// Normalizes user-provided fingerprints (case / separators) for comparison.
pub fn normalize(fp: &str) -> String {
    fp.chars()
        .filter(char::is_ascii_hexdigit)
        .collect::<String>()
        .to_ascii_uppercase()
}

#[derive(Debug)]
pub struct PinVerifier {
    provider: Arc<CryptoProvider>,
    /// `Some` = connection must present exactly this certificate; `None` = probe mode (accept and record).
    pin: Option<String>,
    seen: Mutex<Option<String>>,
}

impl PinVerifier {
    /// Fingerprint of the certificate presented in the last handshake.
    pub fn seen(&self) -> Option<String> {
        self.seen.lock().expect("pin verifier poisoned").clone()
    }
}

impl ServerCertVerifier for PinVerifier {
    fn verify_server_cert(
        &self,
        end_entity: &CertificateDer<'_>,
        _intermediates: &[CertificateDer<'_>],
        _server_name: &ServerName<'_>,
        _ocsp_response: &[u8],
        _now: UnixTime,
    ) -> Result<ServerCertVerified, Error> {
        let seen = fingerprint(end_entity.as_ref());
        *self.seen.lock().expect("pin verifier poisoned") = Some(seen.clone());

        match &self.pin {
            Some(pin) if normalize(pin) != normalize(&seen) => Err(Error::General(
                "server certificate fingerprint does not match the pinned one".into(),
            )),
            _ => Ok(ServerCertVerified::assertion()),
        }
    }

    fn verify_tls12_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, Error> {
        rustls::crypto::verify_tls12_signature(
            message,
            cert,
            dss,
            &self.provider.signature_verification_algorithms,
        )
    }

    fn verify_tls13_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, Error> {
        rustls::crypto::verify_tls13_signature(
            message,
            cert,
            dss,
            &self.provider.signature_verification_algorithms,
        )
    }

    fn supported_verify_schemes(&self) -> Vec<SignatureScheme> {
        self.provider
            .signature_verification_algorithms
            .supported_schemes()
    }
}

/// Builds a rustls client config that only trusts `pin` (or records the certificate when `pin` is `None`).
pub fn client_config(pin: Option<String>) -> (Arc<ClientConfig>, Arc<PinVerifier>) {
    // reqwest（rustls-no-provider）要求进程内已有默认 provider，与 updater 的做法一致。
    if CryptoProvider::get_default().is_none() {
        let _ = ring::default_provider().install_default();
    }
    let provider = Arc::new(ring::default_provider());
    let verifier = Arc::new(PinVerifier {
        provider: provider.clone(),
        pin,
        seen: Mutex::new(None),
    });
    let config = ClientConfig::builder_with_provider(provider)
        .with_protocol_versions(&[&rustls::version::TLS13])
        .expect("TLS 1.3 is supported by the ring provider")
        .dangerous()
        .with_custom_certificate_verifier(verifier.clone())
        .with_no_client_auth();
    (Arc::new(config), verifier)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fingerprint_is_colon_separated_uppercase_sha256() {
        let fp = fingerprint(b"abc");
        assert_eq!(
            fp,
            "BA:78:16:BF:8F:01:CF:EA:41:41:40:DE:5D:AE:22:23:B0:03:61:A3:96:17:7A:9C:B4:10:FF:61:F2:00:15:AD"
        );
    }

    #[test]
    fn normalize_ignores_case_and_separators() {
        assert_eq!(normalize("ba:78-16 bf"), "BA7816BF");
        assert_eq!(normalize("BA7816BF"), "BA7816BF");
    }
}
