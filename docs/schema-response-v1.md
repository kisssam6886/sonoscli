# Schema Response v1

时间：2026-03-19  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 目标

这份文档正式化当前仓库已经落地的响应协议。

重点不是把所有命令输出强行改成完全一样，而是冻结上层 agent 可以稳定依赖的最小公共结构：

- 顶层 `action`
- `execution` envelope
- 成功 / 流式事件 / 特例响应的差异

---

## 1. 标准成功响应

当前绝大多数接入 execution envelope 的 JSON 成功响应，采用以下结构：

```json
{
  "ok": true,
  "action": "queue.play",
  "execution": {
    "version": "v1",
    "capability": "queue",
    "operation": "play",
    "status": "completed",
    "target": {
      "room": "客厅"
    },
    "request": {
      "pos": 1
    },
    "result": {
      "pos": 1
    }
  },
  "pos": 1
}
```

### 字段定义

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ok` | bool | 标准成功响应固定为 `true` |
| `action` | string | 命令级动作名，例如 `queue.play`、`ncm.play` |
| `execution` | object | 稳定 execution envelope |
| 其他顶层字段 | any | 保留的命令专属业务字段，不同命令各自不同 |

---

## 2. `execution` envelope

`execution` 是 v1 最核心的稳定层：

```json
{
  "version": "v1",
  "capability": "music.netease",
  "operation": "play",
  "status": "completed",
  "target": {
    "room": "客厅",
    "speakerIP": "192.168.1.23"
  },
  "request": {
    "query": "郑伊健 心照",
    "category": "tracks",
    "limit": 10,
    "index": 1
  },
  "result": {
    "selectedID": "SONG:123",
    "selectedTitle": "心照",
    "playableHits": 8,
    "queuedTracks": 8,
    "playedPosition": 1
  }
}
```

### 固定字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version` | string | 当前固定为 `v1` |
| `capability` | string | 能力域，例如 `queue`、`transport.mode`、`music.smapi` |
| `operation` | string | 能力域下的操作名，例如 `play`、`set`、`search` |
| `status` | string | 当前值为 `completed` 或 `error` |
| `target` | object | 目标摘要；空时省略 |
| `request` | object | 请求摘要；空时省略 |
| `result` | object | 结果摘要；空时省略 |

### 稀疏输出规则

与 request schema 一样，`target/request/result` 都是稀疏对象。

以下内容会被省略：

- 空字符串
- `nil`
- 空数组
- 空 map

---

## 3. 顶层额外字段的定位

v1 里，agent 可以把响应分成两层来理解：

### 稳定层

- `action`
- `execution.version`
- `execution.capability`
- `execution.operation`
- `execution.status`
- `execution.target`
- `execution.request`
- `execution.result`

### 兼容层

为保持现有 CLI 行为和历史输出可用，很多命令仍保留自己的业务字段，例如：

- `items`
- `count`
- `groups`
- `favorite`
- `playback`
- `device`
- `transport`
- `position`
- `result`

这些字段仍然很有价值，但它们是“命令专属补充”，不是 v1 跨能力最小公共协议。

如果你在做跨命令、跨 agent 的统一解析，优先依赖 `action + execution`。

---

## 4. JSON Line 事件响应

`watch` 与 `schedule serve` 这类流式输出，不使用标准成功响应的 `ok: true` 结构，而使用 JSON line 事件：

```json
{
  "action": "watch.event",
  "execution": {
    "version": "v1",
    "capability": "watch",
    "operation": "event",
    "status": "completed",
    "target": {
      "room": "客厅"
    },
    "result": {
      "service": "avtransport",
      "sid": "uuid:RINCON...",
      "seq": "22"
    }
  },
  "time": "2026-03-19T10:00:00Z",
  "service": "avtransport",
  "sid": "uuid:RINCON...",
  "seq": "22",
  "vars": {
    "TransportState": "PLAYING"
  }
}
```

### JSON line 规则

1. 顶层通常没有 `ok`
2. 仍然有 `action + execution`
3. `execution.status` 可以是：
   - `completed`
   - `error`
4. 事件的完整原始信息放在顶层额外字段中

---

## 5. `doctor` 是一个明确特例

