use build_info::{
    chrono::{DateTime, Utc},
    GitInfo,
};
use sys_info;

pub fn print_system_info() {
    let os_type = sys_info::os_type().unwrap_or_else(|_| "Unknown".to_string());
    let os_release = sys_info::os_release().unwrap_or_else(|_| "Unknown".to_string());
    let hostname = sys_info::hostname().unwrap_or_else(|_| "Unknown".to_string());
    let cpu_num = sys_info::cpu_num().unwrap_or_else(|_| 0);
    let cpu_speed = sys_info::cpu_speed().unwrap_or_else(|_| 0);
    let mem_total = sys_info::mem_info().map(|m| m.total).unwrap_or_else(|_| 0);
    let mem_free = sys_info::mem_info().map(|m| m.free).unwrap_or_else(|_| 0);

    println!("OS: {}", os_type);
    println!("OS Release: {}", os_release);
    println!("Hostname: {}", hostname);
    println!("CPU Cores: {}", cpu_num);
    println!("CPU Speed: {} MHz", cpu_speed);
    println!("Total Memory: {} KB", mem_total);
    println!("Free Memory: {} KB", mem_free);
}

pub fn print_app_info(app_handle: tauri::AppHandle) {
    let package_info = app_handle.package_info();

    println!("App Name: {}", package_info.name);
    println!("App Version: {}", package_info.version);
}

build_info::build_info!(fn build_info);

pub fn print_build_info() {
    println!("Build Profile: {}", build_info::format!("{}", $.profile));
    if let Some(build_features) = build_features() {
        println!("Build Features: {}", build_features.join(" "));
    }
    println!(
        "Build Timestamp: {}",
        build_info::format!("{}", $.timestamp)
    );
    if let Some(git_branch) = git_branch() {
        println!("Git Branch: {}", git_branch);
    }
    if let Some(git_tags) = git_tags() {
        println!("Git Tags: {}", git_tags.join(" "));
    }
    if let Some(git_commit_id) = git_commit_id() {
        println!("Git Commit Id: {}", git_commit_id);
    }
    if let Some(git_commit_timestamp) = git_commit_timestamp() {
        println!("Git Commit Timestamp: {}", git_commit_timestamp);
    }
}

pub fn build_features() -> Option<&'static Vec<String>> {
    let features = &build_info().crate_info.enabled_features;
    if features.is_empty() {
        None
    } else {
        Some(features)
    }
}

pub fn git_info() -> Option<&'static GitInfo> {
    build_info().version_control.as_ref()?.git()
}

pub fn git_short_commit_id() -> Option<&'static str> {
    Some(git_info()?.commit_short_id.as_str())
}

pub fn git_dirty_str() -> Option<&'static str> {
    if git_info()?.dirty {
        Some(".+")
    } else {
        None
    }
}

pub fn git_commit_id() -> Option<String> {
    Some(git_short_commit_id()?.to_owned() + git_dirty_str().unwrap_or_default())
}

pub fn git_commit_timestamp() -> Option<&'static DateTime<Utc>> {
    Some(&git_info()?.commit_timestamp)
}

pub fn git_branch() -> Option<&'static str> {
    Some(git_info()?.branch.as_ref()?.as_str())
}

pub fn git_tags() -> Option<&'static Vec<String>> {
    let tags = &git_info()?.tags;
    if tags.is_empty() {
        None
    } else {
        Some(tags)
    }
}
