//! 进程内双设备端到端测试：两个 [`SyncState`] 共用一个 Tauri 应用句柄，
//! 经真实 mDNS + TCP 完成配对、文本 / 图片同步与 Windows 虚拟文件读取。
//!
//! 写系统剪贴板与放置 OLE 剪贴板被替换为内存记录（见 `apply::TEST_SINK` /
//! `remote_files::CAPTURED`），因此不依赖桌面剪贴板权限，也不会改动用户剪贴板。
//! 会触碰多播网络，默认忽略；用 `--ignored --test-threads=1` 运行。所有持久化都落到临时目录。

use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, Instant};

use serde_json::json;
use tauri::Manager;

use super::identity::DeviceStore;
use super::manager::SyncState;
use crate::clipboard::{
    build_item, ClipboardPayload, ImagePayload, ImageStore, TextPayload, WritebackGuard,
};
use crate::db::items::find_item_by_content_hash;
use crate::db::models::{ClipboardItem, ClipboardKind};
use crate::db::test_support::memory_pool;
use crate::db::DatabaseState;
use crate::settings::SettingsStore;

struct StderrLogger;

impl log::Log for StderrLogger {
    fn enabled(&self, _: &log::Metadata) -> bool {
        true
    }

    fn log(&self, record: &log::Record) {
        if record.target().contains("eco_paste") {
            eprintln!("[{}] {}", record.level(), record.args());
        }
    }

    fn flush(&self) {}
}

struct Fixture {
    dir: PathBuf,
    // `App` 必须活到测试结束，否则句柄失效。
    _app: tauri::App,
    handle: tauri::AppHandle,
    a: SyncState,
    b: SyncState,
}

impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.dir).ok();
    }
}

/// 收发两端同在一个进程、共用同一个时钟，所以发送端的时间戳要比本机已登记的更新才会被接收端采用。
fn remote_stamp() -> u64 {
    super::apply::stamp_local_copy() + 1
}

fn sink_items() -> Vec<ClipboardItem> {
    super::apply::TEST_SINK
        .lock()
        .unwrap()
        .clone()
        .unwrap_or_default()
}

async fn wait_until<F: FnMut() -> bool>(what: &str, timeout: Duration, mut check: F) {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if check() {
            return;
        }
        tokio::time::sleep(Duration::from_millis(200)).await;
    }
    panic!("timed out waiting for: {what}");
}

async fn fixture() -> Fixture {
    let _ = log::set_logger(&StderrLogger).map(|()| log::set_max_level(log::LevelFilter::Debug));

    let dir = std::env::temp_dir().join(format!("ecopaste-sync-e2e-{}", uuid::Uuid::new_v4()));
    std::fs::create_dir_all(&dir).unwrap();

    let app = tauri::Builder::default()
        .any_thread()
        .build(tauri::generate_context!())
        .expect("build tauri app");
    let handle = app.handle().clone();
    *super::apply::TEST_SINK.lock().unwrap() = Some(Vec::new());
    *super::remote_files::CAPTURED.lock().unwrap() = Some(Vec::new());

    let settings = SettingsStore::for_test(dir.join("settings.json"));
    settings
        .update(json!({ "sync": { "enabled": true } }))
        .unwrap();
    handle.manage(settings);
    handle.manage(ImageStore::for_test(dir.join("images")));
    handle.manage(Arc::new(WritebackGuard::new()));
    handle.manage(DatabaseState::new(memory_pool().await));

    let a = SyncState::new(
        handle.clone(),
        DeviceStore::load_from(dir.join("a.json")).unwrap(),
    );
    let b = SyncState::new(
        handle.clone(),
        DeviceStore::load_from(dir.join("b.json")).unwrap(),
    );
    a.apply_settings().await;
    b.apply_settings().await;

    Fixture {
        dir,
        _app: app,
        handle,
        a,
        b,
    }
}

/// 配对并等待双方建立会话。
async fn pair_and_connect(fx: &Fixture) {
    let code = fx.a.start_pairing().expect("start pairing").code;

    let mut paired = false;
    for _ in 0..40 {
        match fx.b.pair_with_code(&code, None).await {
            Ok(_) => {
                paired = true;
                break;
            }
            Err(_) => tokio::time::sleep(Duration::from_millis(500)).await,
        }
    }
    assert!(paired, "pairing never succeeded");

    wait_until("peers paired on both sides", Duration::from_secs(5), || {
        fx.a.status().peers.len() == 1 && fx.b.status().peers.len() == 1
    })
    .await;
    wait_until("session established", Duration::from_secs(40), || {
        fx.a.status().peers.iter().all(|p| p.online) && fx.b.status().peers.iter().all(|p| p.online)
    })
    .await;
}

