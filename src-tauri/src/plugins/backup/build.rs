const COMMANDS: &[&str] = &["export_data", "import_data", "move_data"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
