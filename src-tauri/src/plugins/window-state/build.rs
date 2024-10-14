const COMMANDS: &[&str] = &["save_state", "restore_state"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
