# 离线下载器 · MoeKoe Music 插件（macOS / Windows）

移植版离线下载器：单曲 + 歌单 + 榜单批量下载到本地，支持 标准 / 高品 / 无损 / Hi-Res 四档音质；本地宿主直接写盘，不弹保存框，存完离线可播。

> 本项目基于 [gaozil996/MoeKoeMusic-OfflineDownloader](https://github.com/gaozil996/MoeKoeMusic-OfflineDownloader)（Windows 原版）移植。本地宿主用 Go 重写，提供 darwin/arm64、darwin/x64 与 win32/x64 二进制；取流走客户端内置 `kugouapi` 通道。

## 功能

- **单曲下载**：播放栏下载图标打开面板，下载当前歌曲；
- **歌单批量下载**：识别当前歌单或从「我的歌单」选择，勾选后队列批量下载；
- **榜单批量下载**：发现页榜单按当前高亮榜单，分页拉取曲目后批量下载；
- **四档音质**：标准（128k）/ 高品（320k）/ 无损（FLAC）/ Hi-Res，接口给什么流就存什么格式；
- **本地宿主直接写盘**：通过插件自带本地程序落盘，不走浏览器下载机制，不弹保存框；
- **已下载标记**：歌曲列表里已在磁盘上的曲目挂绿色「已下载」徽章；
- 默认保存目录：`~/Music/MoeKoeMusic`（Windows 为 `%USERPROFILE%\Music\MoeKoeMusic`）。

## 环境要求

- macOS（Apple 芯片 arm64 / Intel x64）或 Windows x64；
- MoeKoe Music 客户端 ≥ 1.7.0；
- 插件取流走客户端内置 API 与本机登录态，需先在客户端正常登录、能播放歌曲。

## 安装

1. 完全退出 MoeKoe Music（Dock 右键 → 退出，不是关窗口）。
2. 把本仓库内容放到插件目录：
   ```
   ~/Library/Application Support/moekoemusic/extensions/offline-downloader-mac/
   ```
   （`extensions` 目录不存在就新建；文件夹名可改，关键是 `manifest.json` 在文件夹内。）
3. 给本地宿主赋权、清除隔离属性：
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
├── manifest.json          # 插件清单（ID / 权限 / 本地宿主声明）
├── background.js
├── bridge.html / bridge.js
├── main-bridge.js        # 主世界桥接（隔离世界转发内置 kugouapi）
├── content.js / local-index.js
├── styles.css
├── host/
│   ├── main.go           # 本地宿主 Go 源码
│   ├── go.mod
│   ├── offline-dl-saver          # darwin/arm64 已编译二进制（M 芯片）
│   ├── offline-dl-saver-amd64    # darwin/x64 已编译二进制（Intel）
│   └── moekoe-saver.exe          # win32/x64 已编译二进制
└── icons/
```

## 重新编译本地宿主

需要自己重编时（Go 1.23+）：

```bash
cd host
# M 芯片 arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o offline-dl-saver main.go
# Intel x64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o offline-dl-saver-amd64 main.go
# Windows x64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o moekoe-saver.exe main.go
```

## 协议与致谢

- 原版作者：[SHIYI / gaozil996](https://github.com/gaozil996/MoeKoeMusic-OfflineDownloader)
- 本移植版按原项目 **MIT License** 发布，保留原版权声明；详见 [LICENSE](LICENSE)。
- 客户端：[MoeKoeMusic](https://github.com/MoeKoeMusic/MoeKoeMusic)

## 更新日志

### 1.1.0

**新增**
- 发现页榜单批量下载：按当前高亮榜单分页拉取曲目后批量下载；
- Windows x64 本地宿主（Go 重写，跨平台源码）；
- 面板标题栏可拖动；
- 歌单 / 榜单分页 2 路并发拉取，翻页间隔缩短。

**改动**
- 取流改为客户端内置 `kugouapi` 通道（隔离世界经 `main-bridge.js` 桥接主世界）；
- 登录票据自动从本地存储解析并随请求带上；
- 返回结构归一化，兼容内置通道与旧 HTTP 两种形态。

**修复**
- 榜单曲目主 hash 落在嵌套 `deprecated.hash` 时取不到的问题；
- 补全歌手（`author_name` / `authors`）、封面（`union_cover`）字段解析；
- 文件名剥离已自带的「歌手 - 」前缀，避免落盘文件名里歌手重复两次。

**兼容**：macOS arm64 / x64、Windows x64；MoeKoe Music 客户端 ≥ 1.7.0。

## 已知限制

- Windows 仅提供 x64 宿主；32 位（ia32）安装包暂不支持，且 Windows 端尚未真机实测下载；
- 下载格式由接口返回的流决定：标准/高品为 MP3，无损/Hi-Res 为 FLAC，本地不做二次转码；
- 客户端大版本若改动插件接口、页面结构或内置 API 端口，本插件可能需要跟随适配。
