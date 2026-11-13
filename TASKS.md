# 任务板

| 编号 | 任务 | 执行者 | 状态 | PR | 备注 |
|---|---|---|---|---|---|
| M0 | 仓库骨架：go mod / mygo.json / 空 tray 浮层 / 文档 | self | ✅ 完成 | — | `go build` 通过；二进制可启动 |
| M1 | 输出音量后端 + Control 桥 + short 浮层 | self | ✅ 完成 | — | CoreAudio 往返 OK；待手动键控 short |
| M2 | 内建屏亮度后端 + GetState / state-changed | self | ✅ 完成 | — | 本机仅外接屏 → available=false；有内建时走 DS/CD |
| M3 | 出厂二维点控 Home + 16 态动画 + scene-host | self | ✅ 完成 | — | 更完整像素房；亮度不可用时锁 Y |
| M4 | 设置页场景 CRUD / 导入 / RestoreHome | self | 🔄 进行 | — | 存储+桥+托盘菜单+设置页已写；`go test` 通过、设置页无头渲染 OK；设置页已按新风格重做（纸感像素风，预览在左）；打包命令 `CGO_ENABLED=1 go run github.com/egoist/mygo/cmd/mygo build` 成功，.app 已启动；导入对话框、托盘切换、浮层热切换待手测 |
| M5 | 打磨、打包、stub 编译、验收清单 | — | ⬜ 待办 | — | |
| M6 | 场景市场：清单仓库 + 沙箱安装 | self | 🔄 进行 | — | 客户端已写；`home` 已上架（nook-scenes#1）；目录页已合并（nook-scenes#2）；哈希与预览检查已合并（nook-scenes#4，5fb4116），Pages 已开 |
| M7 | 应用自更新 | self | 🔄 进行 | — | 代码已接 MyGo 签名更新；secret 已设；代码未提交；下一版仍要手动装 |

## 已定：场景市场（2026-11-10）

- 审核在公开清单仓库 `sliiu/nook-scenes` 的 PR 上，合并才上架；Action 预审危险写法。不做账号后台。
- 应用只安装清单里的包，核对 commit 与 HTML 哈希；入口是 `nook://scene/<id>`。
- 市场场景走 sandbox iframe + 窄桥；本地导入维持同文档 `scene-host`。
- 清单仓库不放进本仓库。2026-11-11 已建公开仓库 `sliiu/nook-scenes`。`home` 经 nook-scenes#1 上架。目录页经 nook-scenes#2 合并。home 的 `htmlSha256` 和预览检查经 nook-scenes#4 合并，`main` 是 `5fb4116`，页面在 `https://sliiu.github.io/nook-scenes/`。
- 更新不需要新机制：应用按市场 id 认场景，同一个 id 再装一次就是覆盖；`nook://scene/<id>` 不带 commit，装的是 `main` 的最新。作者改原目录、提 PR、合并即可。
- 版本号收成三段数字（`1.2.3`）。清单审核和应用安装用同一条规则，已上架的 `home`（`1.0.0`）不受影响。清单 README 写明页面只拿得到音量和亮度。改动在 `nook-scenes` 的 `docs/scene-permissions`，还没提 PR。
- 产品入口放在 `nook` 仓库的 Pages，地址 `sliiu.github.io/nook/`，一页放介绍、下载和场景卡片。场景数据留在 `nook-scenes`，页面发布时写入场景 id 和最新 `.dmg`，浏览器不再请求 `api.github.com`。`nook-scenes` 的 Pages 待入口上线后停掉。

## 已定：应用自更新（2026-11-13）

- 走 MyGo 签名更新，不自己拉 GitHub。`mygo.json` 的 `updates.github` 是 `sliiu/nook`，公钥已写入。
- 签名私钥只在本机 `~/.config/mygo/update-keys/mygo-update.key`，不进仓库。仓库 secret `MYGO_UPDATER_PRIVATE_KEY` 已于 2026-11-13 写入。
- `release.yml` 先建 draft，再 `mygo build -upload`。发布 draft 之后，已装的版本才会收到更新。
- 更新窗口的说明取 `CHANGELOG.md` 里 `## <version>` 那一段。文件在，缺这段构建会失败。
- 已发出的 0.1.4 没有公钥，收不到更新。带更新器的下一版仍要手动装 dmg。
- 应用未公证。更新包用我们的密钥签名，第一次打开仍要右键 → 打开。

## 已定：统一控制接口（2026-11-10）

- 包 `internal/controls`。四种形状：level（0–100；音量和麦克风带静音，亮度和键盘灯不带）、switch（Wi-Fi）、choice（默认输出设备、深色模式）、action（媒体上一首 / 播放暂停 / 下一首）。
- 音量、亮度、麦克风、键盘灯、输出设备、媒体键、深色模式、Wi-Fi 都有后端。蓝牙和专注先不进目录：macOS 26 上蓝牙的读写调用会中止进程，专注的服务连接会被系统拒绝。
- 麦克风是 `audio.NewInput`：macOS 走默认输入设备的 CoreAudio 输入端；没有输入设备时控制项不可用，不阻止启动。只读冒烟是 `cmd/controlsmoke`。
- 场景用 `ListControls` / `SetControl` 读和写整个目录。`GetVolume` / `GetBrightness` 保留。市场沙箱也能调到目录里的每一项。
- 自己写的和导入的场景与应用同权限，可以关 Wi-Fi、切输出设备、改外观。
