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
