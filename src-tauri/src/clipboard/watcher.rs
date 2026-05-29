//! OS 级剪贴板监听：把 [`clipboard_rs`] 的 watcher 接到「读取 → 去重入库 → emit」闭环。
//!
//! [`clipboard_rs`] 内部已实现 macOS（`NSPasteboard.changeCount` 轮询）/ Windows
//! （`AddClipboardFormatListener` → `WM_CLIPBOARDUPDATE`）的平台监听，这里不重复造。
//!
//! 线程模型：`ClipboardWatcherContext::start_watch()` 是阻塞调用，故整个监听跑在独立
//! `std::thread` 上。`ClipboardContext` 等平台句柄**在该线程内构造**，不跨线程移动，
//! 从而绕开其 `Send` 约束；只有 `Send` 的数据（连接池克隆、`AppHandle`、`item`）会被
//! 投递进 Tauri 异步运行时做 sqlx 入库与事件 emit。

use std::sync::Arc;

use clipboard_rs::{ClipboardHandler, ClipboardWatcher, ClipboardWatcherContext};
use serde_json::json;
use sqlx::SqlitePool;
use tauri::{AppHandle, Manager};
use tauri_awesome_rpc::EmitterExt;

use super::guard::WritebackGuard;
use super::ingest::build_item;
use super::read::ClipboardReader;
use super::storage::ImageStore;
use crate::db::items::{upsert_item, UpsertResult};
use crate::db::models::ClipboardItem;

/// 剪贴板更新事件名。前端监听此事件后增量刷新 / 重新拉取列表（阶段 7.2）。
pub const CLIPBOARD_UPDATED_EVENT: &str = "clipboard://updated";

/// 去重入库 + emit「剪贴板更新」事件。监听回调与 `read_clipboard` 命令共用，
/// 保证两条路径的入库语义与事件契约一致。失败仅记日志（监听场景无人接收 Result）。
pub async fn persist_and_notify(
    app: &AppHandle,
    pool: &SqlitePool,
    item: &ClipboardItem,
) -> crate::core::Result<UpsertResult> {
    let result = upsert_item(pool, item).await?;
    // EmitterExt::emit 是 fire-and-forget（返回 ()），经 WS 通道送达前端。
    app.emit(
        CLIPBOARD_UPDATED_EVENT,
        json!({
            "id": result.id,
            "deduplicated": result.deduplicated,
        }),
    );
    Ok(result)
}

/// 启动监听：注册 [`WritebackGuard`] 与 [`ImageStore`] 到 Tauri `State`
/// （供阶段 4 写回打标记 / 取图），并在独立线程上跑 OS 级监听。
/// 应在 `setup` 中、连接池就绪后调用一次。`ImageStore` 创建失败属致命配置错误，直接返回错误。
pub fn init(app: &AppHandle, pool: SqlitePool) -> crate::core::Result<()> {
    let guard = Arc::new(WritebackGuard::new());
    app.manage(guard.clone());

    let store = ImageStore::new(app)?;
    app.manage(store.clone());

    spawn_watch_thread(app.clone(), pool, guard, store);
    Ok(())
}

fn spawn_watch_thread(
    app: AppHandle,
    pool: SqlitePool,
    guard: Arc<WritebackGuard>,
    store: ImageStore,
) {
    std::thread::Builder::new()
        .name("clipboard-watcher".to_owned())
        .spawn(move || {
            // 平台剪贴板句柄在本线程内构造，不跨线程移动。
            let reader = match ClipboardReader::new() {
                Ok(reader) => reader,
                Err(err) => {
                    log::error!("clipboard watcher: failed to create reader: {err}");
                    return;
                }
            };

            let mut watcher = match ClipboardWatcherContext::new() {
                Ok(watcher) => watcher,
                Err(err) => {
                    log::error!("clipboard watcher: failed to create watcher: {err}");
                    return;
                }
            };

            watcher.add_handler(ClipboardChangeHandler {
                reader,
                pool,
                app,
                guard,
                store,
            });

            log::info!("clipboard watcher started");
            // 阻塞直至进程退出。
            watcher.start_watch();
        })
        .expect("failed to spawn clipboard watcher thread");
}

struct ClipboardChangeHandler {
    reader: ClipboardReader,
    pool: SqlitePool,
    app: AppHandle,
    guard: Arc<WritebackGuard>,
    store: ImageStore,
}

impl ClipboardHandler for ClipboardChangeHandler {
    fn on_clipboard_change(&mut self) {
        // 同步读取 + 转换（含图片落盘）：拿到 content_hash 才能判定是否为自身写回。
        let payload = match self.reader.read_all() {
            Ok(Some(payload)) => payload,
            Ok(None) => return,
            Err(err) => {
                log::warn!("clipboard watcher: read failed: {err}");
                return;
            }
        };

        let item = match build_item(&self.store, &payload) {
            Ok(Some(item)) => item,
            Ok(None) => return,
            Err(err) => {
                log::warn!("clipboard watcher: build item failed: {err}");
                return;
            }
        };

        // 自身写回触发的变更：跳过入库，避免回环。
        if self.guard.should_skip(&item.content_hash) {
            return;
        }

        // 入库与 emit 交给异步运行时；只移动 Send 数据，不碰平台句柄。
        let pool = self.pool.clone();
        let app = self.app.clone();
        tauri::async_runtime::spawn(async move {
            if let Err(err) = persist_and_notify(&app, &pool, &item).await {
                log::error!("clipboard watcher: persist failed: {err}");
            }
        });
    }
}

#[cfg(test)]
mod tests {
    use clipboard_rs::{Clipboard, ClipboardContext};

