# Windows OCR resources

Run `scripts/prepare-windows-ocr.ps1` before building on Windows. It pins Tesseract's
vcpkg toolchain and downloads SHA-256-verified `tessdata_fast` English and Simplified
Chinese models (6,582,244 bytes combined). Runtime recognition is fully offline.
The generated `tessdata/` and `licenses/` directories are bundled in the Windows
installer, with licenses for Tesseract, its linked libraries and the models.

Model revision: `87416418657359cb625c412a48b6e1d6d41c29bd`.
Vcpkg revision: `9e3427bc82738568947beb508e78231f99c04f4c` (Tesseract 5.5.2).

Static library triplets use the static C runtime (`x64-windows-static` or
`arm64-windows-static`). Windows NSIS installation needs no MSIX identity or
separate OCR language pack. The model loader verifies the same hashes at runtime
before passing in-memory model bytes to Tesseract.

The script sets an explicit `CARGO_BUILD_TARGET` and disables Tauri's separate
`STATIC_VCRUNTIME` linker shim; Rust and native dependencies share `+crt-static`.
