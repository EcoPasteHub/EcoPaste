# EcoCopy

一款开源的剪切板管理工具，同时支持 MacOS 和 Windows 两个平台。

TODO：

- [ ] 贴边的剪切板窗口（卡在窗口无法在任务栏之上）
- [ ] 复制内容来源的应用信息（欢迎各位 rust 大佬指点）
- [ ] 快捷回复窗口
- [x] 兼容 Windows
  1. - [x] 当前的 `createWindow` 函数在 Windows 系统有 bug（Rust `create_window` 方法必须是异步的）
  2. - [x] `appWindow.onFocusChanged` 在 Windows 系统会频繁触发
  3. - [x] ~~磨砂窗口在 Windows 各版本系统表现不一~~（参考其它软件的界面，为保持一致性，暂时不考虑做磨砂窗口）
  4. - [x] 在 Rust 中使用除 Windows 系统以外生效的函数时不要直接导入，在使用时引入就好了
