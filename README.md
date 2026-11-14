# 离线下载器 (macOS) · MoeKoe Music 插件

macOS 移植版离线下载器：单曲下载 + 歌单批量下载到本地，支持 标准 / 高品 / 无损 / Hi-Res 四档音质；本地程序直接写盘，不弹保存框，存完离线可播。

> 本项目基于 [gaozil996/MoeKoeMusic-OfflineDownloader](https://github.com/gaozil996/MoeKoeMusic-OfflineDownloader)（Windows 版）移植。原版本地写盘程序为 Windows 程序，本版用 Go 重写为 darwin/arm64 + amd64，并适配了 macOS 下载目录与客户端内置 API 端口。

## 功能

- **单曲下载**：播放栏下载图标打开面板，下载当前歌曲；
- **歌单批量下载**：识别当前歌单或从下拉选择我的歌单，勾选后队列批量下载；
- **四档音质**：标准（128k）/ 高品（320k）/ 无损（FLAC）/ Hi-Res，接口给什么流就存什么格式；
- **本地程序直接写盘**：通过插件自带本地程序落盘，不走浏览器下载机制，不弹保存框；
- **已下载标记**：歌曲列表里已在磁盘上的曲目挂绿色「已下载」徽章；
- 默认保存目录：`~/Music/MoeKoeMusic`。

## 环境要求

- macOS（Apple 芯片 arm64 / Intel x64 均可）；
- MoeKoe Music 客户端 ≥ 1.7.0；
- 插件取流走客户端内置 API 与本机登录态，需先在客户端正常登录、能播放歌曲。

## 安装

1. 完全退出 MoeKoe Music（Dock 右键 → 退出，不是关窗口）。
2. 把本仓库内容放到插件目录：
   ```
   ~/Library/Application Support/moekoemusic/extensions/offline-downloader-mac/
   ```
   （`extensions` 目录不存在就新建；文件夹名可改，关键是 `manifest.json` 在文件夹内。）
3. 给本地程序赋权、清除隔离属性：
   ```bash
   EXT=~/Library/Application\ Support/moekoemusic/extensions/offline-downloader-mac
   chmod +x "$EXT/host/offline-dl-saver"
   xattr -cr "$EXT"
   ```
   > Intel 芯片：进入 `host` 目录，把 `offline-dl-saver-amd64` 改名为 `offline-dl-saver` 后再执行上面两条。
4. 重新打开客户端，进插件管理页，给本地程序 `offline-dl-saver` **授权**（首次默认未授权）。
5. 播放任意歌曲，点播放栏下载图标，状态就绪即可下载。

### 改下载目录

在面板所在页面按 `Cmd + Option + I` 打开控制台：

```js
localStorage.setItem('odl_save_dir', '/你的/绝对/目录')
```

然后刷新面板。

## 目录结构

```
├── manifest.json          # 插件清单（ID / 权限 / 本地程序声明）
├── background.js
├── bridge.html / bridge.js
├── content.js / local-index.js
├── styles.css
├── host/
│   ├── main.go           # 本地程序 Go 源码
│   ├── go.mod
│   ├── offline-dl-saver          # darwin/arm64 已编译二进制（M 芯片）
│   └── offline-dl-saver-amd64   # darwin/x64 已编译二进制（Intel）
└── icons/
```

## 重新编译本地程序

需要自己重编时（Go 1.23+）：

```bash
cd host
# M 芯片 arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o offline-dl-saver main.go
# Intel x64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o offline-dl-saver-amd64 main.go
```

## 协议与致谢

- 原版作者：[SHIYI / gaozil996](https://github.com/gaozil996/MoeKoeMusic-OfflineDownloader)
- 本移植版按原项目 **MIT License** 发布，保留原版权声明；详见 [LICENSE](LICENSE)。
- 客户端：[MoeKoeMusic](https://github.com/MoeKoeMusic/MoeKoeMusic)

## 更新日志

### v1.0.1

**问题修复**

- 修复歌单批量面板关闭按钮点击无反应的问题；
- 修复切换歌单后残留上一个歌单曲目、勾选与下载标记的问题；
- 修复面板顶部歌单名称与实际歌单不符的问题；
- 标题行不再显示内部歌单 ID。

**交互优化**

- 识别当前歌单后自动拉取曲目，省去单独拉取一步；
- 打开面板自动加载我的歌单，下拉选择后自动拉取；
- 精简冗余按钮，整理批量操作行排版。

## 已知限制

- 仅适配 macOS / Linux 平台；Windows 请使用原版插件；
- 下载格式由接口返回的流决定：标准/高品为 MP3，无损/Hi-Res 为 FLAC，本地不做二次转码；
- 客户端大版本若改动插件接口、页面结构或内置 API 端口，本插件可能需要跟随适配。
