use anyhow::anyhow;
use core_graphics::event::{CGEvent, CGEventFlags, CGEventTapLocation, CGKeyCode};
use core_graphics::event_source::{CGEventSource, CGEventSourceStateID};

use crate::core::error::Result;

// kVK_ANSI_V，HIToolbox/Events.h 中定义的硬件无关键码。
const KEY_V: CGKeyCode = 0x09;

/// 向系统事件队列投递一次 ⌘V，模拟「粘贴」。
///
/// 需要使用者在「系统设置 → 隐私与安全性 → 辅助功能」授予本应用权限；
/// 未授权时 CGEvent 会被静默丢弃，是 macOS 的安全模型限制，无法绕过。
pub fn simulate_paste() -> Result<()> {
    let source = CGEventSource::new(CGEventSourceStateID::CombinedSessionState)
        .map_err(|_| anyhow!("create CGEventSource failed"))?;

    let key_down = CGEvent::new_keyboard_event(source.clone(), KEY_V, true)
        .map_err(|_| anyhow!("create key-down event failed"))?;
    key_down.set_flags(CGEventFlags::CGEventFlagCommand);
    key_down.post(CGEventTapLocation::HID);

    let key_up = CGEvent::new_keyboard_event(source, KEY_V, false)
        .map_err(|_| anyhow!("create key-up event failed"))?;
    key_up.set_flags(CGEventFlags::CGEventFlagCommand);
    key_up.post(CGEventTapLocation::HID);

    Ok(())
}
