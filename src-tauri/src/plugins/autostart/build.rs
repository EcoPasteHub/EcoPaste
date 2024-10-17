const COMMANDS: &[&str] = &["is_autostart"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
