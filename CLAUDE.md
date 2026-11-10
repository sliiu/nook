# Pixel Control Center

## 产品要点

- MyGo 菜单栏浮层；设置独立窗口。
- v1：输出音量 + 内建亮度；Home 为二维点控（X=音量，Y=亮度，4×4 动画态）。
- 场景 HTML 经 `scene-host` 挂载并注入 `Control` / `events`，**不用 iframe srcdoc 跑可执行场景**。
- 授权默认提案档：可写代码与测试，不部署、不发版。

## 派工

路由见 `~/.claude/workflow/routing.json`。实现走 `feature` → magpie `cursor/grok-4.7@500k`。本会话若无 `delegate_task`，经用户授权可在本会话落地。

## 参考

- 计划：`~/.claude/plans/elegant-plotting-patterson.md`
- 旧音量壳模式：`/Users/pax/Documents/Code/volume` `feat/a13-home-night`（勿整仓迁入）
