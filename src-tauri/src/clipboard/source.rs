//! 「这次剪贴板变更来自哪个应用」的探测：在剪贴板事件回调里调用，
//! 返回稳定 id（macOS bundle id / Windows exe 绝对路径）、显示名、可选的 icon PNG 字节。
//!
//! 必须**同步**在监听回调一发生时立即抓——延后到 await 之后再问，前台应用很可能已经切走。
//! 探测失败（无前台应用 / 自身复制 / 平台 API 错误）一律返回 `None`，不阻断入库。
//!
//! macOS 走 `NSWorkspace.frontmostApplication`，icon 走 `NSImage` → TIFF → `NSBitmapImageRep` → PNG。
//! Windows 走 `GetForegroundWindow` → `QueryFullProcessImageNameW` 拿 exe 路径；
//! icon 走 `ExtractIconExW` → `GetIconInfo` → `GetDIBits`(32bpp BGRA) → `png` crate 编 PNG。

use crate::db::models::Platform;

#[derive(Debug, Clone)]
pub struct FrontmostApp {
    /// 稳定主键。macOS = bundle id（如 `com.apple.Safari`），Windows = exe 绝对路径。
    pub id: String,
    /// 显示名（localizedName / FileDescription / exe stem 的优先回落）。
    pub name: String,
    pub platform: Platform,
    /// 应用图标的 PNG 字节；提取失败则 `None`。
    pub icon_png: Option<Vec<u8>>,
}

/// 探测当前前台应用。失败不报错，只在 trace 级别记日志（监听回调高频，避免噪声）。
pub fn detect_frontmost() -> Option<FrontmostApp> {
    #[cfg(target_os = "macos")]
    {
        macos::detect()
    }
    #[cfg(target_os = "windows")]
    {
        windows::detect()
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        None
    }
}

#[cfg(target_os = "macos")]
mod macos {
    use super::{FrontmostApp, Platform};

    use objc2::rc::{autoreleasepool, Retained};
    use objc2::runtime::AnyObject;
    use objc2::{msg_send, ClassType};
    use objc2_app_kit::{NSBitmapImageFileType, NSBitmapImageRep, NSImage, NSWorkspace};
    use objc2_foundation::{NSData, NSDictionary, NSString};

    pub(super) fn detect() -> Option<FrontmostApp> {
        autoreleasepool(|_| {
            let workspace = NSWorkspace::sharedWorkspace();
            let app = workspace.frontmostApplication()?;

            // 没 bundle id 的进程（命令行子进程等）不入表，避免主键不稳定。
            let id = app.bundleIdentifier().map(|s| nsstring_to_string(&s))?;
            let name = app
                .localizedName()
                .map(|s| nsstring_to_string(&s))
                .unwrap_or_else(|| id.clone());

            let icon_png = app.icon().and_then(|img| unsafe { nsimage_to_png(&img) });

            Some(FrontmostApp {
                id,
                name,
                platform: Platform::Macos,
                icon_png,
            })
        })
    }

    fn nsstring_to_string(s: &NSString) -> String {
        s.to_string()
    }

    /// NSImage → PNG。先 TIFF 表示 → 包装成 NSBitmapImageRep → 编出 PNG。
    /// 直接用 NSImage 的 size 像素，不强行缩放——前端按需 CSS 缩放，避免不同 retina 倍率失真。
    unsafe fn nsimage_to_png(img: &NSImage) -> Option<Vec<u8>> {
        let tiff: Retained<NSData> = msg_send![img, TIFFRepresentation];
        if tiff.is_empty() {
            return None;
        }

        let rep_cls = NSBitmapImageRep::class();
        // imageRepWithData: 返回 NSImageRep（NSBitmapImageRep 的父类）；这里我们知道
        // TIFF 一定走 NSBitmapImageRep 实现，按其类型继续用。
        let rep: Option<Retained<NSBitmapImageRep>> = msg_send![rep_cls, imageRepWithData: &*tiff];
        let rep = rep?;

        let props: Retained<NSDictionary<NSString, AnyObject>> = NSDictionary::new();
        let png: Option<Retained<NSData>> = msg_send![
            &*rep,
            representationUsingType: NSBitmapImageFileType::PNG,
            properties: &*props,
        ];
        let png = png?;
        if png.is_empty() {
            return None;
        }
        Some(png.to_vec())
    }
}

#[cfg(target_os = "windows")]
mod windows {
    use super::{FrontmostApp, Platform};

    use std::mem::{size_of, zeroed};
    use std::path::Path;
    use std::ptr;

    use winapi::shared::minwindef::{DWORD, FALSE};
    use winapi::shared::windef::{HBITMAP, HICON, HWND};
    use winapi::um::handleapi::CloseHandle;
    use winapi::um::processthreadsapi::OpenProcess;
    use winapi::um::shellapi::ExtractIconExW;
    use winapi::um::winbase::QueryFullProcessImageNameW;
    use winapi::um::wingdi::{
        DeleteObject, GetDIBits, GetObjectW, BITMAP, BITMAPINFO, BITMAPINFOHEADER, BI_RGB,
        DIB_RGB_COLORS,
    };
    use winapi::um::winnt::PROCESS_QUERY_LIMITED_INFORMATION;
    use winapi::um::winuser::{
        DestroyIcon, GetDC, GetForegroundWindow, GetIconInfo, GetWindowThreadProcessId, ReleaseDC,
        ICONINFO,
    };

