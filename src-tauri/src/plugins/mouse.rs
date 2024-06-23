use device_query::{DeviceQuery, DeviceState};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Result, Wry,
};

#[command]
async fn get_mouse_coords() -> Result<(i32, i32)> {
    let device_state = DeviceState::new();

    let mouse = device_state.get_mouse();

    Ok(mouse.coords)
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("mouse")
        .invoke_handler(generate_handler![get_mouse_coords])
        .build()
}
