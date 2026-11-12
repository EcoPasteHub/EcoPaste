# macOS 平台集成

## 1. 窗口模型

### 1.1 窗口列表

```text
clipboard          主剪贴板列表
preference         偏好设置
onboarding         首次启动引导
update             更新
clipboard-preview  悬停/空格预览 overlay
```

Windows 另有 `context-menu`、`context-submenu`，macOS 不使用。

### 1.2 主窗口 = NSPanel

`src-tauri/src/window/macos.rs` 把主 WebViewWindow 转为 NSPanel：

```text
is_floating_panel = true
can_become_key_window = true
can_become_main_window = false
style mask: resizable + nonactivating_panel
level: Dock
collection behavior:
  stationary
  move_to_active_space (隐藏时)
  can_join_all_spaces (显示时)
  full_screen_auxiliary
```

目的是：

- 显示时不激活 EcoPaste，不把前台 App 推走。
- 能成为 key window 接收键盘。
- 在所有 Space 和全屏辅助场景可用。
- 失焦可自动隐藏。

### 1.3 显示时序

```text
show_window
  -> 应用窗口布局
  -> async sleep 16ms
  -> main thread:
       panel.show_and_make_key()
       collectionBehavior = can_join_all_spaces
       resume preview
       emit window://visibility(true)
       lifecycle::on_shown
```

16ms 延迟让窗口 geometry/layout 先稳定，避免 NSPanel 第一帧位置跳动。

### 1.4 pin 与自动隐藏

两个独立状态：

- `CLIPBOARD_WINDOW_PINNED`：用户显式固定。
- `CLIPBOARD_WINDOW_AUTO_HIDE_SUSPENDED`：临时暂停，例如系统文件选择。

失焦自动隐藏的条件：

```text
!pinned && !autoHideSuspended
```

固定窗口粘贴时不能让 panel 继续持有 key，否则 `CGEvent` 的 ⌘V 会被 panel 自己吞掉。因此固定粘贴前 `resign_key_window`，粘贴后 50ms 再 `make_key_window`。

### 1.5 窗口位置

`WindowPosition`：

- `FollowCursor`：主窗口出现在光标位置，夹到当前显示器范围内。
- `Center`：当前光标所在显示器居中。
- `Remember`：恢复已保存几何；几何失效时回退光标显示器居中。

打开剪贴板时可配置：

- 回到顶部。
- 范围：preserve/all/favorite。
- 分类：preserve/all/text/image/files。
- 自定义分组：preserve/all/`group:<id>`。

### 1.6 窗口生命周期

`window/lifecycle/` 用 descriptor 统一管理：

| 窗口 | RetainPolicy |
| --- | --- |
| clipboard | Permanent |
| preference | DestroyWhenIdle |
| onboarding | DestroyWhenIdle |
| update | DestroyWhenIdle |
| clipboard-preview | DestroyWhenIdle |

Phase：

```text
notCreated -> created -> ready -> visible
                                  |
                                  v
                              hiddenWarm
                              /        \
                        dormant       destroyPending -> destroyed
```

- 主窗口隐藏 5 秒后 dormant，不销毁。
- 其它窗口空闲 60 秒（设置夹在 5 秒–24 小时）后销毁。
- 销毁前发 `window://before-destroy`，等待 500ms。
- `dirtyOwnerCount > 0` 或 keepalive lease 存在时阻止销毁。
- generation 用于让过期计时器失效。

### 1.7 Dock 与 reopen

- `general.dockIcon` 控制 Dock 可见性。
- 点击 Dock 图标且无可见窗口时，未完成 onboarding 则打开 onboarding，否则打开偏好。

## 2. 全局快捷键

当前使用 `tauri-plugin-global-shortcut`：

- 默认 `open_clipboard = Alt+C`
- 默认 `open_preference = Alt+X`
- 只响应 `Pressed`，避免 Pressed/Released 触发两次。
- 注册失败 emit `shortcut://conflict`，不阻断另一个快捷键。
- 偏好页录制快捷键时 `suspend_global_shortcuts` 注销全部；结束后 160ms debounce 恢复。
- Windows 另有 `win_v`，macOS 忽略。

迁到 nsgo 时：

- nsgo `desktop.RegisterHotKey` 在 macOS 使用 Carbon `RegisterEventHotKey`，不要求 Accessibility。
- 需要保留冲突反馈、暂停/恢复和快捷键字符串格式兼容。
- Carbon 注册 API 的键位映射与 Tauri 插件不完全相同，必须做映射表和冲突测试。

## 3. 托盘/菜单栏

当前 `tray::init`：

- macOS 使用 `assets/tray-mac.ico`，`icon_as_template = true`。
- 左键显示菜单。
- 菜单项：
  - 偏好设置
  - 停止/开启监听
  - 开源地址
  - 检查更新
  - 当前版本（禁用）
  - 重启
  - 退出
- 语言或 `trayIcon` 变化时重建菜单/更新可见性。
- 监听暂停状态翻转后只重建菜单，不重建 watcher 线程。

nsgo `desktop.NewTray` 已提供 macOS `NSStatusItem`，可用于替代该层，但必须验证：

- template image / Retina 图标。
- 左键/右键行为。
- 菜单动态重建。
- 与 NSPanel 的 anchor 定位。

## 4. 右键菜单

### 4.1 macOS

- 使用 Tauri `muda` 原生菜单。
- 菜单构建和 `popup_menu` 必须主线程。
- `popup_menu` 在菜单关闭前阻塞，因此命令立即返回，实际弹出派发到主线程。
- 菜单持有到下一次 popup 前替换，避免事件派发期间 use-after-free。
- 菜单动作通过 `clipboard://menu-action` 回传前端。

