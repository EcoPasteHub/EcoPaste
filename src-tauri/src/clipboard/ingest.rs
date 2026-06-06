//! 把读取到的 [`ClipboardPayload`] 转换为可入库的 [`ClipboardItem`]：
//! 编排子类型识别（[`super::detect`]）与图片落盘（[`super::storage`]）。
//!
//! 文本存储语义（对齐旧项目读取优先级 html > rtf > plain）：
//! - `content` 存**源表示**（HTML/RTF 原文、或纯文本），供前端渲染与写回；
//! - `search_text` 存**纯文本**（HTML 去标签 / RTF 用 OS 提供的 plain 文本），供 FTS 检索与纯文本粘贴；
//! - 纯文本无富文本时 `sub_kind` 走 url/email/color/path 识别。
//!
//! 图片：落盘原图 + 缩略图，`content` 存文件名 `<sha256>.png`，`content_hash` 仍走
//! [`content_hash(Image, file_name)`]，而 `file_name` 源自 PNG 字节哈希 → 去重对字节敏感。

use chrono::Utc;

use super::detect::detect_text_sub_kind;
use super::payload::{ClipboardPayload, TextPayload};
use super::secrets::contains_secret;
use super::storage::ImageStore;
use crate::core::Result;
use crate::db::items::content_hash;
use crate::db::models::{ClipboardItem, ClipboardKind, ClipboardSubKind, Platform};
use crate::settings::{Capture, Sensitive};

/// 列表渲染用摘要的最大字符数（按 Unicode 标量计，不是字节）。
/// 超过此长度的文本会被截断，前端列表只渲染摘要，预览/写回时再读完整 `content`。
pub const SUMMARY_MAX_CHARS: usize = 256;

/// 从纯文本生成列表摘要：trim 后按 [`SUMMARY_MAX_CHARS`] 字符截断。
/// 输入空串返回 `None`。HTML/RTF 也用这个，输入是 OS 同时提供的纯文本，不解析富文本。
fn make_summary(plain: &str) -> Option<String> {
    let trimmed = plain.trim();
    if trimmed.is_empty() {
        return None;
    }
    Some(trimmed.chars().take(SUMMARY_MAX_CHARS).collect())
}

/// 当前平台标记。仅 macOS / Windows 双平台（见 AGENTS.md），其余 target 不应被编译进来。
fn current_platform() -> Platform {
    #[cfg(target_os = "macos")]
    {
        Platform::Macos
    }
    #[cfg(target_os = "windows")]
    {
        Platform::Windows
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        compile_error!("EcoPaste only supports macOS and Windows")
    }
}

/// 转换中间产物：决定一条记录的核心字段，其余字段由 [`build_item`] 补齐。
struct Draft {
    kind: ClipboardKind,
    sub_kind: Option<ClipboardSubKind>,
    content: String,
    search_text: Option<String>,
    summary: Option<String>,
    file_types: Option<String>,
    width: Option<i64>,
    height: Option<i64>,
    size: Option<i64>,
}

/// files 列表入库时的 `content`：换行连接的路径串（与去重哈希、FTS 检索单一来源）。
fn files_to_content(files: &[String]) -> String {
    files.join("\n")
}

/// 文本载荷 → 草稿。优先级 html > rtf > plain（对齐参考插件 tauri-plugin-clipboard-x）：
/// - html：`content` = HTML 源（`get_html()` 原文，前端 DOMPurify 后渲染），`sub_kind` = Html；
/// - rtf：`content` = RTF 源（`get_rich_text()` 原文），`sub_kind` = Rtf；
/// - 两者的 `search_text` 都直接用 OS 同时提供的纯文本（`get_text()`）——
///   复制富文本时剪贴板本就并存纯文本表示，无需自己解析 HTML/RTF；
/// - 富文本存在但对应采集开关关闭时，整条文本载荷跳过，不降级保存为纯文本；
/// - plain：仅在无富文本源时处理，`content` = 纯文本，`sub_kind` = url/email/color/path 识别，
///   `search_text` 与 content 同串（统一由 FTS 索引 search_text）。
///
/// 一律以 trim 后的纯文本作为「是否有可展示内容」的判据：纯文本为空就直接 `None`，
/// 不管 HTML/RTF 源是否存在（只有样式/空白节点的源对用户没意义，列表也渲染不出来）。
fn draft_from_text(text: &TextPayload, capture: &Capture, plain_only: bool) -> Option<Draft> {
    if !capture.text && !capture.html && !capture.rtf {
        return None;
    }

    let plain = text.text.trim();
    if plain.is_empty() {
        return None;
    }

    let plain_size = Some(count_text_chars(plain));
    let plain_search = Some(plain.to_owned());
    let summary = make_summary(plain);

    let html = non_empty(&text.html);
    let rtf = non_empty(&text.rtf);

    if !plain_only {
        if let Some(html) = html {
            if capture.html {
                return Some(Draft {
                    kind: ClipboardKind::Text,
                    sub_kind: Some(ClipboardSubKind::Html),
                    content: html.clone(),
                    search_text: plain_search.clone(),
                    summary: summary.clone(),
                    file_types: None,
                    width: None,
                    height: None,
                    size: plain_size,
                });
            }
        }

        if let Some(rtf) = rtf {
            if capture.rtf {
                return Some(Draft {
                    kind: ClipboardKind::Text,
                    sub_kind: Some(ClipboardSubKind::Rtf),
                    content: rtf.clone(),
                    search_text: plain_search.clone(),
                    summary: summary.clone(),
                    file_types: None,
                    width: None,
                    height: None,
                    size: plain_size,
                });
            }
        }

        if html.is_some() || rtf.is_some() {
            return None;
        }
    }

    if !capture.text {
        return None;
    }

    Some(Draft {
        kind: ClipboardKind::Text,
        sub_kind: detect_text_sub_kind(plain),
        content: plain.to_owned(),
        search_text: plain_search,
        summary,
        file_types: None,
        width: None,
        height: None,
        size: plain_size,
    })
}

