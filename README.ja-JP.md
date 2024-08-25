<a href="https://github.com/ayangweb/EcoPaste">
  <img src="https://socialify.git.ci/ayangweb/EcoPaste/image?description=1&descriptionEditable=MacOS%E3%81%8A%E3%82%88%E3%81%B3Windows%E3%83%97%E3%83%A9%E3%83%83%E3%83%88%E3%83%95%E3%82%A9%E3%83%BC%E3%83%A0%E5%90%91%E3%81%91%E3%81%AE%E3%82%AA%E3%83%BC%E3%83%97%E3%83%B3%E3%82%BD%E3%83%BC%E3%82%B9%E3%82%AF%E3%83%AA%E3%83%83%E3%83%97%E3%83%9C%E3%83%BC%E3%83%89%E7%AE%A1%E7%90%86%E3%83%84%E3%83%BC%E3%83%AB%E3%80%82&font=Source%20Code%20Pro&forks=1&issues=1&logo=https%3A%2F%2Fgithub.com%2Fayangweb%2FEcoPaste%2Fblob%2Fmaster%2Fpublic%2Flogo.png%3Fraw%3Dtrue&name=1&owner=1&pattern=Floating%20Cogs&pulls=1&stargazers=1&theme=Auto" alt="EcoPaste" />
</a>

<div align="center">
  <br/>

  <div>
      日本語 | <a href="./README.md">English</a> | <a href="./README.zh-CN.md">简体中文</a> | <a href="./README.zh-TW.md">繁體中文</a>
  </div>

  <br/>
    
  <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
    <img
      alt="Windows"
      src="https://img.shields.io/badge/-Windows-blue?style=flat-square&logo=windows&logoColor=white"
    />
  </a >  
  <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
    <img
      alt="macOS"
      src="https://img.shields.io/badge/-MacOS-black?style=flat-square&logo=apple&logoColor=white"
    />
  </a >
  <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
    <img 
      alt="Linux"
      src="https://img.shields.io/badge/-Linux-yellow?style=flat-square&logo=linux&logoColor=white" 
    />
  </a>

  <div>
    <a href="https://github.com/ayangweb/EcoPaste/blob/master/LICENSE">
      <img
        src="https://img.shields.io/github/license/ayangweb/EcoPaste?style=flat-square"
      />
    </a >
    <a href="https://github.com/ayangweb/EcoPaste/releases/latest">
      <img
        src="https://img.shields.io/github/package-json/v/ayangweb/EcoPaste?style=flat-square"
      />
    </a >
    <a href="https://github.com/ayangweb/EcoPaste/releases">
      <img
        src="https://img.shields.io/github/downloads/ayangweb/EcoPaste/total?style=flat-square"
      />  
    </a >
  </div>

  <br/>

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./images/app-dark.ja-JP.png" />
    <source media="(prefers-color-scheme: light)" srcset="./images/app-light.ja-JP.png" />
    <img src="./images/app-light.ja-JP.png" />
  </picture>
</div>

## ダウンロード

### Windows

手動インストール: [Universal](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.5/EcoPaste_0.0.5_x64_zh-CN.msi)

インストールなし: [Universal](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.6/EcoPaste_0.0.6_Windows_x64_Portable.zip)

### Macos

手動インストール: [Apple Silicon](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.5/EcoPaste_0.0.5_aarch64.dmg) | [Intel](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.5/EcoPaste_0.0.5_x64.dmg)

HomeBrew:

1. リポジトリを追加:
```shell
brew tap ayangweb/EcoPaste
```

2. インストール:
```shell
brew install ecopaste
```

3. アップデート:
```shell
brew upgrade ecopaste
```

4. アンインストール:
```shell
brew uninstall --cask ecopaste

brew untap ayangweb/EcoPaste
```

### Linux

手動インストール: [AppImage](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.6/eco-paste_0.0.6_amd64.AppImage) | [deb](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.6/eco-paste_0.0.6_amd64.deb) | [rpm](https://mirror.ghproxy.com/https://github.com/ayangweb/EcoPaste/releases/download/v0.0.6/eco-paste-0.0.6-1.x86_64.rpm)

## 機能の概要

- **軽量でコンパクト、多プラットフォーム対応**: 
  
  Tauri構築、リソース消費が少なく、WindowsとMacOSに対応し、一貫したユーザーエクスペリエンスを保証。

- **常駐バックグラウンド、素早く起動**：

  アプリは常にバックグラウンドに常駐、カスタムショートカットキーで素早く呼び出し、クリップボードの内容をすばやく利用。
- 
  **画像OCR、オフライン認識**：

  内蔵のOCR機能、テキストとQRコードの認識をサポートし、右クリックメニューでOCR内容を素早くコピー。

  **組み込み検索**：
  内蔵検索機能、クリップボード内容をすばやく見つける、テキスト、画像（OCRテキスト検索）、ファイルを含む。

- **ローカルストレージ、データセキュリティ**：

  すべてのクリップボード内容はローカルに保存され、データプライバシーを確保。

- **もっと探検する**：

  `EcoPaste` は新機能を継続的に開発しており、より多くの可能性を共に探索することを楽しみにしています

## Q&A

<details>
<summary>1. MacOS のヒント EcoPaste.app は壊れているため開けません。</summary>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/damaged-dark.ja-JP.png" />
  <source media="(prefers-color-scheme: light)" srcset="./images/damaged-light.ja-JP.png" />
  <img src="./images/damaged-light.ja-JP.png" />
</picture>

アプリが実行できるようにするため、`terminal` で次のコマンドを入力し、Enterキーを押してください: 

> コマンドを実行するにはパスワードが必要な場合があります。

```bash
sudo xattr -r -d com.apple.quarantine /Applications/EcoPaste.app
```

その後、アプリケーションを正常に開くことができます。

</details>

## スター歴史

<a href="https://star-history.com/#ayangweb/EcoPaste&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=ayangweb/EcoPaste&type=Date" />
 </picture>
</a>

## 貢献者

EcoPaste への貢献をいただいた皆様に感謝いたします。 

EcoPaste に貢献したい方は、[貢献ガイドライン](./.github/CONTRIBUTING.ja-JP.md)を参照してください。

<a href="https://github.com/ayangweb/EcoPaste/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=ayangweb/EcoPaste" />
</a>
