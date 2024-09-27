<a href="https://github.com/EcoPasteHub/EcoPaste">
  <img src="https://socialify.git.ci/EcoPasteHub/EcoPaste/image?description=1&descriptionEditable=Open%20source%20clipboard%20management%20tools%20for%20Windows%2C%20MacOS%20and%20Linux(x11).&font=Source%20Code%20Pro&forks=1&issues=1&logo=https%3A%2F%2Fgithub.com%2FEcoPasteHub%2FEcoPaste%2Fblob%2Fmaster%2Fpublic%2Flogo.png%3Fraw%3Dtrue&name=1&owner=1&pattern=Floating%20Cogs&pulls=1&stargazers=1&theme=Auto" alt="EcoPaste" />
</a>

<div align="center">
  <br/>
  
  <div>
      English | <a href="./README.md">简体中文</a> | <a href="./README.zh-TW.md">繁體中文</a> | <a href="./README.ja-JP.md">日本語</a>
  </div>

  <br/>
    
  <a href="https://github.com/EcoPasteHub/EcoPaste/releases">
    <img
      alt="Windows"
      src="https://img.shields.io/badge/-Windows-blue?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB0PSIxNzI2MzA1OTcxMDA2IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjE1NDgiIHdpZHRoPSIxMjgiIGhlaWdodD0iMTI4Ij48cGF0aCBkPSJNNTI3LjI3NTU1MTYxIDk2Ljk3MTAzMDEzdjM3My45OTIxMDY2N2g0OTQuNTEzNjE5NzVWMTUuMDI2NzU3NTN6TTUyNy4yNzU1NTE2MSA5MjguMzIzNTA4MTVsNDk0LjUxMzYxOTc1IDgwLjUyMDI4MDQ5di00NTUuNjc3NDcxNjFoLTQ5NC41MTM2MTk3NXpNNC42NzA0NTEzNiA0NzAuODMzNjgyOTdINDIyLjY3Njg1OTI1VjExMC41NjM2ODE5N2wtNDE4LjAwNjQwNzg5IDY5LjI1Nzc5NzUzek00LjY3MDQ1MTM2IDg0Ni43Njc1OTcwM0w0MjIuNjc2ODU5MjUgOTE0Ljg2MDMxMDEzVjU1My4xNjYzMTcwM0g0LjY3MDQ1MTM2eiIgcC1pZD0iMTU0OSIgZmlsbD0iI2ZmZmZmZiI+PC9wYXRoPjwvc3ZnPg=="
    />
  </a >  
  <a href="https://github.com/EcoPasteHub/EcoPaste/releases">
    <img
      alt="MacOS"
      src="https://img.shields.io/badge/-MacOS-black?style=flat-square&logo=apple&logoColor=white"
    />
  </a >
  <a href="https://github.com/EcoPasteHub/EcoPaste/releases">
    <img 
      alt="Linux"
      src="https://img.shields.io/badge/-Linux-yellow?style=flat-square&logo=linux&logoColor=white" 
    />
  </a>

  <div>
    <a href="./LICENSE">
      <img
        src="https://img.shields.io/github/license/EcoPasteHub/EcoPaste?style=flat-square"
      />
    </a >
    <a href="https://github.com/EcoPasteHub/EcoPaste/releases">
      <img
        src="https://img.shields.io/github/package-json/v/EcoPasteHub/EcoPaste?style=flat-square"
      />
    </a >
    <a href="https://github.com/EcoPasteHub/EcoPaste/releases">
      <img
        src="https://img.shields.io/github/downloads/EcoPasteHub/EcoPaste/total?style=flat-square"
      />  
    </a >
  </div>
  
  <br/>

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./images/app-dark.en-US.png" />
    <source media="(prefers-color-scheme: light)" srcset="./images/app-light.en-US.png" />
    <img src="./images/app-light.en-US.png" />
 </picture>
</div>

## Download

### Windows

