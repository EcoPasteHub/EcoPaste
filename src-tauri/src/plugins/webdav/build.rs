const COMMANDS: &[&str] = &[
    "set_config",
    "get_config",
    "get_computer_name",
    "test_config",
    "list_backups",
    "upload_backup",
    "cancel_upload",
    "download_backup",
    "delete_backup",
    "create_slim_database",
];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
