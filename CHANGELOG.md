# Changelog

## [1.0.0](https://github.com/EcoPasteHub/EcoPaste/compare/v0.6.0-beta.3...v1.0.0) (2026-07-03)

This is a fully refactored version of EcoPaste. The app is lighter, faster, and more stable overall.

### ✨ Features

- EcoPaste can now open on Windows without taking focus from other windows.
- Added a Windows Win+V toggle, so EcoPaste can replace the system clipboard panel.
- Added first-run onboarding for permissions, shortcuts, ignored apps, and legacy data import.
- Added source app detection and ignored apps, so you can control which apps are excluded from clipboard history.
- Added capture preferences for saved content types, size limits, and priority.
- Added sensitive content protection to automatically skip high-risk content such as private keys, tokens, AWS keys, and JWTs.
- Added full content preview for text, image, and file records.
- Added drag-out support for dragging text, rich text, images, and file records into external apps.
- Added deletion protection for favorites and pinned items.
- Added preference search and preference reset.
- Backup import and export now support encrypted backups and merge import.
- Added a Windows administrator launch setting.
- Moved software updates into a dedicated window for checking, downloading, and installing new versions.
- More new features are ready for you to download, try, and explore.

### 🐛 Bug Fixes

- Fixed autostart not working when EcoPaste runs as administrator on Windows.

### ⚡️ Performance

- Clipboard monitoring, search, list rendering, and content preview are faster and more stable.
- App startup, everyday usage, and background resource usage have been further optimized.
- Image thumbnails, app icons, and file icons now load more efficiently.
- Added lightweight mode to reduce background refresh work and memory usage after the window is hidden.

### ⚠️ Upgrade Notice

- This refactored version only supports macOS and Windows.
- This version supports migrating history data from the legacy app during first-run onboarding.
