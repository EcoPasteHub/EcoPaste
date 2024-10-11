const COMMANDS: &[&str] = &[
    "check_accessibility_permissions",
    "request_accessibility_permissions",
    "request_full_disk_access_permissions",
];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
