use mouse_position::mouse_position::Mouse;
use tauri::command;

// 获取鼠标光标的位置
#[command]
pub async fn get_mouse_coords() -> Result<(i32, i32), String> {
    let position = Mouse::get_mouse_position();

    match position {
        Mouse::Position { x, y } => Ok((x, y)),
        Mouse::Error => Err(String::from("Failed to get mouse position")),
    }
}
