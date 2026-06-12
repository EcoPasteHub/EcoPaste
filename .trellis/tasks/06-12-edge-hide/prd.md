# Edge Hide

## Goal

主窗口拖到屏幕边缘时自动收起，只保留很窄的触发带；鼠标贴近边缘时自动唤出。

## Scope

- 仅支持主窗口。
- 仅支持 macOS 与 Windows。
- 贴边方向先支持 `Top`、`Left`、`Right`，不做底部，避免与 Dock / 任务栏冲突。
- 默认关闭，由设置开关启用。

## Implementation Notes

- 新增 `window/edge_hide.rs`，维护 `EdgeHideState { docked, hidden, peek_strip_px }`。
- 监听窗口移动结束事件：macOS 使用 NSWindow move notification，Windows 使用 `WM_EXITSIZEMOVE` 或现有窗口事件封装。
- 距当前显示器 work area 边缘小于等于 8px 时进入 docked 状态。
- `slide_out` 将窗口移到屏幕外，只露出约 4px 触发带。
- docked 状态下启动 OS 级鼠标位置轮询，未 docked 时停止。
- 鼠标进入触发带时 `slide_in`，失焦且鼠标离开窗口区域一段时间后再次收起。
- 拖离边缘超过 32px 时解除 docked。
- 新增 `Settings.clipboard.window.edgeHideEnabled: bool`，默认 false。
- 设置变更实时启停 edge hide 模块。
- 如确需命令，`set_edge_hide_enabled(enabled)` 只做薄封装，不把 docked 内部状态暴露给前端。

## Acceptance Criteria

- 三个边缘方向都能稳定收起和唤出。
- 拖动远离边缘后不再自动收起。
- 多显示器与高 DPI 下触发带可见且位置正确。
- 关闭设置后停止轮询并恢复普通窗口行为。
- macOS NSPanel 唤出时不要抢焦点。
- Windows 高 DPI 下触发带宽度按 `monitor.scale_factor()` 换算。
- 全局快捷键唤出窗口时，若处于 hidden docked 状态，应直接 `slide_in`。
