# Handoff — Codex 继续补齐 Source/Spotify 入口

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次把上一轮还没收口的常用播放入口，也统一接进 `execution` envelope：

### `music.spotify`
- `play spotify`

说明：
- 默认执行时，`execution.operation = "play"`
- 如果带 `--enqueue`，则 `execution.operation = "enqueue"`

### `transport.source`
- `play-uri`
- `linein`
- `tv`
- `music`

这些命令在 `--format json` 下，现在都会稳定返回：

- `ok`
- `action`
- `execution.version`
- `execution.capability`
- `execution.operation`
- `execution.status`
- `execution.target`
- `execution.request`
- `execution.result`

---

## 为什么改

上一轮已经把：

- `transport`
- `transport.mode`
- `favorites`
- `open/enqueue`

补进统一输出，但如果：

- `play spotify`
- `play-uri`
- `linein`
- `tv`
- `music`

还停留在旧风格 JSON，上层 agent 仍然要混着写多套解析逻辑，离真正的 execution layer 还差一截。

这次补完后，常见“播放入口”已经基本进入统一格式：

- 直接控播
- 模式切换
- 来源切换
- Spotify 入口
- Favorites

后续 agent 对接时，可以更稳定地按 capability / operation 分流，而不用靠命令名硬猜。

---

## 结果细节

### `play spotify`

统一到：

- `capability = "music.spotify"`
- `operation = "play"` 或 `"enqueue"`

结果里会包含：

- 解析到的 Sonos 音乐服务信息
- `speakerIP`
- `coordinatorIP`
- `selectedID`
- `selectedTitle`
- `selectedType`
- `enqueuedPos`
- `titleEffective`

### `play-uri`

统一到：

- `capability = "transport.source"`
- `operation = "play-uri"`

请求和结果会区分：

- 原始传入 URI
- 最终实际写入 Sonos 的 URI

这样如果启用了 `--radio`，agent 可以看出 URI 已被转换成 radio 形态。

### `linein / tv / music`

统一到：

- `capability = "transport.source"`
- `operation = "linein" / "tv" / "music"`

结果里会附上解析后的成员摘要：

- `name`
- `ip`
- `uuid`

方便上层 agent 做日志、提示和恢复策略。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*PlaySpotify|Test.*PlayURI|Test.*LineIn|Test.*TVCmd|Test.*MusicCmd'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `play spotify` JSON 输出带 `execution`
- `play spotify --enqueue` JSON 输出带 `execution`
- `play-uri` JSON 输出带 `execution`
- `linein` JSON 输出带 `execution`
- `tv` JSON 输出带 `execution`
- `music` JSON 输出带 `execution`

---

## 当前限制

1. 这次仍然是“协议统一”工作，不是“音乐版权/版本可播性”修复
   - 即：agent 更容易稳定接入
   - 但不代表所有 Sonos 服务资源都一定能正常播放

2. `transport.source` 只是把入口标准化了
   - 不代表 `play-uri` 已适合拿来当网易云默认播放路径
   - 网易云默认路径仍应坚持 `SMAPI 入队后播`

3. 还有一些高频命令未接入 envelope：
   - `status`
   - `volume`
   - `mute`
   - `search spotify`
   - 以及部分运维/发现类入口

---

## 下一步建议

建议按下面顺序继续：

1. 把 `status / volume / mute / search spotify` 接入 `execution`
2. 正式写 `request/response/error schema`
3. 把播放稳定性 runbook 逐步沉淀成可执行恢复策略
4. 再推进 `snapshot / restore / undo / auto-heal`

---

## 给后续 agent 的一句话

> 常用播放入口现在大部分已经统一进 execution envelope；后续优先继续补协议、状态恢复和自动修复，不要再把 repo 拉回零散命令集合。
