<div align="center">
  <img src="https://socialify.git.ci/ayangweb/EcoPaste/image?description=1&font=Source%20Code%20Pro&forks=1&issues=1&logo=https%3A%2F%2Fgithub.com%2Fayangweb%2FEcoPaste%2Fblob%2Fmaster%2Fpublic%2Flogo.png%3Fraw%3Dtrue&name=1&owner=1&pattern=Floating%20Cogs&pulls=1&stargazers=1&theme=Auto" alt="EcoPaste" />

  <div>
    <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
      <img
        alt="macOS"
        src="https://img.shields.io/badge/-MacOS-black?style=flat-square&logo=apple&logoColor=white"
      />
    </a>
    <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
      <img
        alt="Windows"
        src="https://img.shields.io/badge/-Windows-blue?style=flat-square&logo=windows&logoColor=white"
      />
    </a>
  </div>

  <div>
    <a href="https://github.com/ayangweb/EcoPaste/blob/master/LICENSE">
      <img
        src="https://img.shields.io/github/license/ayangweb/EcoPaste?style=flat-square"
      />
    </a>
    <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
      <img
        src="https://img.shields.io/github/package-json/v/ayangweb/EcoPaste?style=flat-square"
      />
    </a>
    <a href="https://github.com/ayangweb/EcoPaste/releases">
      <img
        src="https://img.shields.io/github/downloads/ayangweb/EcoPaste/total?style=flat-square"
      />  
    </a>
  </div>
</div>

## 下载

- **MacOS**: [Apple Silicon](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.4/EcoPaste_0.0.4_aarch64.dmg) | [Intel](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.4/EcoPaste_0.0.4_x64.dmg)
- **Windows**: [Universal](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.4/EcoPaste_0.0.4_x64_zh-CN.msi)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/app-dark.png" />
  <source media="(prefers-color-scheme: light)" srcset="./images/app-light.png" />
  <img src="./images/app-light.png" />
</picture>

## 功能介绍

- **轻量小巧，多平台用**：
  
  使用 Tauri 构建，应用更加小巧精致，资源占用极少，同时完美适配 Windows 和 MacOS 平台，保证多平台一致的用户体验。

- **常驻后台，快捷唤醒**：

  应用常驻后台运行，通过自定义快捷键一键唤醒，帮助用户快速调用剪切板内容，极大地提高工作效率。

- **本地存储，数据安全**：

  所有剪切板内容均在本地存储，确保数据隐私和安全，不会有任何数据泄漏风险。

- **智能管理，类型分组**：

  支持纯文本、富文本、HTML、图片和文件类型，并自动根据剪切板内容类型进行分组管理，方便用户高效查找和使用。

- **收藏功能，快速访问**：

  支持收藏剪切板内容，方便快速访问和管理重要信息。无论是工作中的关键数据还是日常生活中的常用信息，都可以轻松收藏和查看。

- **内置搜索，轻松查找**：

  内置搜索功能，帮助用户快速找到任何剪切板上的内容，无论是文本、图片（OCR文本搜索）还是文件，所有内容都可以一目了然。

- **图片OCR，离线识别**：

  内置系统 OCR 功能，支持文本和二维码识别，通过右键菜单快速复制识别到的 OCR 内容，实现离线识别，使用更便捷。

- **右键菜单，操作便捷**：

  提供丰富的自定义选项，用户可以根据自己的使用习惯和需求调整应用的各项设置，打造最适合自己的剪切板管理工具。

- **自由定制，个性体验**：

  提供详细的配置选项，用户可以自由设置和调整应用效果，打造个性化的使用体验，满足不同场景和需求。

- **自动更新，保持最新**：

  软件支持自动更新和手动检查更新功能，确保用户始终使用最新版本，享受最优质的使用体验。

- **数据备份，轻松迁移**：

  支持导出和导入配置及剪切板内容，便于数据备份与迁移到不同平台，保证数据的持续可用性。

- **更多功能，等你探索**：

  EcoPaste 还在不断开发和添加更多有趣实用的功能，期待与你一起探索和发现这款剪切板管理工具的更多可能性。

## 常见问题

<details>
<summary>1. MacOS 提示 app 已损坏，无法打开。</summary>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/injure-dark.png" />
  <source media="(prefers-color-scheme: light)" srcset="./images/injure-light.png" />
  <img src="./images/injure-light.png" />
</picture>

在终端窗口输入以下命令，按回车键后输入系统密码并再次按回车键即可。

```bash
sudo xattr -r -d com.apple.quarantine /Applications/EcoPaste.app
```

</details>

## 参与贡献

请参考 [Contributing Guide](https://github.com/ayangweb/EcoPaste/blob/master/.github/CONTRIBUTING.md)。

## 历史星标

<a href="https://star-history.com/#ayangweb/EcoPaste&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date" />
 </picture>
</a>

## 感谢贡献者们做出的努力

<a href="https://github.com/ayangweb/EcoPaste/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=ayangweb/EcoPaste" />
</a>
