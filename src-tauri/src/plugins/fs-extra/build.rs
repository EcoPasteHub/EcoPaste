const COMMANDS: &[&str] = &["metadata", "open_path"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
