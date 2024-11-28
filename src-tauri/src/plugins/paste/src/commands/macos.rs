use cocoa::base::{id, nil};
use cocoa::foundation::{NSAutoreleasePool, NSString};
use objc::declare::ClassDecl;
use objc::runtime::{Class, Object, Sel};
use objc::{msg_send, sel, sel_impl};
use std::ffi::CStr;
use std::process::Command;
use std::sync::Mutex;
use std::thread;
use tauri::{command, AppHandle, Runtime, WebviewWindow};
use tauri_plugin_eco_window::{set_macos_panel, MacOSPanelStatus, MAIN_WINDOW_TITLE};

static PREVIOUS_WINDOW: Mutex<Option<i32>> = Mutex::new(None);

extern "C" fn application_did_activate(_self: &Object, _cmd: Sel, notification: id) {
    unsafe {
        let ns_app_key = NSString::alloc(nil).init_str("NSWorkspaceApplicationKey");

        let user_info: id = msg_send![notification, userInfo];
        if user_info == nil {
            return;
        }

        let app: id = msg_send![user_info, objectForKey: ns_app_key];
        if app == nil {
            return;
        }

        let localized_name: id = msg_send![app, localizedName];
        let name_str: *const i8 = msg_send![localized_name, UTF8String];
        let name_cstr = CStr::from_ptr(name_str);
        let name = name_cstr.to_str().unwrap_or("Unknown").to_string();

        if name == MAIN_WINDOW_TITLE {
            return;
        }

        let process_id: i32 = msg_send![app, processIdentifier];

        let mut previous_window = PREVIOUS_WINDOW.lock().unwrap();
        let _ = previous_window.insert(process_id);
    }
}

// 监听窗口切换
pub fn observe_app() {
    thread::spawn(|| unsafe {
        let _pool = NSAutoreleasePool::new(nil);

        let superclass = Class::get("NSObject").unwrap();
        let mut decl = ClassDecl::new("AppObserver", superclass).unwrap();
        decl.add_method(
            sel!(applicationDidActivate:),
            application_did_activate as extern "C" fn(&Object, Sel, id),
        );
        let observer_class = decl.register();
        let observer: id = msg_send![observer_class, new];

        let workspace: id = msg_send![Class::get("NSWorkspace").unwrap(), sharedWorkspace];
        let notification_center: id = msg_send![workspace, notificationCenter];
        let ns_notification_name =
            NSString::alloc(nil).init_str("NSWorkspaceDidActivateApplicationNotification");

        let _: id = msg_send![notification_center,
            addObserver: observer
            selector: sel!(applicationDidActivate:)
            name: ns_notification_name
            object: nil
        ];

        let run_loop: id = msg_send![Class::get("NSRunLoop").unwrap(), currentRunLoop];
        let _: () = msg_send![run_loop, run];
    });
}

// 获取前一个窗口
pub fn get_previous_window() -> Option<i32> {
    return PREVIOUS_WINDOW.lock().unwrap().clone();
}

// 粘贴
#[command]
pub async fn paste<R: Runtime>(app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    set_macos_panel(&app_handle, &window, MacOSPanelStatus::Resign);

    let script = r#"tell application "System Events" to keystroke "v" using command down"#;

    Command::new("osascript")
        .args(["-e", script])
        .output()
        .expect("failed to execute process");
}
