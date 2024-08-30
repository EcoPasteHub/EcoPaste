use mouse_position::mouse_position::Mouse;
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Error, Result, Wry,
};

#[command]
async fn get_mouse_coords() -> Result<(i32, i32)> {
    let position = Mouse::get_mouse_position();

    match position {
        Mouse::Position { x, y } => Ok((x, y)),
        Mouse::Error => Err(Error::InvokeKey),
    }
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("mouse")
        .invoke_handler(generate_handler![get_mouse_coords])
        .build()
}
