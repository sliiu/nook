# Nook

菜单栏系统控制中心（MyGo）。v1：输出音量 + 内建屏亮度；HTML 场景驱动界面。

## 状态

M3：出厂二维点控 Home（X=音量，Y=亮度，4×4 态）+ `scene-host`。无内建屏时亮度轴降级。设置页场景 CRUD 尚未接入。

## 安装

从 [Releases](https://github.com/sliiu/nook/releases) 下载最新的 `.dmg`，拖入「应用程序」。应用未签名/公证，首次打开请右键 → 打开。

## 开发

需要 Go 1.27.3+，macOS 上构建需 `CGO_ENABLED=1`。

```bash
go mod tidy
CGO_ENABLED=1 go build -o pixel-control-center .
CGO_ENABLED=1 go tool mygo build   # 产出 dist 下的 .app（若已装 mygo CLI）
```

## 信任与风险

- 自定义场景 HTML 可调用系统控制桥，与个人本地场景同信任级。
- 亮度将使用私有 DisplayServices API，系统升级可能失效。

## 灵感致谢

像素房间视觉参考 [slider-resident](https://ludion.ai/p/slider-resident)（MIT，@Lattice_Node）。出厂场景为原创二维点控，不整文件复用。

## 支持

如果 Nook 对你有帮助，欢迎请我喝杯咖啡：

[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-pax__z-FFDD00?logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/pax_z)