Install Manually: [Universal](https://ecopaste-updater.ayangweb.cn/api/stable?platform=windows)

### Macos

Install Manually: [Apple Silicon](https://ecopaste-updater.ayangweb.cn/api/stable?platform=macos-arm) | [Intel](https://ecopaste-updater.ayangweb.cn/api/stable?platform=macos-x64)

HomeBrew:

1. Add our tap:
```shell
brew tap EcoPasteHub/EcoPaste
```

2. Install:
```shell
brew install ecopaste
```

3. Upgrade:
```shell
brew upgrade ecopaste
```

4. Uninstall:
```shell
brew uninstall --cask ecopaste

brew untap EcoPasteHub/EcoPaste
```

### Linux(x11)

Install Manually: [AppImage](https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-appimage) | [deb](https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-deb) | [rpm](https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-rpm)

* Ubuntu、Debain:
```shell
wget "https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-deb" -O ecopaste-stable.deb
sudo dpkg -i ecopaste-stable.deb
```

* Red Hat、CentOS、Fedora:
```shell
wget "https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-rpm" -O ecopaste-stable.rpm
sudo yum install ecopaste-stable.rpm
```

* Manjaro、ArchLinux:
```shell
yay -S eco-paste-bin
```

* Others:
```shell
wget "https://ecopaste-updater.ayangweb.cn/api/stable?platform=linux-appimage" -O EcoPaste.AppImage
chmod +x EcoPaste.AppImage
./EcoPaste.AppImage --appimage-extract

sed -i 's|Exec=.*|Exec=/opt/ecopaste/EcoPaste.AppImage|g' squashfs-root/usr/share/applications/eco-paste.desktop
sed -i 's/^Icon=.*/Icon=EcoPaste/g' squashfs-root/usr/share/applications/eco-paste.desktop
install -Dm644 squashfs-root/usr/share/applications/eco-paste.desktop /usr/share/applications/EcoPaste.desktop

wget "https://raw.githubusercontent.com/EcoPasteHub/EcoPaste-Logo/refs/heads/master/img/logo.svg" -O EcoPaste.svg
install -Dm644 EcoPaste.svg /usr/share/icons/hicolor/scalable/apps/EcoPaste.svg

rm -r squashfs-root EcoPaste.svg

mkdir -p /opt/ecopaste
mv ecopaste.AppImage /opt/ecopaste/
```

## Features

- **Lightweight & Cross-platform Supported**: 
  
  Built with Tauri, the application is lightweight and refined, consuming minimal resources. It also delivers a uniform user experience across both Windows, MacOS and Linux platforms.

- **Background Operation & Instant Access**：

  The application is resident in the background, wake up with one click through custom shortcut keys, save time and improve efficiency.

- **Local Storage & Data Security**：

  All clipboard content is stored locally to ensure data privacy and security.

- **Smart Grouping**：

  Supports plain text, rich text, HTML, images, and files. Automatically groups clipboard content by type, managing your clipboard content efficiently.

- **Favorites**：

  Allows you to bookmark clipboard content for easy and fast access. Whether it's crucial data for work or frequently used information in daily life, you can effortlessly save and retrieve it.

- **Built-in Search**：

  Built-in search helps users quickly find any content on the clipboard, whether it's text, images (with OCR text search), or files, making everything easily accessible at a glance.

- **Offline Image OCR**：

  Recognize text and QR codes in seconds. Quickly copy recognized content via the context menu, enabling convenient offline text recognition.

- **Enhanced Context Menu**：

  Providing rich right-click menu options, users can quickly perform various operations, which greatly improves the ease of use.

- **Highly Customizable**：

  Provide abundant customization options to meet various scenarios and individual preferences.

- **Automatic Update**：

  Supports automatic and manual update checks, ensuring users always have the latest version for the best experience.

- **Backup & Migration**：

  Supports exporting and importing configurations and clipboard content, making data backup and migration to different platforms seamless, and ensuring continuous data availability.

- **More to Explore**：

  `EcoPaste` continues to evolve with exciting new features. We look forward to sharing more possibilities with you.

## Q&A

<details>
<summary>1. "EcoPaste.app is damaged and can't be opened" on MacOS </summary>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/damaged-dark.en-US.png" />
  <source media="(prefers-color-scheme: light)" srcset="./images/damaged-light.en-US.png" />
  <img src="./images/damaged-light.en-US.png" />
</picture>

Type the following command and press Enter in `terminal` to allow the app to run: 

> Password may be required to run the command.

```bash
sudo xattr -r -d com.apple.quarantine /Applications/EcoPaste.app
```

After that, you can open the app normally.

</details>

## Star History

<a href="https://star-history.com/#EcoPasteHub/EcoPaste&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=EcoPasteHub/EcoPaste&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=EcoPasteHub/EcoPaste&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=EcoPasteHub/EcoPaste&type=Date" />
 </picture>
</a>

## Community

We recommend using [issue](https://github.com/EcoPasteHub/EcoPaste/issues) to provide the most direct and effective feedback. Of course, the following options for feedback are also available:

- WeChat

<img width="25%" src="https://picture-bed.ayangweb.cn/EcoPaste/wechat.png" />

## Contributors

Thanks to everyone who has already contributed to EcoPaste. 

If you want to contribute to EcoPaste, please refer to [Contributing Guide](./.github/CONTRIBUTING/en-US.md).

<a href="https://github.com/EcoPasteHub/EcoPaste/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=EcoPasteHub/EcoPaste" />
</a>