fn non_empty(value: &Option<String>) -> Option<&String> {
    value.as_ref().filter(|s| !s.trim().is_empty())
}

/// 文本 size 统一按纯文本 Unicode 标量数量计算，不使用 HTML/RTF 源码长度。
fn count_text_chars(text: &str) -> i64 {
    text.chars().count() as i64
}

/// 使用默认采集开关把载荷转换为待入库记录。
#[cfg(test)]
pub fn build_item(store: &ImageStore, payload: &ClipboardPayload) -> Result<Option<ClipboardItem>> {
    build_item_with_settings(
        store,
        payload,
        &Capture::default(),
        &Sensitive::default(),
        false,
    )
}

/// 把载荷转换为待入库记录，同时应用内容类型与隐私过滤设置。
pub fn build_item_with_settings(
    store: &ImageStore,
    payload: &ClipboardPayload,
    capture: &Capture,
    sensitive: &Sensitive,
    plain_only: bool,
) -> Result<Option<ClipboardItem>> {
    let draft = match payload {
        ClipboardPayload::Text(text) => {
            if sensitive.secret_detection && contains_secret(&text.text) {
                return Ok(None);
            }

            draft_from_text(text, capture, plain_only)
        }
        ClipboardPayload::Files(files) => {
            if !capture.files {
                return Ok(None);
            }

            let content = files_to_content(files);
            if content.trim().is_empty() {
                None
            } else {
                // 记录每个路径的类型：d=directory, f=file
                let file_types: Vec<&str> = files
                    .iter()
                    .map(|p| {
                        if std::path::Path::new(p).is_dir() {
                            "d"
                        } else {
                            "f"
                        }
                    })
                    .collect();
                let file_types_str = file_types.join(",");

                // 文件名写入 search_text 供 FTS 命中；列表名由命令层从 content 路径现算，summary 保持为空。
                let basenames = files
                    .iter()
                    .map(|p| {
                        std::path::Path::new(p)
                            .file_name()
                            .and_then(|n| n.to_str())
                            .unwrap_or(p.as_str())
                    })
                    .collect::<Vec<_>>()
                    .join("\n");

                Some(Draft {
                    kind: ClipboardKind::Files,
                    sub_kind: None,
                    content,
                    search_text: Some(basenames),
                    summary: None,
                    file_types: Some(file_types_str),
                    width: None,
                    height: None,
                    size: None,
                })
            }
        }
        ClipboardPayload::Image(image) => {
            if !capture.image {
                return Ok(None);
            }

            let stored = store.store(image)?;
            Some(Draft {
                kind: ClipboardKind::Image,
                sub_kind: None,
                content: stored.file_name,
                search_text: None,
                summary: None,
                file_types: None,
                width: Some(stored.width),
                height: Some(stored.height),
                size: Some(stored.size),
            })
        }
    };

    let Some(draft) = draft else {
        return Ok(None);
    };

    let now = Utc::now();
    Ok(Some(ClipboardItem {
        id: uuid::Uuid::new_v4().to_string(),
        content_hash: content_hash(draft.kind, &draft.content),
        kind: draft.kind,
        sub_kind: draft.sub_kind,
        group_id: None,
        source_app_id: None,
        content: draft.content,
        search_text: draft.search_text,
        summary: draft.summary,
        file_types: draft.file_types,
        size: draft.size,
        width: draft.width,
        height: draft.height,
        use_count: 1,
        is_favorite: false,
        is_pinned: false,
        platform: current_platform(),
        note: None,
        created_at: now,
        updated_at: now,
        source_app_name: None,
        source_app_icon_file: None,
        source_app_icon_path: None,
        image_thumbnail_path: None,
        file_entries: None,
        files_preview_kind: None,
        available_actions: Vec::new(),
        color_preview: None,
        display_created_at: String::new(),
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::clipboard::payload::ImagePayload;

    fn text_payload(text: &str, html: Option<&str>, rtf: Option<&str>) -> ClipboardPayload {
        ClipboardPayload::Text(TextPayload {
            text: text.to_owned(),
            html: html.map(str::to_owned),
            rtf: rtf.map(str::to_owned),
        })
    }

    /// build_item 需要 ImageStore；非图片用例不触图，故指向临时目录即可。
    fn store() -> (TempDir, ImageStore) {
        let dir = TempDir::new();
        let store = ImageStore::for_test(dir.path().join("resources").join("clipboard-images"));
        (dir, store)
    }

    #[test]
    fn plain_text_runs_subtype_detection() {
        let (_d, s) = store();
        let item = build_item(&s, &text_payload("https://example.com", None, None))
            .unwrap()
            .unwrap();
        assert_eq!(item.kind, ClipboardKind::Text);
        assert_eq!(item.sub_kind, Some(ClipboardSubKind::Url));
        assert_eq!(item.content, "https://example.com");
        assert_eq!(item.search_text.as_deref(), Some("https://example.com"));
        assert_eq!(item.size, Some(19));
    }

    #[test]
    fn html_keeps_source_as_content_and_plain_as_search() {
        let (_d, s) = store();
        let item = build_item(
            &s,
            &text_payload("Hello World", Some("<b>Hello</b> World"), None),
        )
        .unwrap()
        .unwrap();
        assert_eq!(item.sub_kind, Some(ClipboardSubKind::Html));
        assert_eq!(item.content, "<b>Hello</b> World");
        // OS 提供的 plain 文本优先作为检索文本。
        assert_eq!(item.search_text.as_deref(), Some("Hello World"));
        assert_eq!(item.size, Some(11));
    }

    #[test]
    fn plain_only_stores_html_payload_as_plain_text() {
        let (_d, s) = store();
        let item = build_item_with_settings(
            &s,
            &text_payload("Hello World", Some("<b>Hello</b> World"), None),
            &Capture::default(),
            &Sensitive::default(),
            true,
        )
        .unwrap()
        .unwrap();

        assert_eq!(item.sub_kind, None);
        assert_eq!(item.content, "Hello World");
        assert_eq!(item.search_text.as_deref(), Some("Hello World"));
        assert_eq!(item.summary.as_deref(), Some("Hello World"));
    }

    #[test]
    fn html_without_os_plain_is_skipped() {
        // 无 OS 纯文本时整条丢弃：列表只能渲染 summary（基于 plain 文本），
        // plain 为空意味着卡片渲染不出有效内容，不入库。
        let (_d, s) = store();
        let item = build_item(&s, &text_payload("", Some("<p>only html</p>"), None)).unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn rtf_uses_os_plain_text_for_search() {
        let (_d, s) = store();
        let item = build_item(
            &s,
            &text_payload("plain repr", None, Some(r"{\rtf1 plain repr}")),
        )
        .unwrap()
        .unwrap();
        assert_eq!(item.sub_kind, Some(ClipboardSubKind::Rtf));
        assert_eq!(item.content, r"{\rtf1 plain repr}");
        assert_eq!(item.search_text.as_deref(), Some("plain repr"));
        assert_eq!(item.size, Some(10));
    }

    #[test]
    fn disabled_plain_text_yields_none() {
        let (_d, s) = store();
        let capture = Capture {
            text: false,
            ..Capture::default()
        };
        let item = build_item_with_settings(
            &s,
            &text_payload("hello", None, None),
            &capture,
            &Sensitive::default(),
            false,
        )
        .unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn disabled_html_skips_rich_text_payload() {
        let (_d, s) = store();
        let capture = Capture {
            html: false,
            ..Capture::default()
        };
        let item = build_item_with_settings(
            &s,
            &text_payload("Hello World", Some("<b>Hello</b> World"), None),
            &capture,
            &Sensitive::default(),
            false,
        )
        .unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn disabled_rtf_skips_rich_text_payload() {
        let (_d, s) = store();
        let capture = Capture {
            rtf: false,
            ..Capture::default()
        };
        let item = build_item_with_settings(
            &s,
            &text_payload("plain repr", None, Some(r"{\rtf1 plain repr}")),
            &capture,
            &Sensitive::default(),
            false,
        )
        .unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn disabled_html_still_allows_enabled_rtf_payload() {
        let (_d, s) = store();
        let capture = Capture {
            html: false,
            ..Capture::default()
        };
        let item = build_item_with_settings(
            &s,
            &text_payload(
                "plain repr",
                Some("<b>plain repr</b>"),
                Some(r"{\rtf1 plain repr}"),
            ),
            &capture,
            &Sensitive::default(),
            false,
        )
        .unwrap()
        .unwrap();
        assert_eq!(item.sub_kind, Some(ClipboardSubKind::Rtf));
        assert_eq!(item.content, r"{\rtf1 plain repr}");
        assert_eq!(item.search_text.as_deref(), Some("plain repr"));
    }

    #[test]
    fn secret_detection_disabled_keeps_token_text() {
        let (_d, s) = store();
        let item = build_item_with_settings(
            &s,
            &text_payload("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890", None, None),
            &Capture::default(),
            &Sensitive {
                secret_detection: false,
            },
            false,
        )
        .unwrap()
        .unwrap();

        assert_eq!(item.kind, ClipboardKind::Text);
        assert_eq!(item.content, "sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890");
    }

    #[test]
    fn secret_detection_enabled_skips_token_text() {
        let (_d, s) = store();
        let item = build_item_with_settings(
            &s,
            &text_payload("sk-abcdefghijklmnopqrstuvwxyzABCDE1234567890", None, None),
            &Capture::default(),
            &Sensitive {
                secret_detection: true,
            },
            false,
        )
        .unwrap();

        assert!(item.is_none());
    }

    #[test]
    fn files_join_with_newline() {
        let (_d, s) = store();
        let payload = ClipboardPayload::Files(vec!["/a/b.txt".to_owned(), "/c/d".to_owned()]);
        let item = build_item(&s, &payload).unwrap().unwrap();
        assert_eq!(item.kind, ClipboardKind::Files);
        assert_eq!(item.content, "/a/b.txt\n/c/d");
        assert_eq!(
            item.content_hash,
            content_hash(ClipboardKind::Files, "/a/b.txt\n/c/d")
        );
    }

    #[test]
    fn image_is_stored_and_recorded() {
        let (_d, s) = store();
        let payload = ClipboardPayload::Image(ImagePayload {
            bytes: sample_png(20, 10),
            width: 20,
            height: 10,
        });
        let item = build_item(&s, &payload).unwrap().unwrap();
        assert_eq!(item.kind, ClipboardKind::Image);
        assert!(item.content.ends_with(".png"));
        assert_eq!(item.width, Some(20));
        assert_eq!(item.height, Some(10));
        assert!(item.size.unwrap() > 0);
        assert_eq!(
            item.content_hash,
            content_hash(ClipboardKind::Image, &item.content)
        );
        // 原图确实落盘。
        assert!(s.origin_path(&item.content).exists());
    }

    #[test]
    fn disabled_image_yields_none() {
        let (_d, s) = store();
        let capture = Capture {
            image: false,
            ..Capture::default()
        };
        let payload = ClipboardPayload::Image(ImagePayload {
            bytes: sample_png(20, 10),
            width: 20,
            height: 10,
        });
        let item =
            build_item_with_settings(&s, &payload, &capture, &Sensitive::default(), false).unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn disabled_files_yields_none() {
        let (_d, s) = store();
        let capture = Capture {
            files: false,
            ..Capture::default()
        };
        let payload = ClipboardPayload::Files(vec!["/a/b.txt".to_owned()]);
        let item =
            build_item_with_settings(&s, &payload, &capture, &Sensitive::default(), false).unwrap();
        assert!(item.is_none());
    }

    #[test]
    fn blank_text_yields_none() {
        let (_d, s) = store();
        assert!(build_item(&s, &text_payload("  \n\t", None, None))
            .unwrap()
            .is_none());
    }

    // ---- 测试辅助 ----

    fn sample_png(w: u32, h: u32) -> Vec<u8> {
        use std::io::Cursor;
        let buf = image::RgbaImage::from_pixel(w, h, image::Rgba([1, 2, 3, 255]));
        let mut out = Cursor::new(Vec::new());
        image::DynamicImage::ImageRgba8(buf)
            .write_to(&mut out, image::ImageFormat::Png)
            .unwrap();
        out.into_inner()
    }

    struct TempDir(std::path::PathBuf);
    impl TempDir {
        fn new() -> Self {
            let p = std::env::temp_dir().join(format!("ecopaste-ingest-{}", uuid::Uuid::new_v4()));
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
