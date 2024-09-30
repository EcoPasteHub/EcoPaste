use crate::core::app::get_previous_window;
use cocoa::{
    appkit::{NSApplicationActivationOptions, NSRunningApplication},
    base::nil,
};
use tauri::command;

fn focus_previous_window() {
    let process_id = match get_previous_window() {
        Some(process_id) => process_id,
        None => return,
    };

    unsafe {
        let app = NSRunningApplication::runningApplicationWithProcessIdentifier(nil, process_id);

        app.activateWithOptions_(
            NSApplicationActivationOptions::NSApplicationActivateIgnoringOtherApps,
        );
    }
}

#[command]
pub async fn paste() {
    focus_previous_window();

    let script =
        r#"osascript -e 'tell application "System Events" to keystroke "v" using command down'"#;

    std::process::Command::new("sh")
        .arg("-c")
        .arg(script)
        .output()
        .expect("failed to execute process");
}
