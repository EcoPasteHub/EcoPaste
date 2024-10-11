const COMMANDS: &[&str] = &["get_mouse_coords"];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
