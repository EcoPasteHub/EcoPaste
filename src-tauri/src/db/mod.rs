pub mod apps;
pub mod groups;
pub mod init;
pub mod items;
pub mod models;
pub mod path;

pub use init::init;
pub use path::db_path;

#[cfg(test)]
pub(crate) mod test_support {
    use std::str::FromStr;

    use sqlx::sqlite::{SqliteConnectOptions, SqlitePoolOptions};
    use sqlx::SqlitePool;

    /// 仓储层单测用的内存库连接池：单连接（保证所有查询命中同一个 in-memory DB）+
    /// 开启外键（验证 `ON DELETE SET NULL`）+ 跑完整 `migrations`。
    pub async fn memory_pool() -> SqlitePool {
        let options = SqliteConnectOptions::from_str("sqlite::memory:")
            .expect("valid sqlite url")
            .foreign_keys(true);

        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect_with(options)
            .await
            .expect("failed to open in-memory sqlite");

        sqlx::migrate!("./migrations")
            .run(&pool)
            .await
            .expect("failed to run migrations on in-memory sqlite");

        pool
    }
}
