# 阶段 0 验证报告：糖屋在场景市场沙箱中运行

日期 2026-11-16。验证对象：`sliiu/nook-scenes` 的 `scenes/candy/scene.html`（543,544 字节，three.js r170，当前有未提交改动）。

用真实的 `frontend/scene-host.js` 的 `mountMarketScene` 驱动，在无头 Edge 157 中运行。

## 结论

**可以落地。** 543 KB 的 three.js 场景在真实的目录沙箱里正常挂载、正常渲染、桥调用全部通达。

## 通过项

| 检查 | 结果 |
|---|---|
| iframe 挂载 | ✅ 挂载成功，`sandbox` 恰为 `allow-scripts` |
| 不透明源 | ✅ 父页面读不到 frame 的 DOM（`contentDocument` 为 null） |
| 场景脚本执行 | ✅ 场景根节点、canvas 480×540 均渲染 |
| WebGL | ✅ `WebGL 2.0 (OpenGL ES 3.0 Chromium)` 上下文存活 |
| 实际绘制 | ✅ 截图取证，房间完整渲染（见 `stage0-sandbox-shot.png`） |
| 尺寸握手 | ✅ 首帧 360×360 → 收到 `size:480x540` 后改为 480×540，两次一致 |
| `localStorage` | ✅ 抛 `SecurityError`（沙箱缺 `allow-same-origin`） |
| `sessionStorage` | ✅ 同样被拒 |
| `eval` / `new Function` | ✅ 被 CSP 拒绝（`script-src 'nonce-…'`） |
| 桥注入 | ✅ `Control`、`events`、`stateChanged.on`、`pillForm.on` 都在 |
| 桥调用 | ✅ `listControls` 穿过桥；场景订阅了 `stateChanged` |
| 音量/亮度读数 | ✅ 读到 `音量 42 · 亮度 66 · 外观 自动` |

## 审核扫描

复刻 `internal/market` 的 `unsafeScript`，对相关产物逐一扫描：

| 产物 | 体积 | 判定 |
|---|---|---|
| 现有 `candy/scene.html` | 543 KB | ✅ PASS |
| `home/scene.html` | 104 KB | ✅ PASS |
| `three.core.min.js` | 381 KB | ❌ `fetch(` |
| `three.module.min.js` | 339 KB | ✅ 但沙箱跑不了（丢 `type`） |
| esbuild 全量 IIFE | 704 KB | ❌ `fetch(` + `parent.` |
| esbuild 按需 IIFE | 514 KB | ❌ `parent as window` |
| 按需 + `--mangle-props=^parent$` | 500 KB | ✅ PASS |

现有场景用的是去掉 `export` 尾部 + 局部 `const T = {...}` 绑定的做法，已验证 PASS，无需改动。

## 发现的真实缺口（已修复，回归通过）

回归脚本 `fixes.py` 两条路径全绿，结果见文末「修复回归」。

### 1. 浮层隐藏后渲染循环不停

**更正一处此前的误读**：`pillForm` 订阅数在第一次探测里报成 1，那是 `scene-host.js` 自己的 `MARKET_EVENTS` 循环在订阅，不是场景。场景侧的实际订阅数是 **0**——`grep -c pillForm` 对场景源码、构建产物、仓库副本都返回 0，场景只订阅了 `events.stateChanged.on`。桥用 `Set` 分发，宿主订阅与场景订阅互不覆盖，这一点已经实测确认。

所以缺口成立：场景从不获知浮层已隐藏，`requestAnimationFrame` 会一直跑满帧。对比 `pixel-control-center/frontend/scenes/home.html:2574` 有 `visibilitychange` 处理。

**修法**：订阅 `events.pillForm`，`"hidden"` 停 `draw()`，其余形态恢复；再挂 `visibilitychange` 兜底，`webglcontextlost` 也接到同一个 `stop()`。

### 2. `setPointerCapture` 无保护

构建产物里是 `Tr.setPointerCapture(K.pointerId)`，没有 try/catch（源文件 `scene.mjs:417` 有，构建版没有）。指针在边缘丢失时会抛 `NotFoundError`，打断 `pointerdown` 后续逻辑（含 `noteDrag(true)`），导致浮层拖拽期间被误收起。

**修法**：包 try/catch；把 `pointerup` 的内联提交体抽成 `commit(d, tap)`，由 `pointerup` / `lostpointercapture` 走提交路径，`pointercancel` 走中止路径（`endPointer(false)`，不发任何控制写入，只把手势收尾并把百叶帘退回当前外观的目标位）。之前还留过一个重复的 `pointerup` 监听，会让一次手势提交两遍，一并删掉。

### 3. 挂载/卸载无清理钩子

场景自身没有 `dispose()` / `forceContextLoss()`（`grep` 命中的那 47 个 `dispose` 在 three.js 内部，不是场景自己的）。同一文档挂载时反复切换场景会累积 WebGL 上下文。

**修法**：`disposeScene()` 遍历 `scene` 释放几何与材质贴图，再 `renderer.dispose()` + `forceContextLoss()`。

