# Handoff — Codex 继续补齐 Status / Audio / Search 入口

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次继续把剩余高频入口接进统一 `execution` envelope：

### `transport.status`
- `status`

### `transport.volume`
- `volume get`
- `volume set`

### `transport.mute`
- `mute get`
- `mute on`
- `mute off`
- `mute toggle`

### `music.spotify`
- `search spotify`

说明：
- `search spotify` 默认是 `execution.operation = "search"`
- 如果带 `--open`，则是 `execution.operation = "search_open"`
- 如果带 `--enqueue`，则是 `execution.operation = "search_enqueue"`

---

## 为什么改

上一轮已经补齐了：

- `transport`
- `transport.mode`
- `transport.source`
- `favorites`
- `open/enqueue`
- `play spotify`
- `ncm`
- `smapi`

但如果：

- `status`
- `volume`
- `mute`
- `search spotify`

还停留在旧风格 JSON，那么上层 agent 仍然无法把“查状态 / 调音量 / 静音 / 搜索”跟其他执行动作放进同一套协议里。

这次补完后，用户最常用的这批命令基本都已经能按统一结构返回：

- action
- capability
- operation
- target
- request
- result

对接时不再需要针对每条命令单独猜 JSON 形状。

---

## 结果细节

### `status`

统一到：

- `capability = "transport.status"`
- `operation = "get"`

除了保留原有：

- `device`
- `transport`
- `position`
- `nowPlaying`
- `albumArtURL`
- `volume`
- `mute`

还会在 `execution.result` 里给出摘要：

- `speaker`
- `ip`
- `state`
- `track`
- `uri`
- `time`
- `duration`
- `title`
- `artist`
- `album`
- `volume`
- `mute`

### `volume`

统一到：

- `capability = "transport.volume"`
- `operation = "get"` / `"set"`

### `mute`

统一到：

- `capability = "transport.mute"`
- `operation = "get"` / `"set"` / `"toggle"`

其中：

- `mute.on`
- `mute.off`

都会走 `operation = "set"`，具体值放在 request/result 里。

### `search spotify`

统一到：

- `capability = "music.spotify"`

operation 根据行为区分：

- `search`
- `search_open`
- `search_enqueue`

这样上层 agent 可以明确知道这次只是搜索，还是“搜索后直接入队/播放”。

---

## 这次顺手补的标准化

`search.go` 里原本就已经开始往 machine-readable errors 靠拢，这次没有回退，继续沿当前状态前进。

也就是说，`search spotify` 现在同时具备：

1. 更稳定的错误码路径
2. 更稳定的成功输出 envelope

这两者对 agent 对接都是有价值的。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*Status|Test.*Volume|Test.*Mute|Test.*SearchSpotify'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `status` JSON 输出带 `execution`
- `volume get/set` JSON 输出带 `execution`
- `mute get/toggle` JSON 输出带 `execution`
- `search spotify` JSON 输出带 `execution`
- `search spotify --open` JSON 输出带 `execution`

---

## 当前限制

1. 这次仍然是“协议标准化”，不是音乐服务可靠性修复
   - 即：更利于 agent 调用和判断
   - 不代表所有来源都因此变得可播

2. `search spotify` 现在虽然标准化了，但本质仍是 Web API 搜索 + Sonos 入队
   - 不是 Sonos SMAPI 搜索
   - 与 `play spotify` 的来源路径不同

3. 还有一批入口未接入 envelope：
   - `discover`
   - `doctor`
   - `group`
   - `watch`
   - `config`
   - `auth`

---

## 下一步建议

建议继续按下面顺序：

1. 把 `discover / doctor / group / watch / config / auth` 接进 `execution`
2. 正式写 `request/response/error schema`
3. 把播放 runbook 逐步落成可执行恢复策略
4. 再推进 `snapshot / restore / undo / auto-heal`

---

## 给后续 agent 的一句话

> 高频播放与状态入口现在基本都已统一到 execution envelope；后续重点应转向剩余入口收口、正式 schema、以及把排障经验转成可执行恢复能力。
