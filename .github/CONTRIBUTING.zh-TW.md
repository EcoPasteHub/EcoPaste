# 貢獻指南

非常感謝您對 EcoPaste 的關注和貢獻！在您提交貢獻之前，請先花一些時間閱讀以下指南，以確保您的貢獻能夠順利進行。

## 透明的開發

所有工作都在 GitHub 上公開進行。無論是核心團隊成員還是外部貢獻者的 Pull Request，都需要經過相同的 review 流程。

## 提交 Issue

我們使用 [Github Issues](https://github.com/EcoPasteHub/EcoPaste/issues) 進行 Bug 報告和新 Feature 建議。在提交 Issue 之前，請確保已經搜索過類似的問題，因為它們可能已經得到解答或正在被修復。對於 Bug 報告，請包含可用於重現問題的完整步驟。對於新 Feature 建議，請指出你想要的更改以及期望的行為。

## 提交 Pull Request

### 共建流程

- 認領 issue：在 Github 建立 Issue 並認領（或直接認領已有 Issue），告知大家自己正在修復，避免重複工作。
- 項目開發：在完成準備工作後，進行 Bug 修復或功能開發。
- 提交 PR

### 準備工作

- [Rust](https://tauri.app/v1/guides/getting-started/prerequisites/): 請自行根據官網步驟安裝 rust 環境。
- [Node.js](https://nodejs.org/en/): 用於運行項目。
- [Pnpm](https://pnpm.io/)：本項目使用 Pnpm 進行包管理。

### 下載依賴

```shell
pnpm install
```

### 啟動應用

```shell
pnpm tauri dev
```

### 打包應用

> 如果需要打包後進行調試，請在以下命令後面加上 `--debug`

```shell
pnpm tauri build
```

## Commit 指南

Commit messages 請遵循[conventional-changelog 標準](https://www.conventionalcommits.org/en/v1.0.0/)。

### Commit 類型

以下是 commit 類型列表:

- feat: 新特性或功能
- fix: 缺陷修復
- docs: 文檔更新
- style: 代碼風格更新
- refactor: 代碼重構，不引入新功能和缺陷修復
- perf: 性能優化
- chore: 其他提交

期待您的參與，讓我們一起使 EcoPaste 變得更好！