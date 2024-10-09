use tauri::command;

// 获取系统语言
#[command]
pub fn get_locale() -> String {
    let locale = current_locale::current_locale();

    if locale.is_ok() {
        return locale.ok().unwrap();
    }

    return "zh-CN".to_string();
}
