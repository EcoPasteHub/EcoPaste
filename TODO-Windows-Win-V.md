# TODO: Windows 接管 Win+V

## 目标

Windows 平台使用 `Win+V` 唤出 EcoPaste，替代系统自带剪贴板历史面板。

## 范围

- 仅 Windows 实现，macOS 不做对应能力。
- 不修改注册表禁用系统剪贴板历史。
- 用户可在设置里改回其他快捷键。

## Rust 设计

- 在 `src-tauri/src/shortcut/windows.rs` 增加 Windows 专用注册逻辑。
- 优先尝试 `RegisterHotKey(NULL, id, MOD_WIN, 'V')`。
- 成功后在线程 message loop 中监听 `WM_HOTKEY`，触发主窗口 toggle。
- 失败时走低级键盘钩子兜底，拦截 `VK_LWIN/RWIN + V` 并返回非零阻断系统默认处理。
- 复用现有 Windows keyboard hook 分发，不要重复安装多个 `WH_KEYBOARD_LL`。

## 设置联动

- Windows 下主窗默认快捷键设为 `Win+V`。
- `shortcut/mod.rs` 注册阶段检测主窗快捷键：
  - 是 `Win+V`：跳过 `tauri-plugin-global-shortcut`，走 Windows 专用逻辑。
  - 不是 `Win+V`：拆掉专用钩子，走常规全局快捷键注册。

## 注意

- 启动时可检测系统剪贴板历史是否开启并写日志提示，但不要主动修改系统设置。
- 错误或冲突要通过日志与前端提示让用户知道。

## 验收

- Windows 上按 `Win+V` 唤出 EcoPaste，系统剪贴板历史不弹出。
- 修改为其他快捷键后，`Win+V` 恢复系统默认行为。
- 不影响主窗口可见时的方向键、Space 预览等已有键盘钩子。

