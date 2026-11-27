use std::future::Future;

use sqlx::SqlitePool;
use tokio::sync::Mutex;

use crate::core::Result;

/// Holds the current SQLite pool and serializes hot replacement during backup overwrite import.
pub struct DatabaseState {
    pool: Mutex<SqlitePool>,
}

impl DatabaseState {
    /// Creates database state from the startup pool.
    pub fn new(pool: SqlitePool) -> Self {
        Self {
            pool: Mutex::new(pool),
        }
    }

    /// Returns a clone of the currently active pool.
    pub async fn pool(&self) -> SqlitePool {
        self.pool.lock().await.clone()
    }

    /// Closes the active pool, runs replacement work, and installs the new pool atomically.
    pub async fn close_and_replace<F, Fut>(&self, replace: F) -> Result<()>
    where
        F: FnOnce() -> Fut,
        Fut: Future<Output = Result<SqlitePool>>,
    {
        let mut current = self.pool.lock().await;
        current.close().await;
        let next = replace().await?;
        *current = next.clone();
        // 换了库（切换数据目录 / 备份覆盖导入）就重建列表索引，否则窗口打开还在用旧库的顺序。
        crate::db::list_index::activate(next);
        Ok(())
    }
}