`doctor` 的 JSON 输出已经接入 execution envelope，但顶层 `ok` 语义不同于普通命令。

普通成功命令：

- `ok: true` 表示命令执行成功

`doctor`：

- `ok` 表示 doctor 报告本身是否健康

当前 `doctor` 响应形态如下：

```json
{
  "action": "doctor",
  "execution": {
    "version": "v1",
    "capability": "doctor",
    "operation": "report",
    "status": "completed",
    "result": {
      "reportOK": true,
      "warningCount": 0,
      "repoBinaryExists": true,
      "runningRepoBinary": true,
      "pathMatchesCurrent": true
    }
  },
  "ok": true,
  "version": "dev",
  "build": { "...": "..." },
  "binary": { "...": "..." },
  "capabilities": { "...": true }
}
```

所以对接时要注意：

1. `doctor` 顶层 `ok` 是报告健康度
2. 命令执行是否成功，仍然要结合 CLI 返回码判断
3. 如果要统一解析，请优先看 `action = "doctor"` 和 `execution`

---

## 6. 当前已覆盖的 capability

截至 2026-03-19，以下能力域已进入 execution envelope：

| capability | operation |
| --- | --- |
| `say` | `announce` |
| `scene` | `list`, `save`, `apply`, `delete` |
| `schedule` | `list`, `add`, `remove`, `run`, `serve` |
| `queue` | `list`, `clear`, `play`, `remove` |
| `discover` | `scan` |
| `config` | `path`, `get`, `set`, `unset` |
| `watch` | `event` |
| `doctor` | `report` |
| `transport` | `play`, `pause`, `stop`, `next`, `prev` |
| `transport.status` | `get` |
| `transport.mode` | `get`, `set` |
| `transport.source` | `play-uri`, `linein`, `tv`, `music` |
| `transport.volume` | `get`, `set` |
| `transport.mute` | `get`, `set`, `toggle` |
| `group` | `status`, `join`, `unjoin`, `solo`, `party`, `dissolve` |
| `group.volume` | `get`, `set` |
| `group.mute` | `get`, `set`, `toggle` |
| `favorites` | `list`, `open` |
| `music.netease` | `play`, `lucky` |
| `music.smapi` | `search`, `browse` |
| `music.spotify` | `open`, `enqueue`, `play`, `search`, `search_open`, `search_enqueue` |
| `auth.smapi` | `begin`, `complete` |

---

## 7. 结果字段约定

`execution.result` 当前没有一套跨全部能力的单一字段表，但已经形成一些稳定模式：

### A. 状态 / 音量类

常见字段：

- `volume`
- `mute`
- `playMode`
- `state`
- `track`
- `time`
- `duration`
- `coordinatorIP`

### B. 列表 / 分页类

常见字段：

- `count`
- `numberReturned`
- `totalMatches`
- `updateID`

### C. 选择 / 入队类

常见字段：

- `selectedID`
- `selectedTitle`
- `selectedType`
- `enqueuedPos`
- `queuedTracks`
- `playedPosition`
- `playableHits`

### D. 批量执行 / 编排类

常见字段：

- `affected`
- `groupCount`
- `deviceCount`
- `ran`
- `remaining`

agent 不应把这些字段当成“所有命令都必须有”，但可以在对应 capability 下稳定利用。

---

## 8. 解析建议

### 如果你在写统一 agent 适配层

推荐顺序：

1. 先读顶层 `action`
2. 再读 `execution.capability`
3. 再读 `execution.operation`
4. 按 capability-specific 解析 `execution.request` / `execution.result`
5. 有需要时再消费顶层兼容字段

### 如果你只想快速判断“命令成没成功”

标准 JSON 成功响应：

- 看顶层 `ok == true`

JSON line 事件：

- 没有统一 `ok`
- 看 `execution.status`

`doctor`：

- 顶层 `ok` 不是“命令成功”，而是“doctor 报告健康度”

---

## 9. 当前边界

1. 当前错误响应还没有统一进入 `execution` envelope
2. 不是所有 JSON 命令都强制带同一批顶层业务字段
3. v1 正式冻结的是最小公共协议，不是最终产品协议

错误部分请配合：

- `docs/schema-errors-v1.md`

一起看。
