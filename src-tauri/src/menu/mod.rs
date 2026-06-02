//! 原生菜单（Rust 侧）：解决前端 `Menu.new` 即用即丢导致的 Windows muda
//! use-after-free（点击菜单项后崩溃/卡顿），并把菜单生命周期收回到 Rust。

pub mod clipboard_item;
