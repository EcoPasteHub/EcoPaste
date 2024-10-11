const COMMANDS: &[&str] = &["get_locale"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
