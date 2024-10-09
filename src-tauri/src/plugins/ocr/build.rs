const COMMANDS: &[&str] = &["system_ocr"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
