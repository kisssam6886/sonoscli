# Execution Envelope v1 — `say / scene / schedule` 统一执行输出

时间：2026-03-18  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 1) 目的

为了让任何 agent / bot / automation 更容易稳定调用，`say / scene / schedule`
开始统一输出一层 `execution` envelope。

这个 envelope 不取代现有 JSON 字段，而是：

- 保留原有 `ok` / `action` / `item` / `results` 等字段
- 额外补一层稳定结构，方便上层 agent 统一解析

---

## 2) 统一结构

JSON 模式下，这三类命令会额外返回：

```json
{
  "ok": true,
  "action": "scene.apply",
  "execution": {
    "version": "v1",
    "capability": "scene",
    "operation": "apply",
    "status": "completed",
    "target": {
      "room": "客厅"
    },
    "request": {
      "name": "Movie Night"
    },
    "result": {
      "name": "Movie Night"
    }
  }
}
```

---

## 3) 字段定义

### `execution.version`
- 当前固定：`v1`

### `execution.capability`
- 这次统一的能力域
- 当前包括：
  - `say`
  - `scene`
  - `schedule`

### `execution.operation`
- 具体动作
- 示例：
  - `announce`
  - `list`
  - `save`
  - `apply`
  - `delete`
  - `add`
  - `run`
  - `serve`

### `execution.status`
- 当前值：
  - `completed`
  - `error`

### `execution.target`
- 执行目标
- 当前常见字段：
  - `room`
  - `ip`
  - `coordinatorIP`

### `execution.request`
- 本次调用的标准化请求摘要
- 目的是让 agent 不必从原始 CLI 参数里反推意图

### `execution.result`
- 本次调用的标准化结果摘要
- 目的是让 agent 不必只靠命令文本或零散字段判断结果

---

## 4) 当前覆盖范围

### `say`
- `say`

### `scene`
- `scene list`
- `scene save`
- `scene apply`
- `scene delete`

### `schedule`
- `schedule list`
- `schedule add`
- `schedule remove`
- `schedule run`
- `schedule serve`（JSON line event）

### `queue`
- `queue list`
- `queue clear`
- `queue play`
- `queue remove`

### `discover`
- `discover`

### `config`
- `config path`
- `config get`
- `config set`
- `config unset`

### `watch`
- `watch`（JSON line event）

### `auth.smapi`
- `auth smapi begin`
- `auth smapi complete`

### `transport`
- `play`
- `pause`
- `stop`
- `next`
- `prev`

### `transport.status`
- `status`

### `transport.mode`
- `mode get`
- `mode shuffle`
- `mode shuffle-norepeat`
- `mode repeat`
- `mode repeat-one`
- `mode normal`

### `transport.source`
- `play-uri`
- `linein`
- `tv`
- `music`

### `transport.volume`
- `volume get`
- `volume set`

### `transport.mute`
- `mute get`
- `mute on`
- `mute off`
- `mute toggle`

### `favorites`
- `favorites list`
- `favorites open`

### `music.netease`
- `ncm play`
- `ncm lucky`

### `music.smapi`
- `smapi search`
- `smapi browse`

### `music.spotify`
- `open`
- `enqueue`
- `play spotify`
- `search spotify`

---

## 5) 为什么这样做

这不是为了“JSON 更好看”，而是为了把仓库往真正的 execution layer 推进一步：

1. 上层 agent 不再需要针对每个命令写一套不同解析器
2. 后续把 `music / queue / scene / say / schedule` 收拢到统一 request schema 时，迁移成本更低
3. 可以逐步把“执行成功”的判断，从命令风格输出，切到统一结构

---

## 6) 当前边界

这次只是第一步，还没有做：

1. 一个统一的 `execute` 入口命令
2. 一个正式的 request schema 输入层
3. 所有命令都接入 envelope

当前是先把：

- `say`
- `scene`
- `schedule`
- `queue`
- `ncm`
- `smapi search/browse`

这些比较适合“场景化执行”的能力，先统一起来。

---

## 7) 下一步建议

后续建议按这个顺序继续收口：

1. `group` / `doctor` 等剩余运维入口接入同一层 `execution`
2. 定义正式 request schema：
   - `target`
   - `service`
   - `source`
   - `action`
   - `strategy`
   - `reliability`
   - `feedback`
3. 再考虑是否增加一个统一 `execute` 入口
