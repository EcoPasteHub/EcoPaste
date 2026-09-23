//! Image OCR command boundary; queue/search policy is owned by the Rust OCR service.
use tauri::AppHandle;

use crate::core::Result;
use crate::ocr::{self, ImageOcrStatus};

#[tauri::command]
pub async fn get_image_ocr_status(app: AppHandle) -> Result<ImageOcrStatus> {
    ocr::status(&app).await
}

#[tauri::command]
pub async fn queue_image_ocr_history(app: AppHandle) -> Result<ImageOcrStatus> {
    ocr::queue_history(&app).await
}

#[tauri::command]
pub async fn clear_image_ocr_index(app: AppHandle) -> Result<ImageOcrStatus> {
    ocr::clear(&app).await
}
