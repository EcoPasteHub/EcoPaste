/// 初始化 prevent-default 插件，禁用 webview 内置的浏览器默认行为
/// （右键菜单、F5 刷新、Ctrl+P 打印、Ctrl+F 查找等），让应用更像原生窗口。
///
/// debug 构建保留 devtools / reload / 右键菜单等开发所需行为；
/// release 构建用 `init()` 全部禁用。
pub fn init() -> tauri::plugin::TauriPlugin<tauri::Wry> {
    #[cfg(debug_assertions)]
    {
        tauri_plugin_prevent_default::debug()
    }

    #[cfg(not(debug_assertions))]
    tauri_plugin_prevent_default::init()
}
