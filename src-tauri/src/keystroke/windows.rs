use anyhow::anyhow;
use std::mem::{size_of, zeroed};
use winapi::um::winuser::{
    SendInput, INPUT, INPUT_KEYBOARD, KEYBDINPUT, KEYEVENTF_KEYUP, VK_CONTROL,
};

use crate::core::error::Result;

/// V 键的虚拟键码，winapi 未直接导出。
const VK_V: u16 = 0x56;

/// 向系统事件队列投递一次 Ctrl+V，模拟「粘贴」。
///
/// 之所以选 Ctrl+V 而非 Shift+Insert：
/// - Monaco editor（VS Code 聊天输入、所有 Web 嵌入式代码编辑器）把 Insert 解释为
///   「切换插入/改写模式」并吞掉 Shift 修饰，导致「粘贴失效 + 光标变成块状」。
/// - Ctrl+V 是 Windows / Chromium / Electron / 现代 IDE 输入控件普遍约定，
///   与剪贴板内容类型无关，覆盖面最广。
pub fn simulate_paste() -> Result<()> {
    let mut inputs: [INPUT; 4] = unsafe { zeroed() };

    // Ctrl 按下
    inputs[0].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[0].u.ki_mut() = KEYBDINPUT {
            wVk: VK_CONTROL as u16,
            wScan: 0,
            dwFlags: 0,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // V 按下
    inputs[1].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[1].u.ki_mut() = KEYBDINPUT {
            wVk: VK_V,
            wScan: 0,
            dwFlags: 0,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // V 抬起
    inputs[2].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[2].u.ki_mut() = KEYBDINPUT {
            wVk: VK_V,
            wScan: 0,
            dwFlags: KEYEVENTF_KEYUP,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // Ctrl 抬起
    inputs[3].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[3].u.ki_mut() = KEYBDINPUT {
            wVk: VK_CONTROL as u16,
            wScan: 0,
            dwFlags: KEYEVENTF_KEYUP,
            time: 0,
            dwExtraInfo: 0,
        };
    }

    let sent = unsafe {
        SendInput(
            inputs.len() as u32,
            inputs.as_mut_ptr(),
            size_of::<INPUT>() as i32,
        )
    };
    if sent as usize != inputs.len() {
        return Err(anyhow!("SendInput injected {sent}/{} events", inputs.len()).into());
    }
    Ok(())
}
