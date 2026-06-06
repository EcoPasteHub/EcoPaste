//! 历史清理后台任务：按 `clipboard.history.retention` + `maxCount` 定期裁剪。
//!
//! 启动即跑一次；之后按用户设置的清理周期触发，每次都从 `SettingsStore` 取最新配置——
//! 用户在偏好里调时长 / 上限后不必重启即可生效。置顶与收藏项一律保留（由 [`cleanup_history`] 保证）。

use std::time::{Duration, Instant};

use chrono::{DateTime, Duration as ChronoDuration, Utc};
use serde_json::json;
use sqlx::SqlitePool;
use tauri::{AppHandle, Emitter, Manager};

use super::storage::ImageStore;
use super::watcher::CLIPBOARD_UPDATED_EVENT;
use crate::db::items::cleanup_history;
use crate::settings::{Retention, RetentionUnit, SettingsStore};

/// 调度器检查设置与到期状态的频率；真正清理只在用户设置周期到期后执行。
const SCHEDULER_TICK_INTERVAL: Duration = Duration::from_secs(60);

/// 启动历史清理后台任务：启动立即清理一次，之后按设置周期到点清理。
pub fn spawn(app: AppHandle, pool: SqlitePool) {
    tauri::async_runtime::spawn(async move {
        run_once(&app, &pool).await;
        let mut last_cleanup_at = Instant::now();
        let mut ticker = tokio::time::interval(SCHEDULER_TICK_INTERVAL);
        ticker.tick().await;

        loop {
            ticker.tick().await;
            let Some(interval) = cleanup_interval(&app) else {
                continue;
            };

            if last_cleanup_at.elapsed() < interval {
                continue;
            }

            run_once(&app, &pool).await;
            last_cleanup_at = Instant::now();
        }
    });
}

async fn run_once(app: &AppHandle, pool: &SqlitePool) {
    let history = match app.try_state::<SettingsStore>() {
        Some(store) => store.snapshot().clipboard.history,
        None => return,
    };

    let cutoff = retention_cutoff(&history.retention, Utc::now());
    let max = (history.max_count > 0).then_some(history.max_count);

    if cutoff.is_none() && max.is_none() {
        return;
    }

    match cleanup_history(pool, cutoff, max).await {
        Ok(outcome) if outcome.removed == 0 => {}
        Ok(outcome) => {
            remove_images(app, &outcome.image_files);
            log::info!("history cleanup removed {} item(s)", outcome.removed);
            if let Err(err) = app.emit(
                CLIPBOARD_UPDATED_EVENT,
                json!({ "cleanup": outcome.removed }),
            ) {
                log::warn!("emit cleanup event failed: {err}");
            }
        }
        Err(err) => log::warn!("history cleanup failed: {err}"),
    }
}

/// 读取当前清理周期。`0` 表示关闭周期性清理。
fn cleanup_interval(app: &AppHandle) -> Option<Duration> {
    let store = app.try_state::<SettingsStore>()?;
    let hours = store.snapshot().clipboard.history.cleanup_interval_hours;

    if hours == 0 {
        return None;
    }

    Some(Duration::from_secs(u64::from(hours) * 60 * 60))
}

/// 删除被清理图片记录的落盘文件（原图 + 缩略图）。`ImageStore` 未注册或单个文件删除失败
/// 都只记日志、不阻断——清理本身已成功，残留文件最坏只是占用磁盘，不影响功能。
fn remove_images(app: &AppHandle, file_names: &[String]) {
    if file_names.is_empty() {
        return;
    }
    let Some(store) = app.try_state::<ImageStore>() else {
        log::warn!(
            "image store unavailable; skip removing {} image file(s)",
            file_names.len()
        );
        return;
    };
    for file_name in file_names {
        if let Err(err) = store.remove(file_name) {
            log::warn!("remove cleaned image {file_name} failed: {err}");
        }
    }
}

/// `Retention` → 绝对截止时间。`Forever` 或 `value == 0` 表示禁用。
/// 月份近似按 30 天处理（与前端展示口径一致，不引日历库）。
fn retention_cutoff(r: &Retention, now: DateTime<Utc>) -> Option<DateTime<Utc>> {
    if r.value == 0 {
        return None;
    }
    let dur = match r.unit {
        RetentionUnit::Forever => return None,
        RetentionUnit::Hours => ChronoDuration::hours(r.value as i64),
        RetentionUnit::Days => ChronoDuration::days(r.value as i64),
        RetentionUnit::Weeks => ChronoDuration::weeks(r.value as i64),
        RetentionUnit::Months => ChronoDuration::days((r.value as i64) * 30),
    };
    Some(now - dur)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn now() -> DateTime<Utc> {
        DateTime::from_timestamp(1_700_000_000, 0).unwrap()
    }

    #[test]
    fn retention_cutoff_returns_none_when_disabled() {
        assert!(retention_cutoff(
            &Retention {
                value: 0,
                unit: RetentionUnit::Days
            },
            now()
        )
        .is_none());
        assert!(retention_cutoff(
            &Retention {
                value: 7,
                unit: RetentionUnit::Forever
            },
            now()
        )
        .is_none());
    }

    #[test]
    fn retention_cutoff_subtracts_by_unit() {
        let n = now();
        assert_eq!(
            retention_cutoff(
                &Retention {
                    value: 2,
                    unit: RetentionUnit::Hours
                },
                n
            ),
            Some(n - ChronoDuration::hours(2))
        );
        assert_eq!(
            retention_cutoff(
                &Retention {
                    value: 3,
                    unit: RetentionUnit::Days
                },
                n
            ),
            Some(n - ChronoDuration::days(3))
        );
        assert_eq!(
            retention_cutoff(
                &Retention {
                    value: 1,
                    unit: RetentionUnit::Weeks
                },
                n
            ),
            Some(n - ChronoDuration::weeks(1))
        );
        assert_eq!(
            retention_cutoff(
                &Retention {
                    value: 1,
                    unit: RetentionUnit::Months
                },
                n
            ),
            Some(n - ChronoDuration::days(30))
        );
    }
}
