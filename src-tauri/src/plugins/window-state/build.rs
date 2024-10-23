const COMMANDS: &[&str] = &["save_window_state", "restore_window_state"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
