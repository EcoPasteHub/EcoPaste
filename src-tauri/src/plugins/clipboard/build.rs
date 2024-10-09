const COMMANDS: &[&str] = &[
    "start_listen",
    "stop_listen",
    "has_files",
    "has_image",
    "has_html",
    "has_rtf",
    "has_text",
    "read_files",
    "read_image",
    "read_html",
    "read_rtf",
    "read_text",
    "write_files",
    "write_image",
    "write_html",
    "write_rtf",
    "write_text",
];

fn main() {
    tauri_plugin::Builder::new(COMMANDS).build();
}
