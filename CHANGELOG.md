# 离线下载器 1.2.2

## 新增

- **「保存到」行新增「恢复默认」按钮**。1.2.1 改过目录后没有任何入口回得去，只能开控制台
  `localStorage.removeItem('odl_save_dir')`。现在点一下就清掉覆盖值，回落到默认目录：
  macOS / Linux 为 `~/Music/MoeKoeMusic`，Windows 为 `D:\MoeKoeMusic`。
- **按钮按状态显隐**：没有覆盖值时按钮隐藏（不满意的路径不会一直挂在那里），改过目录或改过
  外接盘后自动出现；恢复默认后立即消失。
- **两步确认**：第一次点「恢复默认」只把按钮变成「确认恢复？」，3 秒内再点一次才真正清除，
  防止误点把外接盘路径一键抹掉。

## 修复

- 系统文件夹选择框点「取消」时补一句状态提示「已取消，保存目录未变：<当前路径>」。
  1.2.1 是完全静默的，容易被当成按钮没反应。
- 目录提示不再长期占用下载进度行：改为走状态行样式，3 秒后自动退回「就绪」，
  期间若有新进度或仍在下载中则保留原文案。- 「选择」「恢复默认」两个按钮加 `flex: none` 保护，长路径下不再被压成两行。
- 歌单列表取 `userid` 改为复用 `myUid()`：`MoeData` 被清空或损坏时不再抛异常导致列表加载失败。

# 离线下载器 1.2.1

## 新增

- **弹窗内可改保存目录**。点「选择」调起系统文件夹对话框，可直接选外接盘 / U 盘。
  1.2.0 的保存目录写死在 `content.js` 的 `var SAVE_DIR = '~/Music/MoeKoeMusic'`，
  UI 上没有入口，只能自己改源码。
- **默认目录按平台区分**：macOS / Linux 为 `~/Music/MoeKoeMusic`，Windows 为 `D:\MoeKoeMusic`
  （与其它原版 Windows 插件保持一致）。
- **选择结果持久化**：写入 `localStorage['odl_save_dir']`，重启 App 后保留，
  优先级高于默认值。
- **单曲页签补上「保存到」一行**。1.2.0 只在「歌单批量」页签渲染该行。
- **宿主程序目录回退**：目标目录建不出来（典型场景是只有 C 分区的机器上默认
  `D:\MoeKoeMusic`）时，自动回退到 `~/Music/MoeKoeMusic`，不再直接报
  `MkdirAll` 失败。改动在 `host/main.go` 的 `cmdDownload` 与 `cmdSaveText` 两处。

## 修复

- 保存目录过长时被 flex 布局压成竖排单字的问题（`styles.css`）。
- `manifest.json` 的 `linux` 平台原先指向 `host/offline-dl-saver` 这个 Mach-O arm64
  二进制，Linux 上无法执行；现改为独立的 `host/offline-dl-saver-linux`（ELF x86-64）。

## 交互约定

- 取消系统文件夹对话框 = 静默，不改动当前目录，也不弹输入框。
- 只有对话框弹不出来时，才回退为手动路径输入。
- Windows 侧依然可以手输盘符路径（如 `E:\Music`）。

## 三平台宿主程序

全部用 Go（纯标准库，`CGO_ENABLED=0`）重新编译，均包含目录回退逻辑：

| 文件 | 平台 | 说明 |
|---|---|---|
| `host/offline-dl-saver` | darwin / arm64 | Apple Silicon，含 ad-hoc 代码签名 |
| `host/offline-dl-saver-amd64` | darwin / amd64 | Intel Mac，无需签名 |
| `host/offline-dl-saver-linux` | linux / amd64 | ELF x86-64 |
| `host/moekoe-saver.exe` | win32 / amd64 | PE32+ |

darwin/arm64 的 ad-hoc 签名由 Go 链接器写入（纯 Go 实现，交叉编译时同样生效），
已校验 `LC_CODE_SIGNATURE` 存在、`flags = 0x20002`（CS_ADHOC | CS_LINKER_SIGNED）、
SHA256 / pageSize 4096 / 1293 页哈希全部匹配。

## 重新编译

```bash
# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o moekoe-saver.exe main.go
# macOS Apple Silicon
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w" -o offline-dl-saver main.go
# macOS Intel
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w" -o offline-dl-saver-amd64 main.go
# Linux
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w" -o offline-dl-saver-linux main.go
```