    use super::*;
    use crate::clipboard::{ImageStore, WritebackGuard};
    use crate::db::items::find_item_by_id;
    use crate::db::test_support::memory_pool;

    fn temp_image_store() -> (TempDir, ImageStore) {
        let dir = TempDir::new();
        let store = ImageStore::for_test(dir.path().join("resources").join("images"));
        (dir, store)
    }

    // 复刻 on_clipboard_change 的同步部分（读取 → 转换 → 去重判定）+ async 入库，
    // 但绕开 Tauri AppHandle / emit（无法在单测里构造），验证整条数据链路。
    // 触碰真实系统剪贴板，默认 ignore；本机用 `cargo test -- --ignored` 验证。
    #[tokio::test(flavor = "multi_thread")]
    #[ignore = "touches the real system clipboard; run with --ignored on a desktop session"]
    async fn end_to_end_text_ingests_once_then_dedups() {
        let pool = memory_pool().await;
        let guard = WritebackGuard::new();
        let (_dir, store) = temp_image_store();

        // 串行锁只覆盖触碰真实剪贴板的同步段，await DB 前即释放（不跨 await 持锁）。
        let item = {
            let _serial = crate::clipboard::test_lock::serial();
            let ctx = ClipboardContext::new().unwrap();
            ctx.set_text("e2e ecopaste watcher".to_owned()).unwrap();

            let reader = ClipboardReader::new().unwrap();
            let payload = reader.read_all().unwrap().expect("should read text");
            build_item(&store, &payload)
                .unwrap()
                .expect("should map to item")
        };
        assert!(!guard.should_skip(&item.content_hash));

        // 首次入库：新行。
        let first = upsert_item(&pool, &item).await.unwrap();
        assert!(!first.deduplicated);
        assert_eq!(
            find_item_by_id(&pool, &first.id)
                .await
                .unwrap()
                .unwrap()
                .content,
            "e2e ecopaste watcher"
        );

        // 同内容再来一次：命中去重，use_count 累加，不新增行。
        let second = upsert_item(&pool, &item).await.unwrap();
        assert!(second.deduplicated);
        assert_eq!(first.id, second.id);
        assert_eq!(
            find_item_by_id(&pool, &first.id)
                .await
                .unwrap()
                .unwrap()
                .use_count,
            2
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    #[ignore = "touches the real system clipboard; run with --ignored on a desktop session"]
    async fn writeback_guard_suppresses_self_copy() {
        let (_dir, store) = temp_image_store();
        let _serial = crate::clipboard::test_lock::serial();
        let ctx = ClipboardContext::new().unwrap();
        ctx.set_text("self writeback content".to_owned()).unwrap();

        let reader = ClipboardReader::new().unwrap();
        let item = build_item(&store, &reader.read_all().unwrap().unwrap())
            .unwrap()
            .unwrap();

        // 模拟写回前登记 → 监听读到同内容 → 被抑制。
        let guard = WritebackGuard::new();
        guard.suppress(item.content_hash.clone());
        assert!(guard.should_skip(&item.content_hash));
    }

    // 验证真实剪贴板图片链路：set_image（OS 原生 TIFF）→ read_all 解码为 PNG →
    // build_item 落盘原图/缩略图 → upsert 入库。覆盖合成 PNG 测不到的 OS 解码段。
    #[tokio::test(flavor = "multi_thread")]
    #[ignore = "touches the real system clipboard; run with --ignored on a desktop session"]
    async fn end_to_end_image_stores_and_ingests() {
        use clipboard_rs::common::RustImage;

        let pool = memory_pool().await;
        let (_dir, store) = temp_image_store();

        let item = {
            let _serial = crate::clipboard::test_lock::serial();
            let png = {
                use std::io::Cursor;
                let buf = image::RgbaImage::from_pixel(40, 24, image::Rgba([7, 8, 9, 255]));
                let mut out = Cursor::new(Vec::new());
                image::DynamicImage::ImageRgba8(buf)
                    .write_to(&mut out, image::ImageFormat::Png)
                    .unwrap();
                out.into_inner()
            };
            let ctx = ClipboardContext::new().unwrap();
            ctx.set_image(clipboard_rs::RustImageData::from_bytes(&png).unwrap())
                .unwrap();

            let reader = ClipboardReader::new().unwrap();
            let payload = reader.read_all().unwrap().expect("should read image");
            build_item(&store, &payload)
                .unwrap()
                .expect("image should map to item")
        };

        assert_eq!(item.kind, crate::db::ClipboardKind::Image);
        assert!(item.content.ends_with(".png"));
        assert!(item.width.unwrap() > 0 && item.height.unwrap() > 0);
        assert!(store.origin_path(&item.content).exists());
        assert!(store.thumbnail_path(&item.content).exists());

        let result = upsert_item(&pool, &item).await.unwrap();
        assert!(!result.deduplicated);
        assert_eq!(
            find_item_by_id(&pool, &result.id)
                .await
                .unwrap()
                .unwrap()
                .kind,
            crate::db::ClipboardKind::Image
        );
    }

    struct TempDir(std::path::PathBuf);
    impl TempDir {
        fn new() -> Self {
            let p = std::env::temp_dir().join(format!("ecopaste-watcher-{}", uuid::Uuid::new_v4()));
            std::fs::create_dir_all(&p).unwrap();
            Self(p)
        }
        fn path(&self) -> &std::path::Path {
            &self.0
        }
    }
    impl Drop for TempDir {
        fn drop(&mut self) {
            std::fs::remove_dir_all(&self.0).ok();
        }
    }
}
