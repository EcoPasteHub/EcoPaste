# 图片文字搜索 / Image OCR search

在“偏好设置 → 历史 → 图片文字识别”中开启。macOS 使用 Apple Vision，Windows 使用随安装包携带的 Tesseract 和中英文模型，均在本机识别，不上传图片。Windows 不依赖 MSIX 身份或单独安装的系统语言包。

- 默认关闭，升级不会自动识别已有历史。开启后，新收录图片自动加入后台队列。
- “识别历史图片”把已有图片加入队列，并重试失败的项目。重复点击不会重做成功的识别。
- 识别完成后直接在原来的搜索框输入图片里的文字；支持中文短词、分类、收藏和自定义分组筛选。
- 暂停只停止后台处理，已识别文字仍可搜索。正在识别的任务完成后会丢弃结果，恢复后重新处理。
- 关闭 OCR 同时停止识别并排除 OCR 搜索结果；已缓存文字保留以便重新开启。
- “清除识别索引”会暂停识别并清除派生文字/任务，不删除图片、剪贴板记录、备注、收藏或置顶。要重新处理旧图，再点击历史识别并继续任务。
- 每次仅处理一张图片。单图上限 20 MiB、4,000 万像素，最多索引 10 万字符；空白图片计为已完成，超限、缺失或识别失败计入失败数。
- 本版只处理剪贴板中的图片记录，不读取文件记录的正文，不扫描磁盘。识别准确率受图片清晰度、字体和布局影响。
- OCR 属于可重建索引。历史导出/合并不承诺保留此索引；导入后可重新识别。停用或清除索引不是原始图片中的文字脱敏。

## English

Enable **Preferences → History → Image OCR**. macOS uses Apple Vision; Windows uses statically linked Tesseract with bundled English and Simplified Chinese models. Recognition stays on-device without API keys, MSIX identity or a separately installed OCR language pack.

OCR is off by default. Enabling queues newly captured image items; **Index existing images** explicitly queues history and retries failed items. Completed text is searched through the ordinary search box with existing filters. Pause stops processing but keeps indexed matches; disable also excludes OCR matches. Clear pauses processing and deletes derived jobs/text only. Original images, history, favorites, pins and notes remain unchanged.

The serial worker accepts up to 20 MiB and 40 megapixels per image and indexes up to 100,000 characters. Empty text is a successful result. Missing, oversized and failed images are counted and retried only on request. Files/PDF/Word extraction is outside this version. Rebuild the OCR index after importing history if needed.

## Windows builds

From PowerShell, run `./scripts/prepare-windows-ocr.ps1 -Architecture x64` before `pnpm tauri build --target $env:CARGO_BUILD_TARGET`. For the ARM64 target use `-Architecture arm64` and the corresponding Rust target. The script pins the vcpkg toolchain, builds static libraries, verifies both model hashes, and prepares the model/library licenses bundled with NSIS. Keep the generated environment variables in the same shell.

The model loader checks exact model hashes before initializing the native engine. Missing/corrupt resources disable OCR with an installation message instead of failing queued jobs indefinitely. Models are loaded through Rust into memory to support non-ASCII installation paths.
