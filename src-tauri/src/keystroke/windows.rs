use anyhow::anyhow;
use std::mem::{size_of, zeroed};
use winapi::um::winuser::{
    SendInput, INPUT, INPUT_KEYBOARD, KEYBDINPUT, KEYEVENTF_KEYUP, VK_INSERT, VK_SHIFT,
};

use crate::core::error::Result;

/// 向系统事件队列投递一次 Shift+Insert，模拟「粘贴」。
///
/// 之所以选 Shift+Insert 而非 Ctrl+V：
/// - 传统 Win32 控件、cmd/PowerShell 控制台、部分 Electron/终端类应用对
///   Ctrl+V 不响应或被自定义快捷键吞掉，而 Shift+Insert 是 Windows 系统级
///   编辑约定（NT 时代沿用），几乎所有可输入控件都接收。
/// - 与剪贴板里的内容类型无关，纯触发系统粘贴行为。
pub fn simulate_paste() -> Result<()> {
    let mut inputs: [INPUT; 4] = unsafe { zeroed() };

    // Shift 按下
    inputs[0].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[0].u.ki_mut() = KEYBDINPUT {
            wVk: VK_SHIFT as u16,
            wScan: 0,
            dwFlags: 0,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // Insert 按下
    inputs[1].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[1].u.ki_mut() = KEYBDINPUT {
            wVk: VK_INSERT as u16,
            wScan: 0,
            dwFlags: 0,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // Insert 抬起
    inputs[2].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[2].u.ki_mut() = KEYBDINPUT {
            wVk: VK_INSERT as u16,
            wScan: 0,
            dwFlags: KEYEVENTF_KEYUP,
            time: 0,
            dwExtraInfo: 0,
        };
    }
    // Shift 抬起
    inputs[3].type_ = INPUT_KEYBOARD;
    unsafe {
        *inputs[3].u.ki_mut() = KEYBDINPUT {
            wVk: VK_SHIFT as u16,
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