    pub(super) fn detect() -> Option<FrontmostApp> {
        let exe_path = unsafe { foreground_exe_path() }?;
        // 自身写回事件依赖 WritebackGuard 的 content_hash 判定，这里不过滤自身——
        // 与 macOS 行为一致：哪怕拿到的是 EcoPaste 自己，guard 也会在下游 short-circuit。
        let name = Path::new(&exe_path)
            .file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or(&exe_path)
            .to_owned();
        let icon_png = unsafe { extract_icon_png(&exe_path) };

        Some(FrontmostApp {
            id: exe_path,
            name,
            platform: Platform::Windows,
            icon_png,
        })
    }

    unsafe fn foreground_exe_path() -> Option<String> {
        let hwnd: HWND = GetForegroundWindow();
        if hwnd.is_null() {
            return None;
        }
        let mut pid: DWORD = 0;
        GetWindowThreadProcessId(hwnd, &mut pid);
        if pid == 0 {
            return None;
        }
        // PROCESS_QUERY_LIMITED_INFORMATION 足够 QueryFullProcessImageNameW，且不需要管理员权限。
        let handle = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
        if handle.is_null() {
            return None;
        }

        let mut buf = [0u16; 1024];
        let mut size: DWORD = buf.len() as DWORD;
        let ok = QueryFullProcessImageNameW(handle, 0, buf.as_mut_ptr(), &mut size);
        CloseHandle(handle);

        if ok == 0 || size == 0 {
            return None;
        }
        Some(String::from_utf16_lossy(&buf[..size as usize]))
    }

    /// HICON → 32bpp BGRA → RGBA → PNG。失败一律 None，不影响入库。
    unsafe fn extract_icon_png(exe_path: &str) -> Option<Vec<u8>> {
        let wide: Vec<u16> = exe_path.encode_utf16().chain(std::iter::once(0)).collect();
        let mut large: HICON = ptr::null_mut();
        // 只取 large icon（index 0），small icon 用不上。
        let count = ExtractIconExW(wide.as_ptr(), 0, &mut large, ptr::null_mut(), 1);
        if count == 0 || large.is_null() {
            return None;
        }

        let png = hicon_to_png(large);
        DestroyIcon(large);
        png
    }

    unsafe fn hicon_to_png(icon: HICON) -> Option<Vec<u8>> {
        let mut info: ICONINFO = zeroed();
        if GetIconInfo(icon, &mut info) == 0 {
            return None;
        }
        // 任何分支返回前都得 DeleteObject 两个 bitmap，避免 GDI 句柄泄漏。
        let result = hbitmap_to_png(info.hbmColor);
        if !info.hbmColor.is_null() {
            DeleteObject(info.hbmColor as _);
        }
        if !info.hbmMask.is_null() {
            DeleteObject(info.hbmMask as _);
        }
        result
    }

    unsafe fn hbitmap_to_png(hbm: HBITMAP) -> Option<Vec<u8>> {
        if hbm.is_null() {
            return None;
        }
        let mut bm: BITMAP = zeroed();
        if GetObjectW(
            hbm as _,
            size_of::<BITMAP>() as i32,
            &mut bm as *mut _ as *mut _,
        ) == 0
        {
            return None;
        }
        let width = bm.bmWidth;
        let height = bm.bmHeight;
        if width <= 0 || height <= 0 {
            return None;
        }

        let mut bi: BITMAPINFO = zeroed();
        bi.bmiHeader.biSize = size_of::<BITMAPINFOHEADER>() as u32;
        bi.bmiHeader.biWidth = width;
        // 负高度 = top-down 行序，省一次反向遍历。
        bi.bmiHeader.biHeight = -height;
        bi.bmiHeader.biPlanes = 1;
        bi.bmiHeader.biBitCount = 32;
        bi.bmiHeader.biCompression = BI_RGB;

        let stride = (width as usize) * 4;
        let mut buf = vec![0u8; stride * height as usize];

        let hdc = GetDC(ptr::null_mut());
        let scanned = GetDIBits(
            hdc,
            hbm,
            0,
            height as u32,
            buf.as_mut_ptr() as *mut _,
            &mut bi,
            DIB_RGB_COLORS,
        );
        ReleaseDC(ptr::null_mut(), hdc);

        if scanned == 0 {
            return None;
        }

        // GDI 给的是 BGRA，转 RGBA。
        for px in buf.chunks_exact_mut(4) {
            px.swap(0, 2);
        }

        encode_png(width as u32, height as u32, &buf)
    }

    fn encode_png(width: u32, height: u32, rgba: &[u8]) -> Option<Vec<u8>> {
        let mut out = Vec::new();
        {
            let mut encoder = png::Encoder::new(&mut out, width, height);
            encoder.set_color(png::ColorType::Rgba);
            encoder.set_depth(png::BitDepth::Eight);
            let mut writer = encoder.write_header().ok()?;
            writer.write_image_data(rgba).ok()?;
        }
        Some(out)
    }
}