#[tokio::test(flavor = "multi_thread")]
#[ignore = "uses mDNS and the local network"]
async fn wrong_code_is_rejected_and_code_burns_out() {
    let fx = fixture().await;
    let code = fx.a.start_pairing().expect("start pairing").code;
    let wrong = if code == "000000" { "000001" } else { "000000" };

    let mut rejected = 0;
    for _ in 0..40 {
        match fx.b.pair_with_code(wrong, None).await {
            Err(super::PairError::CodeInvalid) => {
                rejected += 1;
                if rejected == 3 {
                    break;
                }
            }
            _ => tokio::time::sleep(Duration::from_millis(500)).await,
        }
    }
    assert_eq!(rejected, 3, "expected three wrong-code rejections");

    // 三次失败后真码也不再可用。
    assert!(fx.b.pair_with_code(&code, None).await.is_err());
    assert!(fx.a.status().pairing.is_none());
    assert!(fx.a.status().peers.is_empty());
    assert!(fx.b.status().peers.is_empty());
}

#[tokio::test(flavor = "multi_thread")]
#[ignore = "uses mDNS and the local network"]
async fn manual_address_pairs_and_remembers_peer() {
    let fx = fixture().await;
    let port = fx.a.listen_port().expect("a is listening");
    let code = fx.a.start_pairing().expect("start pairing").code;

    assert!(matches!(
        fx.b.pair_with_code(&code, Some("not-an-ip")).await,
        Err(super::PairError::AddressInvalid)
    ));
    assert!(matches!(
        fx.b.pair_with_code(&code, Some("127.0.0.1:1")).await,
        Err(super::PairError::Unreachable)
    ));

    let host = format!("127.0.0.1:{port}");
    fx.b.pair_with_code(&code, Some(&host))
        .await
        .expect("manual pairing");

    let a_id = fx.a.status().device_id;
    let stored = super::identity::DeviceStore::load_from(fx.dir.join("b.json"))
        .unwrap()
        .get(&a_id)
        .and_then(|d| d.addr);
    assert_eq!(stored.as_deref(), Some(host.as_str()));

    wait_until("session established", Duration::from_secs(40), || {
        fx.a.status().peers.iter().all(|p| p.online)
            && fx.b.status().peers.iter().all(|p| p.online)
            && fx.a.status().peers.len() == 1
    })
    .await;
}

