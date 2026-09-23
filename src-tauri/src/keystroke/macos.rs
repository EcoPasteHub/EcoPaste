use std::sync::atomic::{AtomicBool, Ordering};

use anyhow::anyhow;
use core_graphics::event::{CGEvent, CGEventFlags, CGEventTapLocation, CGKeyCode};
use core_graphics::event_source::{CGEventSource, CGEventSourceStateID};
use macos_accessibility_client::accessibility::{
    application_is_trusted, application_is_trusted_with_prompt,
};

use crate::core::error::Result;

/// kVK_ANSI_V，HIToolbox/Events.h 中定义的硬件无关键码。
const KEY_V: CGKeyCode = 0x09;

/// 本进程生命周期内是否已弹过系统授权引导。
static PERMISSION_PROMPTED: AtomicBool = AtomicBool::new(false);

/// 当前进程是否已被「辅助功能」信任。
///
/// 授权绑定的是二进制的代码签名：未走 Apple 签名的构建在更新或重编译后，
/// 系统设置里的开关仍显示开启，但这里会返回 false，需要用户删除后重新添加。
pub fn is_paste_permitted() -> bool {
    application_is_trusted()
}

/// 弹出系统「辅助功能」授权引导；每个进程只弹一次，避免连续粘贴时反复打断。
pub fn prompt_paste_permission() {
    if PERMISSION_PROMPTED.swap(true, Ordering::SeqCst) {
        return;
    }

    application_is_trusted_with_prompt();
}

/// 向系统事件队列投递一次 ⌘V，模拟「粘贴」。
///
/// 需要使用者在「系统设置 → 隐私与安全性 → 辅助功能」授予本应用权限；
/// 未授权时 CGEvent 会被静默丢弃，是 macOS 的安全模型限制，无法绕过，
/// 调用方应先用 [`is_paste_permitted`] 判断。
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