菜单动作分组包括：

- 粘贴/纯文本粘贴/粘贴为路径
- 复制/纯文本复制
- 保存图片
- 打开链接/发邮件
- Finder 中显示
- 收藏/置顶
- 移动到分组
- 编辑备注
- 删除

### 4.2 目标实现

AppKit 原生菜单应使用 `NSMenu` / `NSMenuItem` / `NSMenuItemValidation`，不应继续走 WebView 菜单窗口。菜单快捷键、禁用态、勾选态和二级分组菜单要在 responder chain 中正确工作。

## 5. 粘贴模拟

`src-tauri/src/keystroke/macos.rs`：

- `CGEventSourceStateID::CombinedSessionState`
- key code `0x09` = kVK_ANSI_V
- key down/up 设置 `CGEventFlagCommand`
- post 到 `CGEventTapLocation::HID`

前提：

- 系统设置 -> 隐私与安全性 -> 辅助功能授权。
- 未授权时事件被系统静默丢弃。

完整粘贴时序：

```text
写回剪贴板 + WritebackGuard
  -> 记录复用次数（可选）
  -> 固定窗口: resign key
     非固定窗口: hide
  -> await 50ms
  -> post Cmd+V
  -> 固定窗口: await 50ms + make key
```

迁移风险：

- AppKit 重写后窗口是否是 key、前台 App 是否接管事件，需要重新实测。
- 固定窗口和自动隐藏模式必须分别验证。
- 辅助功能权限在产品迁移、签名身份变化后可能要求用户重新授权。

## 6. 拖出

`src-tauri/src/drag_out/macos.rs` 直接调用 AppKit：

- 文件：`NSDraggingItem` + file URL。
- 文本：`NSPasteboardItem` 同时写 plain/html/rtf。
- 使用 `NSDraggingSession` 和自定义 `NSDraggingSource`。
- 取 `window.ns_view()` -> `contentView` -> `beginDraggingSessionWithItems_event_source:`。
- 预览图固定长边 128pt，避免高分辨率 PNG 按像素当 point 显示过大。
- 文本预览在 PNG 缺失时用 `NSImage` + `NSAttributedString` 现场渲染。

目标 AppKit 版本应保留：

- 拖出文件 URL、图片、plain/html/rtf 多格式。
- 拖拽预览比例和 Retina 清晰度。
- 拖拽取消/完成回调。

nsgo 当前主要提供“拖入文件”事件，不提供拖出到外部应用能力，必须自建。

## 7. 来源应用识别

macOS：

```text
NSWorkspace.sharedWorkspace.frontmostApplication
  -> bundleIdentifier
  -> localizedName
  -> bundleURL
  -> 图标 PNG
```

约束：

- 必须在剪贴板事件回调里同步捕获。
- 没有 bundle id 的进程不入表，避免主键不稳定。
- 自身写回也会先抓到 EcoPaste，但随后被 `WritebackGuard` 跳过。

## 8. 权限

偏好页暴露：

| 权限 | 当前用途 |
| --- | --- |
| Accessibility | 模拟 ⌘V；未授权时 CGEvent 被丢弃 |
| Full Disk Access | 偏好页通过 macOS permissions plugin 检测和请求；当前 Rust 业务代码主要把它作为权限入口，具体资源访问依赖仍需按各模块路径复核 |

实现注意：

- 权限状态不能只存本地布尔值，必须读系统真实状态。
- 迁移后 bundle id、签名和 app 路径变化可能导致系统重新要求授权。
- 首次启动 onboarding 和偏好页都要提供明确的跳转/重试路径。

## 9. 更新

当前 Tauri updater：

- stable endpoint：`https://releases.ecopaste.cn/update?channel=stable`
- beta/nightly endpoint 通过同一 host 的 channel query 区分。
- 自动检查 daily/weekly/monthly。
- 下载进度 `update://progress`。
- 跳过版本写入 settings。
- macOS 更新包和签名由 Tauri updater 体系决定。

nsgo：

- macOS 使用 Sparkle。
- `nsgo build` 生成 `appcast-macos_<arch>.xml`。
- Sparkle 验证 EdDSA 签名并在独立窗口完成更新。
- 现有 Tauri endpoint 和签名 key 不能直接复用。

迁移策略必须在动手前确定，见 [08-migration-plan-and-risks.md](08-migration-plan-and-risks.md)。

## 10. 单实例与文件打开

- Tauri single-instance plugin 处理第二实例。
- 第二实例是 autostart 拉起时只退出，不打开偏好。
- 第二实例带 `.ecopastebak` 参数时转交备份接收。
- macOS 文件关联走 `RunEvent::Opened`，冷启动时可能早于 `setup` 完成，因此有就绪闸。
- RunEvent 回调跨 ObjC C 边界，panic 不可 unwind，代码用 `catch_unwind` 兜底。

AppKit 版本应使用 `application:openURLs:` 或 `NSApplicationDelegate` 的对应回调，并保留“启动早期收到文件”的排队语义。

## 11. 不能只靠编译验证的行为

以下必须在真实 macOS 桌面会话压测：

1. 菜单栏应用无 Dock 图标时 NSPanel 能否成为 key。
2. 多显示器、全屏 Space、不同缩放比例下的定位。
3. 失焦隐藏、pin、临时暂停的组合。
4. 固定/非固定窗口粘贴。
5. 辅助功能授权被撤销/重新授予。
6. 拖出到 Finder、浏览器、Word/Pages、纯文本编辑器。
7. 复制图片、HTML、RTF、文件列表后立即切换前台应用。
8. 系统截图、浏览器复制图片、密码管理器复制等边界来源。

