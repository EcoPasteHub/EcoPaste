const COMMANDS: &[&str] = &["metadata", "preview_path"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
