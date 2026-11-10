# Pixel Control Center

菜单栏系统控制中心（MyGo）。v1：输出音量 + 内建屏亮度；HTML 场景驱动界面。

## 状态

M0 骨架：托盘 + 360² 浮层 + 设置窗占位。音量/亮度/二维点控尚未接入。

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
