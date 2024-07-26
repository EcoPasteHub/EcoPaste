use super::App;
use cocoa::base::{id, nil};
use cocoa::foundation::{NSAutoreleasePool, NSString};
use objc::declare::ClassDecl;
use objc::runtime::{Class, Object, Sel};
use objc::{msg_send, sel, sel_impl};
use std::ffi::CStr;
use std::sync::Mutex;
use std::thread;

static FOREMOST_APPS: Mutex<Vec<App>> = Mutex::new(Vec::new());

extern "C" fn application_did_activate(_self: &Object, _cmd: Sel, notification: id) {
    unsafe {
        let ns_app_key = NSString::alloc(nil).init_str("NSWorkspaceApplicationKey");
        let user_info: id = msg_send![notification, userInfo];
        if user_info == nil {
            println!("No userInfo in notification");
            return;
        }

        let app: id = msg_send![user_info, objectForKey: ns_app_key];
        if app == nil {
            println!("No application found in userInfo");
            return;
        }

        let process_id: i32 = msg_send![app, processIdentifier];
        let localized_name: id = msg_send![app, localizedName];
        let name_str: *const i8 = msg_send![localized_name, UTF8String];

        // 确保从 C 字符串转换为 Rust 字符串时，正确处理了生命周期问题
        let name_cstr = CStr::from_ptr(name_str);
        let name = name_cstr.to_str().unwrap_or("Unknown").to_string();

        let mut apps = FOREMOST_APPS.lock().unwrap();

        if apps.len() == 2 {
            apps.remove(0);
        }

        apps.push(App { name, process_id });
    }
}

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

pub fn get_frontmost_apps() -> Vec<App> {
    return FOREMOST_APPS.lock().unwrap().to_vec();
}
