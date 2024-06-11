# EcoCopy

一款开源的剪切板管理工具，同时支持 MacOS 和 Windows 两个平台。

TODO：

- [ ] 贴边的剪切板窗口（卡在窗口无法在任务栏之上）
- [ ] 复制内容来源的应用信息（欢迎各位 rust 大佬指点）
- [ ] 快捷回复窗口
- [ ] 兼容 Windows
    1. - [ ] 当前的 `createWindow` 函数在 Windows 系统有 bug
    2. - [ ] `appWindow.onFocusChanged` 在 Windows 系统会频繁触发
    3. - [ ] 磨砂窗口在 Windows 系统不生效
    4. - [ ] 在 Rust 中使用除 Windows 系统以外生效的函数时不要直接导入，在使用时引入就好了