fn main() {
    #[cfg(target_os = "windows")]
    link_windows_ocr();

    tauri_build::build()
}

/// Use the same static MSVC runtime for Rust and OCR libraries; no runtime OCR DLL lookup.
#[cfg(target_os = "windows")]
fn link_windows_ocr() {
    println!("cargo:rerun-if-env-changed=VCPKG_ROOT");
    println!("cargo:rerun-if-env-changed=VCPKGRS_TRIPLET");
    println!("cargo:rerun-if-env-changed=VCPKGRS_DYNAMIC");
    let architecture = std::env::var("CARGO_CFG_TARGET_ARCH").expect("Cargo target architecture");
    let architecture = match architecture.as_str() {
        "x86_64" => "x64",
        "aarch64" => "arm64",
        _ => panic!("Windows OCR requires an x64 or ARM64 MSVC target"),
    };
    let expected_triplet = format!("{architecture}-windows-static");
    let triplet = std::env::var("VCPKGRS_TRIPLET").expect(
        "Set VCPKGRS_TRIPLET to the target's windows-static triplet after preparing Tesseract",
    );
    assert_eq!(
        triplet, expected_triplet,
        "Windows OCR must use static libraries with the static MSVC CRT"
    );
    assert!(
        std::env::var_os("VCPKGRS_DYNAMIC").is_none(),
        "Windows OCR must not depend on external OCR DLLs"
    );
    let features = std::env::var("CARGO_CFG_TARGET_FEATURE").unwrap_or_default();
    assert!(
        features.split(',').any(|feature| feature == "crt-static"),
        "Use the repository Windows target flags and an explicit CARGO_BUILD_TARGET"
    );
    vcpkg::Config::new().find_package("tesseract").expect(
        "Prepare the pinned static Tesseract vcpkg installation before building Windows OCR",
    );

    // vcpkg-rs walks port dependencies but does not emit their Windows SDK libraries.
    for library in [
        "advapi32", "bcrypt", "crypt32", "gdi32", "iphlpapi", "normaliz", "ole32", "secur32",
        "shell32", "user32", "uuid", "version", "winmm", "ws2_32", "wldap32",
    ] {
        println!("cargo:rustc-link-lib={library}");
    }
}
