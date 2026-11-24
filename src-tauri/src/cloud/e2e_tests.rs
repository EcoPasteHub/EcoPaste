//! 与真实 EcoPasteServer 进程的端到端测试：证书指纹固定、登录、实时同步文本 / 图片、拉取历史。
//!
//! 需要服务端可执行文件：环境变量 `ECOPASTE_SERVER_EXE`，缺省为同级仓库
//! `../EcoPasteServer/desktop/src-tauri/ecopaste-server.exe`。找不到时测试直接跳过。
//! 写剪贴板被替换为内存记录（`sync::apply::TEST_SINK`），不触碰系统剪贴板。

use std::net::TcpStream;
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::Arc;
use std::time::{Duration, Instant};

use serde_json::json;
use tauri::Manager;

use super::{CloudError, CloudState, LinkState};
use crate::clipboard::{
    build_item, ClipboardPayload, ImagePayload, ImageStore, TextPayload, WritebackGuard,
};
use crate::db::models::ClipboardKind;
use crate::db::test_support::memory_pool;
use crate::db::DatabaseState;
use crate::settings::SettingsStore;

fn server_exe() -> Option<PathBuf> {
    let path = std::env::var("ECOPASTE_SERVER_EXE")
        .map(PathBuf::from)
        .unwrap_or_else(|_| {
            PathBuf::from(
                r"D:\Projects\claude\EcoPasteServer\desktop\src-tauri\ecopaste-server.exe",
            )
        });
    path.exists().then_some(path)
}

struct Server {
    child: Child,
    dir: PathBuf,
    port: u16,
    admin_port: u16,
}

impl Drop for Server {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
        std::fs::remove_dir_all(&self.dir).ok();
    }
}

fn free_port() -> u16 {
    std::net::TcpListener::bind("127.0.0.1:0")
        .unwrap()
        .local_addr()
        .unwrap()
        .port()
}

fn start_server(exe: &PathBuf) -> Server {
    let dir = std::env::temp_dir().join(format!("ecopaste-cloud-e2e-{}", uuid::Uuid::new_v4()));
    std::fs::create_dir_all(&dir).unwrap();
    let (port, admin_port) = (free_port(), free_port());
    std::fs::write(
        dir.join("config.json"),
        json!({ "port": port, "adminPort": admin_port }).to_string(),
    )
    .unwrap();

    let child = Command::new(exe)
        .arg("-data")
        .arg(&dir)
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .expect("spawn server");

    let start = Instant::now();
    while TcpStream::connect(("127.0.0.1", port)).is_err() {
        assert!(
            start.elapsed() < Duration::from_secs(10),
            "server did not start"
        );
        std::thread::sleep(Duration::from_millis(100));
    }
    Server {
        child,
        dir,
        port,
        admin_port,
    }
}

async fn create_user(server: &Server, name: &str, pass: &str) {
    let token = std::fs::read_to_string(server.dir.join("admin.token")).unwrap();
    let resp = reqwest::Client::new()
        .post(format!(
            "http://127.0.0.1:{}/admin/users",
            server.admin_port
        ))
        .header("X-Admin-Token", token.trim())
        .json(&json!({ "username": name, "password": pass }))
        .send()
        .await
        .unwrap();
    assert!(resp.status().is_success(), "create user: {}", resp.status());
}

async fn wait_until<F: FnMut() -> bool>(what: &str, timeout: Duration, mut check: F) {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if check() {
            return;
        }
        tokio::time::sleep(Duration::from_millis(100)).await;
    }
    panic!("timed out waiting for: {what}");
}

fn sink_items() -> Vec<crate::db::models::ClipboardItem> {
    crate::sync::apply_sink_items()
}