#[tokio::test(flavor = "multi_thread")]
#[ignore = "uses mDNS and the local network"]
async fn pair_then_sync_text_and_image() {
    let fx = fixture().await;
    pair_and_connect(&fx).await;

    let store = fx.handle.state::<ImageStore>().inner().clone();

    // 文本
    let marker = format!("sync-e2e-{}", uuid::Uuid::new_v4());
    let item = build_item(
        &store,
        &ClipboardPayload::Text(TextPayload {
            text: marker.clone(),
            html: Some("<b>rich</b>".to_owned()),
            rtf: None,
        }),
    )
    .unwrap()
    .unwrap();
    fx.a.broadcast(item.clone(), remote_stamp());

    wait_until(
        "text is applied on the peer",
        Duration::from_secs(10),
        || {
            sink_items()
                .iter()
                .any(|i| i.kind == ClipboardKind::Text && i.search_text.as_deref() == Some(&marker))
        },
    )
    .await;
    let pool = fx.handle.state::<DatabaseState>().pool().await;
    let start = Instant::now();
    loop {
        let found = find_item_by_content_hash(&pool, &item.content_hash)
            .await
            .unwrap();
        if found.is_some() {
            break;
        }
        assert!(
            start.elapsed() < Duration::from_secs(5),
            "text not persisted"
        );
        tokio::time::sleep(Duration::from_millis(200)).await;
    }

    // 图片（大于一个分块，验证分块重组）
    let (width, height) = (700u32, 500u32);
    let mut pixels = image::RgbaImage::new(width, height);
    for (x, y, px) in pixels.enumerate_pixels_mut() {
        *px = image::Rgba([(x % 251) as u8, (y % 241) as u8, ((x * y) % 239) as u8, 255]);
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
    fx.a.broadcast(image_item, remote_stamp());

    wait_until(
        "image is applied on the peer",
        Duration::from_secs(15),
        || {
            sink_items().iter().any(|i| {
                i.kind == ClipboardKind::Image
                    && i.width == Some(i64::from(width))
                    && i.height == Some(i64::from(height))
            })
        },
    )
    .await;
}

#[cfg(target_os = "windows")]
#[tokio::test(flavor = "multi_thread")]
#[ignore = "uses mDNS and the local network"]
async fn files_are_pulled_lazily_through_virtual_data_object() {
    let fx = fixture().await;
    pair_and_connect(&fx).await;

    let store = fx.handle.state::<ImageStore>().inner().clone();

    // 3.5 MiB 的伪随机内容，跨多个传输块；外加一个带子目录的文件夹。
    let src = fx.dir.join("src");
    let folder = src.join("album");
    std::fs::create_dir_all(folder.join("inner")).unwrap();
    let big: Vec<u8> = (0..3_670_016u32)
        .map(|i| (i.wrapping_mul(2_654_435_761) >> 13) as u8)
        .collect();
    std::fs::write(src.join("big.bin"), &big).unwrap();
    std::fs::write(folder.join("inner").join("note.txt"), b"hello folder").unwrap();

    let item = build_item(
        &store,
        &ClipboardPayload::Files(vec![
            src.join("big.bin").to_string_lossy().into_owned(),
            folder.to_string_lossy().into_owned(),
        ]),
    )
    .unwrap()
    .unwrap();
    fx.a.broadcast(item, remote_stamp());

    wait_until("file offer arrives", Duration::from_secs(10), || {
        super::remote_files::CAPTURED
            .lock()
            .unwrap()
            .as_ref()
            .is_some_and(|c| !c.is_empty())
    })
    .await;
    let captured = super::remote_files::CAPTURED
        .lock()
        .unwrap()
        .as_mut()
        .and_then(|c| c.pop())
        .expect("captured offer");

    let copies: Vec<_> = (0..4)
        .map(|i| {
            let path = src.join(format!("later-{i}.txt"));
            std::fs::write(&path, b"another copy").unwrap();
            build_item(
                &store,
                &ClipboardPayload::Files(vec![path.to_string_lossy().into_owned()]),
            )
            .unwrap()
            .unwrap()
        })
        .collect();

    // 不经系统剪贴板，直接像资源管理器一样读取数据对象；
    // 在运行时之外的线程上读，`Read` 内部会 block_on 经网络拉取。
    let sender = fx.a.clone();
    let reader = std::thread::spawn(move || {
        windows_reader::read_virtual_files(captured, big, || {
            tauri::async_runtime::block_on(async {
                for copy in copies {
                    sender.prepare_files(&copy).await.unwrap();
                }
            });
        })
    });
    tokio::task::spawn_blocking(move || reader.join().expect("reader thread"))
        .await
        .unwrap()
        .expect("virtual file paste");
}

#[cfg(target_os = "windows")]
mod windows_reader {
    use windows::core::PCWSTR;
    use windows::Win32::System::Com::{
        IDataObject, IStream, DVASPECT_CONTENT, FORMATETC, TYMED_HGLOBAL, TYMED_ISTREAM,
    };
    use windows::Win32::System::DataExchange::RegisterClipboardFormatW;
    use windows::Win32::System::Memory::{GlobalLock, GlobalUnlock};
    use windows::Win32::System::Ole::ReleaseStgMedium;
    use windows::Win32::UI::Shell::FILEDESCRIPTORW;

    use crate::sync::remote_files::Captured;

    fn format(name: &str, lindex: i32, tymed: i32) -> FORMATETC {
        let wide: Vec<u16> = name.encode_utf16().chain(std::iter::once(0)).collect();
        FORMATETC {
            cfFormat: unsafe { RegisterClipboardFormatW(PCWSTR(wide.as_ptr())) } as u16,
            ptd: std::ptr::null_mut(),
            dwAspect: DVASPECT_CONTENT.0,
            lindex,
            tymed: tymed as u32,
        }
    }

    fn descriptor_names(data: &IDataObject) -> Option<Vec<String>> {
        let fmt = format("FileGroupDescriptorW", -1, TYMED_HGLOBAL.0);
        let mut medium = unsafe { data.GetData(&fmt) }.ok()?;
        let handle = unsafe { medium.u.hGlobal };
        let base = unsafe { GlobalLock(handle) } as *const u8;
        if base.is_null() {
            unsafe { ReleaseStgMedium(&mut medium) };
            return None;
        }

        let count = unsafe { std::ptr::read_unaligned(base as *const u32) } as usize;
        let item = std::mem::size_of::<FILEDESCRIPTORW>();
        let mut names = Vec::new();
        for i in 0..count {
            let descriptor: FILEDESCRIPTORW = unsafe {
                std::ptr::read_unaligned(base.add(4 + i * item) as *const FILEDESCRIPTORW)
            };
            let name = descriptor.cFileName;
            let end = name.iter().position(|c| *c == 0).unwrap_or(name.len());
            names.push(String::from_utf16_lossy(&name[..end]));
        }
        unsafe {
            let _ = GlobalUnlock(handle);
            ReleaseStgMedium(&mut medium);
        }
        Some(names)
    }

    fn open_stream(data: &IDataObject, index: i32) -> Result<IStream, String> {
        let fmt = format("FileContents", index, TYMED_ISTREAM.0);
        let mut medium = unsafe { data.GetData(&fmt) }.map_err(|e| format!("GetData: {e}"))?;
        let stream = unsafe { (*medium.u.pstm).clone() };
        unsafe { ReleaseStgMedium(&mut medium) };
        stream.ok_or_else(|| "no stream".to_owned())
    }

    fn read_chunk(stream: &IStream, buf: &mut [u8]) -> Result<usize, String> {
        let mut read = 0u32;
        unsafe { stream.Read(buf.as_mut_ptr().cast(), buf.len() as u32, Some(&mut read)) }
            .ok()
            .map_err(|e| format!("Read: {e}"))?;
        Ok(read as usize)
    }

    fn read_rest(stream: &IStream, out: &mut Vec<u8>) -> Result<(), String> {
        let mut buf = vec![0u8; 256 * 1024];
        loop {
            let read = read_chunk(stream, &mut buf)?;
            if read == 0 {
                break;
            }
            out.extend_from_slice(&buf[..read]);
        }
        Ok(())
    }

    pub fn read_virtual_files(
        captured: Captured,
        expected_big: Vec<u8>,
        copy_more_files: impl FnOnce(),
    ) -> Result<(), String> {
        let (data, is_in_use) = crate::sync::remote_files::create_data_object(
            captured.state,
            captured.peer_id,
            captured.offer,
            captured.entries,
        );
        let names = descriptor_names(&data).ok_or("descriptor unavailable")?;

        let find = |name: &str| names.iter().position(|n| n == name);
        let big_index = find("big.bin").ok_or(format!("big.bin missing in {names:?}"))?;
        let note_index =
            find("album\\inner\\note.txt").ok_or(format!("note.txt missing in {names:?}"))?;
        find("album").ok_or("album dir missing")?;

        let first = open_stream(&data, big_index as i32)?;
        let mut first_bytes = vec![0; 256 * 1024];
        let read = read_chunk(&first, &mut first_bytes)?;
        assert!(read > 0);
        first_bytes.truncate(read);

        copy_more_files();
        let second = open_stream(&data, big_index as i32)?;
        let note = open_stream(&data, note_index as i32)?;
        drop(data);
        assert!(is_in_use());

        let mut second_bytes = vec![0; 128 * 1024];
        let read = read_chunk(&second, &mut second_bytes)?;
        assert!(read > 0);
        second_bytes.truncate(read);
        read_rest(&first, &mut first_bytes)?;
        read_rest(&second, &mut second_bytes)?;
        assert_eq!(blake3::hash(&first_bytes), blake3::hash(&expected_big));
        assert_eq!(blake3::hash(&second_bytes), blake3::hash(&expected_big));

        let mut note_bytes = Vec::new();
        read_rest(&note, &mut note_bytes)?;
        assert_eq!(note_bytes, b"hello folder");
        drop((first, second, note));
        assert!(!is_in_use());
        Ok(())
    }
}
