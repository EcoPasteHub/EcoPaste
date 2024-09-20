use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    updater::builder,
    AppHandle, Wry,
};

#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct CheckUpdateResult {
    manifest: Option<UpdateManifest>,
    should_update: bool,
}

#[derive(Debug, serde::Serialize)]
struct UpdateManifest {
    version: String,
    date: String,
    body: String,
}

#[command]
async fn check_update(app_handle: AppHandle, join_beta: bool) -> Result<CheckUpdateResult, String> {
    builder(app_handle)
        .header("join-beta", join_beta.to_string())
        .map_err(|err| err.to_string())?
        .check()
        .await
        .map(|update| {
            let should_update = update.is_update_available();
            let version = update.latest_version().to_string();
            let date = update.date().unwrap().unix_timestamp().to_string();
            let body = update.body().unwrap().to_string();

            if should_update {
                CheckUpdateResult {
                    should_update,
                    manifest: Some(UpdateManifest {
                        version,
                        date,
                        body,
                    }),
                }
            } else {
                CheckUpdateResult {
                    should_update,
                    manifest: None,
                }
            }
        })
        .map_err(|err| err.to_string())
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("updater")
        .invoke_handler(generate_handler![check_update])
        .build()
}