**这里有一个我自己写的 bug，被回归测出来了**：最初用 `MutationObserver` 观察 `document.body` 的 `childList`，判断 `!root.isConnected`。但 `clearHost()` 清的是**宿主节点**（`host.replaceChildren()`），`body` 的直接子节点从未变动，所以在真正生效的同文档挂载路径上观察器永远不触发，`disposeScene()` 一次都不会跑。改成观察 `root.parentNode`。修完后 `isContextLost()` 由 `false` 变 `true`，实测通过。

## 物件与控制项的现状

现有场景的映射和你这次的要求**不一致**：

| 物件 | 现有场景 | 你的要求 |
|---|---|---|
| 吊灯 | `wifi.power` | **`display.level`（内屏亮度）** |
| 地板拖拽 | 音量（X）+ 亮度（Y） | 未提 |
| 窗帘 | `appearance`（浅色/深色/自动） | **百叶帘，控制外观** |
| 音响 | `output.level` + 点按静音 | 一致 |
| 麦克风 | `input.level` + 点按静音 | 一致 |
| 唱机三键 | `media.previous/toggle/next` | 一致 |
| 路由器 | **不存在** | **新增，`wifi.power`** |

也就是说，要按你的方案落地，需要：吊灯改接 `display.level`、新增路由器接管 `wifi.power`、窗帘改成百叶帘形态。

## 修复回归

源码改完重新打包，产物 **544,840 字节**（原 543,544，+1,296），审核扫描仍 **PASS**，六个修复标记在压缩后各命中一次（`MutationObserver`、`forceContextLoss`、`pillForm`、`visibilitychange`、`pointercancel`、`lostpointercapture`）。

两条路径分别实测：

| 路径 | 检查 | 结果 |
|---|---|---|
| 目录沙箱 | 场景挂载，`sandbox` 为 `allow-scripts`，canvas 480×540 | ✅ |
| 目录沙箱 | `"hidden"` 取消挂起的动画帧 | ✅ `cancelAnimationFrame` ×1 |
| 目录沙箱 | 重复 `"hidden"` 不再调度 | ✅ `requestAnimationFrame` ×0 |
| 目录沙箱 | 其余形态恢复渲染 | ✅ `requestAnimationFrame` ×1 |
| 目录沙箱 | 重复 show 不叠加第二条循环 | ✅ `requestAnimationFrame` ×0 |
| 目录沙箱 | 取消的手势不产生控制写入 | ✅ 只有 `noteDrag(true) → noteDrag(false) → noteIdle` |
| 目录沙箱 | 取消的手势仍然收尾（释放拖拽） | ✅ `noteDrag(false)` |
| 目录沙箱 | 同一落点的 `pointerup` 确实提交 `appearance` | ✅ 作为对照组，证明取消检查不是空过 |
| 同文档挂载 | 卸载前 `isContextLost()` | ✅ `false` |
| 同文档挂载 | 重挂后旧上下文 `isContextLost()` | ✅ `true`，上下文已释放 |
| 同文档挂载 | 新场景接替，旧 canvas 脱离宿主 | ✅ |

对照组很重要：如果取消路径和提交路径一样什么都不写，"取消不写入"就是空过。实测同一坐标 `pointerup` 写入了 `appearance {selected:"light"}`，取消则不写。

**未能验证的一项**：`pillForm` 在真实浮层里的收益（隐藏期间省下多少 CPU）。无头装置跑不动时钟——主页面 `requestAnimationFrame` 只推进 2 帧就停滞，iframe 内 `setTimeout` 一次都不触发。所以只能同步验证"取消/发起动画帧"这个机制本身，省电幅度需要装上 App 后实测。

## 沙箱限制（设计前提）

1. `localStorage` 不可用 → 百叶帘位置无法持久化，每次打开回默认。
2. `connect-src 'none'` → 一切贴图必须程序化，或 `data:` URI。
3. 1 MiB 上限 → 544,840 B 已占 52%，只剩约 503 KB 给新增建模。
4. 每次打开重新挂载 → 开场可以用百叶帘自动展开代替"记住上次"。

## 复现方式

阶段 0 探测（挂载、隔离、CSP、尺寸握手）：

```bash
cd /tmp/sandbox-test && python3 -I stage0c.py
```

```bash
"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge" --headless=new --allow-file-access-from-files --window-size=560,900 --virtual-time-budget=30000 --dump-dom "file:///tmp/sandbox-test/stage0c.html"
```

三项修复的回归：

```bash
cd /tmp/sandbox-test && python3 -I fixes.py
```

```bash
"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge" --headless=new --allow-file-access-from-files --window-size=560,900 --virtual-time-budget=30000 --dump-dom "file:///tmp/sandbox-test/fixes-a.html"
```

```bash
"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge" --headless=new --allow-file-access-from-files --window-size=560,900 --virtual-time-budget=30000 --dump-dom "file:///tmp/sandbox-test/fixes-b.html"
```

两条注意：

1. 一次浏览器启动只能测一次挂载。`--virtual-time-budget` 会被首个 srcdoc 加载耗尽，同页多挂载会假失败。
2. 这个装置**跑不动时钟**：主页面 `requestAnimationFrame` 推进 2 帧后停滞，iframe 内 `setTimeout` 完全不触发。所以回归里每个断言都必须同步成立，不能靠等待；任何"过一会儿应该怎样"的检查在这里都会假失败。我最初那版探针就栽在这上面（探针写在 `setTimeout` 里，从不执行，报成"场景没跑起来"）。
