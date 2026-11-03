//! WebDAV 后台同步调度器：负责定时同步与有变动时防抖自动同步。

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use chrono::{DateTime, Utc};
use tauri::{AppHandle, Listener, Manager};
use tokio::sync::Notify;

use super::sync::sync_now;
use crate::settings::SettingsStore;

/// 调度器检查定时间隔的时钟步长。
const SCHEDULER_TICK_INTERVAL: Duration = Duration::from_secs(60);

/// 变动防抖等待时长（当有剪贴板变动时，静默 20 秒后触发增量同步）。
const DEBOUNCE_DELAY: Duration = Duration::from_secs(20);

/// 启动 WebDAV 后台同步任务。
pub fn spawn(app: AppHandle) {
    let notify = Arc::new(Notify::new());
    let pending_change = Arc::new(AtomicBool::new(false));

    // 监听剪贴板更新事件，触发防抖同步
    {
        let notify = notify.clone();
        let pending = pending_change.clone();
        let app_handle = app.clone();
        app.listen("clipboard://updated", move |_| {
            if is_auto_sync_enabled(&app_handle) {
                pending.store(true, Ordering::SeqCst);
                notify.notify_one();
            }
        });
    }

    // 监听设置更新事件，触发防抖同步
    {
        let notify = notify.clone();
        let pending = pending_change.clone();
        let app_handle = app.clone();
        app.listen("settings://updated", move |_| {
            if is_auto_sync_enabled(&app_handle) {
                pending.store(true, Ordering::SeqCst);
                notify.notify_one();
            }
        });
    }

    // 变动防抖自动同步任务
    {
        let app = app.clone();
        let notify = notify.clone();
        let pending = pending_change.clone();
        tauri::async_runtime::spawn(async move {
            loop {
                notify.notified().await;
                // 防抖循环：等待 DEBOUNCE_DELAY，如果期间再次收到通知，持续等待直到空闲
                loop {
                    tokio::time::sleep(DEBOUNCE_DELAY).await;
                    if pending.swap(false, Ordering::SeqCst) {
                        // 期间又有新变动，再等一个防抖窗口
                        continue;
                    }
                    break;
                }

                if is_auto_sync_enabled(&app) {
                    log::info!("webdav: auto sync triggered by changes");
                    if let Err(err) = sync_now(&app).await {
                        log::warn!("webdav auto sync failed: {err}");
                    }
                }
            }
        });
    }

    // 定时周期性同步任务
    {
        let app = app.clone();
        tauri::async_runtime::spawn(async move {
            let mut ticker = tokio::time::interval(SCHEDULER_TICK_INTERVAL);
            ticker.tick().await;

            let mut last_periodic_sync = Instant::now();

            loop {
                ticker.tick().await;

                let settings = match app.try_state::<SettingsStore>() {
                    Some(s) => s.snapshot(),
                    None => continue,
                };

                let webdav = &settings.webdav;
                if !webdav.enabled || webdav.sync_interval_mins == 0 {
                    continue;
                }

                let interval = Duration::from_secs(u64::from(webdav.sync_interval_mins) * 60);

                // 判断距上次同步是否已达到设置的周期
                let should_sync = match &webdav.last_sync_at {
                    Some(last_str) => match DateTime::parse_from_rfc3339(last_str) {
                        Ok(dt) => {
                            let elapsed = Utc::now().signed_duration_since(dt.with_timezone(&Utc));
                            elapsed.num_seconds().max(0) as u64 >= interval.as_secs()
                        }
                        Err(_) => last_periodic_sync.elapsed() >= interval,
                    },
                    None => true,
                };

                if should_sync {
                    log::info!("webdav: periodic sync triggered");
                    if let Err(err) = sync_now(&app).await {
                        log::warn!("webdav periodic sync failed: {err}");
                    }
                    last_periodic_sync = Instant::now();
                }
            }
        });
    }
}

fn is_auto_sync_enabled(app: &AppHandle) -> bool {
    let settings = match app.try_state::<SettingsStore>() {
        Some(s) => s.snapshot(),
        None => return false,
    };
    settings.webdav.enabled && settings.webdav.auto_sync_on_change
}
