const COMMANDS: &[&str] = &["paste"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
