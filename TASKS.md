# 任务板

| 编号 | 任务 | 执行者 | 状态 | PR | 备注 |
|---|---|---|---|---|---|
| M0 | 仓库骨架：go mod / mygo.json / 空 tray 浮层 / 文档 | self | ✅ 完成 | — | `go build` 通过；二进制可启动 |
| M1 | 输出音量后端 + Control 桥 + short 浮层 | self | ✅ 完成 | — | CoreAudio 往返 OK；待手动键控 short |
| M2 | 内建屏亮度后端 + GetState / state-changed | self | ✅ 完成 | — | 本机仅外接屏 → available=false；有内建时走 DS/CD |
| M3 | 出厂二维点控 Home + 16 态动画 + scene-host | self | ✅ 完成 | — | 更完整像素房；亮度不可用时锁 Y |
| M4 | 设置页场景 CRUD / 导入 / RestoreHome | self | 🔄 进行 | — | 存储+桥+托盘菜单+设置页已写；`go test` 通过、设置页无头渲染 OK；设置页已按新风格重做（纸感像素风，预览在左）；打包命令 `CGO_ENABLED=1 go run github.com/egoist/mygo/cmd/mygo build` 成功，.app 已启动；导入对话框、托盘切换、浮层热切换待手测 |
| M5 | 打磨、打包、stub 编译、验收清单 | — | ⬜ 待办 | — | |