#[tokio::test(flavor = "multi_thread")]
#[ignore = "spawns the real EcoPasteServer binary"]
async fn cloud_sync_against_real_server() {
    let Some(exe) = server_exe() else {
        eprintln!("skipped: ecopaste-server.exe not found");
        return;
    };
    let _ = rustls::crypto::ring::default_provider().install_default();
    let server = start_server(&exe);
    create_user(&server, "alice", "correct-horse").await;

    let dir = std::env::temp_dir().join(format!("ecopaste-cloud-client-{}", uuid::Uuid::new_v4()));
    std::fs::create_dir_all(&dir).unwrap();
    let app = tauri::Builder::default()
        .any_thread()
        .build(tauri::generate_context!())
        .expect("build tauri app");
    let handle = app.handle().clone();
    crate::sync::apply_sink_enable();

    let settings = SettingsStore::for_test(dir.join("settings.json"));
    settings
        .update(json!({ "cloud": { "enabled": true } }))
        .unwrap();
    handle.manage(settings);
    handle.manage(ImageStore::for_test(dir.join("images")));
    handle.manage(Arc::new(WritebackGuard::new()));
    handle.manage(DatabaseState::new(memory_pool().await));

    let a = CloudState::build(&handle, dir.join("a.json"), Default::default());
    let b = CloudState::build(&handle, dir.join("b.json"), Default::default());
    let url = format!("127.0.0.1:{}", server.port);

    // 探测指纹；错误指纹的登录被拒绝；正确指纹登录成功。
    let probe = a.probe(&url).await.expect("probe");
    assert_eq!(probe.fingerprint.split(':').count(), 32);
    let wrong = "00:".repeat(31) + "00";
    let err = a
        .login(&url, &wrong, "alice", "correct-horse")
        .await
        .expect_err("wrong fingerprint must be rejected");
    assert!(matches!(
        err.downcast_ref::<CloudError>(),
        Some(CloudError::PinMismatch)
    ));
    let err = a
        .login(&url, &probe.fingerprint, "alice", "wrong-password")
        .await
        .expect_err("bad password");
    assert!(matches!(
        err.downcast_ref::<CloudError>(),
        Some(CloudError::BadCredentials)
    ));

    a.login(&url, &probe.fingerprint, "alice", "correct-horse")
        .await
        .expect("login a");
    b.login(&url, &probe.fingerprint, "alice", "correct-horse")
        .await
        .expect("login b");
    tauri::async_runtime::spawn(super::run(a.0.clone()));
    tauri::async_runtime::spawn(super::run(b.0.clone()));

    wait_until("both connected", Duration::from_secs(10), || {
        a.status().state == LinkState::Connected && b.status().state == LinkState::Connected
    })
    .await;

    let store = handle.state::<ImageStore>().inner().clone();

    // 文本：A 上传 -> B 实时应用。
    let marker = format!("cloud-e2e-{}", uuid::Uuid::new_v4());
    let text_item = build_item(
        &store,
        &ClipboardPayload::Text(TextPayload {
            text: marker.clone(),
            html: Some("<b>rich</b>".into()),
            rtf: None,
        }),
    )
    .unwrap()
    .unwrap();
    super::upload_item(&a.0, &text_item)
        .await
        .expect("upload text");
    wait_until("text reaches B", Duration::from_secs(10), || {
        sink_items()
            .iter()
            .any(|i| i.kind == ClipboardKind::Text && i.search_text.as_deref() == Some(&marker))
    })
    .await;

    // 图片：经 blob 上传 / 下载，尺寸一致。
    let (width, height) = (320u32, 200u32);
    let mut pixels = image::RgbaImage::new(width, height);
    for (x, y, px) in pixels.enumerate_pixels_mut() {
        *px = image::Rgba([(x % 255) as u8, (y % 253) as u8, ((x ^ y) % 251) as u8, 255]);
    }
    let mut png = std::io::Cursor::new(Vec::new());
    image::DynamicImage::ImageRgba8(pixels)
        .write_to(&mut png, image::ImageFormat::Png)
        .unwrap();
    let image_item = build_item(
        &store,
        &ClipboardPayload::Image(ImagePayload {
            bytes: png.into_inner(),
            width,
            height,
        }),
    )
    .unwrap()
    .unwrap();
    super::upload_item(&a.0, &image_item)
        .await
        .expect("upload image");
    wait_until("image reaches B", Duration::from_secs(10), || {
        sink_items().iter().any(|i| {
            i.kind == ClipboardKind::Image
                && i.width == Some(i64::from(width))
                && i.height == Some(i64::from(height))
        })
    })
    .await;

    // 新设备拉取近期历史（不写剪贴板，只入历史，可重复使用已见哈希）。
    let c = CloudState::build(&handle, dir.join("c.json"), Default::default());
    c.login(&url, &probe.fingerprint, "alice", "correct-horse")
        .await
        .expect("login c");
    let pulled = c.pull_recent().await.expect("pull recent");
    assert_eq!(pulled, 2, "history should contain the text and the image");

    // 登出后令牌失效。
    a.logout().await;
    assert!(!a.status().logged_in);
    assert!(a.0.api().is_err());
    assert!(b.status().logged_in, "other devices stay signed in");
}
