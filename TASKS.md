# 任务板

| 编号 | 任务 | 执行者 | 状态 | PR | 备注 |
|---|---|---|---|---|---|
| M0 | 仓库骨架：go mod / mygo.json / 空 tray 浮层 / 文档 | self | ✅ 完成 | — | `go build` 通过；二进制可启动 |
| M1 | 输出音量后端 + Control 桥 + short 浮层 | self | ✅ 完成 | — | CoreAudio 往返 OK；待手动键控 short |
| M2 | 内建屏亮度后端 + GetState / state-changed | self | ✅ 完成 | — | 本机仅外接屏 → available=false；有内建时走 DS/CD |
| M3 | 出厂二维点控 Home + 16 态动画 + scene-host | self | ✅ 完成 | — | 更完整像素房；亮度不可用时锁 Y |
| M4 | 设置页场景 CRUD / 导入 / RestoreHome | self | 🔄 进行 | — | 存储+桥+托盘菜单+设置页已写；`go test` 通过、设置页无头渲染 OK；设置页已按新风格重做（纸感像素风，预览在左）；打包命令 `CGO_ENABLED=1 go run github.com/egoist/mygo/cmd/mygo build` 成功，.app 已启动；导入对话框、托盘切换、浮层热切换待手测 |
| M5 | 打磨、打包、stub 编译、验收清单 | — | ⬜ 待办 | — | |
| M6 | 场景市场：清单仓库 + 沙箱安装 | self | 🔄 进行 | — | 客户端已写：安装、哈希校验、沙箱窄桥；清单仓库与审核 Action 未建 |

## 已定：场景市场（2026-11-10）

- 审核在公开清单仓库 `sliiu/nook-scenes` 的 PR 上，合并才上架；Action 预审危险写法。不做账号后台。
- 应用只安装清单里的包，核对 commit 与 HTML 哈希；入口是 `nook://scene/<id>`。
- 市场场景走 sandbox iframe + 窄桥；本地导入维持同文档 `scene-host`。
- 清单仓库不放进本仓库。客户端已落地，仓库本身还没建。

## 已定：统一控制接口（2026-11-10）

- 包 `internal/controls`。三种形状：level（0–100；音量和麦克风带静音，亮度不带）、choice（默认输出设备）、action（媒体上一首 / 播放暂停 / 下一首）。
- 音量、亮度、麦克风经 `FromAudio` / `FromBrightness` / `FromInput` 接入。输出设备和媒体键在目录里，还没有后端，`Apply` 返回 `ErrUnavailable`。
- 麦克风是 `audio.NewInput`：macOS 走默认输入设备的 CoreAudio 输入端；其他平台返回错误，目录里显示为不可用。冒烟程序 `cmd/micsmoke`。
- 浮层桥仍是 `GetVolume` / `GetBrightness`。观察者仍用音量的 `DefaultKey` 区分换设备。
