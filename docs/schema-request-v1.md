# Schema Request v1

时间：2026-03-19  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 目标

这份文档正式化当前仓库里已经落地的“请求模型”。

这里说的 request，不是 `sonos execute` 的输入契约，而是当前已经稳定存在于 JSON 成功输出里的：

- `execution.target`
- `execution.request`

也就是说，v1 先冻结“当前命令已经如何向上层 agent 暴露请求语义”，而不是假设一个还没实现的新输入协议。

---

## 范围与非目标

### v1 覆盖什么

1. 统一 `execution` envelope 下的目标与请求摘要
2. 当前已经接入 envelope 的能力域
3. 当前已经真实出现在仓库里的字段命名与省略规则

### v1 不覆盖什么

1. 一个已经完整 formalize 的统一 CLI 输入层
2. 一个强制所有能力都使用同样嵌套结构的输入层
3. 旧草案里 `source/action/strategy/state` 的完整实现

说明：

1. 仓库现在已经有最小 `sonos execute` PoC
2. 但它仍是局部 capability 覆盖，不等于完整输入 schema 已冻结
3. `execute` 输入说明请看 `docs/execute-entry-poc-v1-2026-03-19.md`

旧草案仍然有参考价值，但本 v1 文档以当前代码现实为准。

---

## 统一模型

当前成功响应里的请求语义分两层：

```json
{
  "action": "ncm.play",
  "execution": {
    "version": "v1",
    "capability": "music.netease",
    "operation": "play",
    "status": "completed",
    "target": {
      "room": "客厅",
      "ip": "192.168.1.20"
    },
    "request": {
      "query": "郑伊健 心照",
      "category": "tracks",
      "limit": 10,
      "index": 1
    }
  }
}
```

### `execution.target`

表示这次动作打到了哪个 Sonos 目标。

### `execution.request`

表示这次动作的标准化请求摘要，目的是让 agent 不必从 CLI 参数字符串里反推调用意图。

---

## 规范规则

### 1. 稀疏对象

`execution.target` 和 `execution.request` 都是稀疏对象。

空值会被省略，当前实现会移除：

- 空字符串
- `nil`
- 空数组
- 空 map

因此，agent 不能要求某字段“总是存在但可能为空”；正确做法是按“存在则有值，不存在则未提供”解析。

### 2. 顶层 target 与 request 分离

当前 v1 不把 target 混进 request。

也就是说：

- 播放目标在 `execution.target`
- 业务参数在 `execution.request`

后续如果增加统一输入层，可以把两者重新组装，但现阶段对接请按当前分层读取。

### 3. 字段命名

当前命名以 lowerCamelCase 为主，例如：

- `coordinatorIP`
- `speakerIP`
- `holdSeconds`
- `titleOverride`
- `selectionAction`

少量枚举值使用已有 CLI / Sonos 语义，不在 v1 内另做重命名。

### 4. 不强制统一为 `source/action/strategy`

当前实现并没有统一输出：

- `source`
- `action`
- `strategy`
- `state`

这几个嵌套对象。

因此 v1 先固定“扁平 request 字段集”，后续再考虑是否在 v2 重构为更强约束的嵌套模型。

---

## `execution.target` 字段

以下字段已经在当前实现中出现，且可视为 v1 稳定字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `room` | string | 由 `--name` 解析出的目标房间名 |
| `ip` | string | 由 `--ip` 指定的目标 IP |
| `speakerIP` | string | 某些音乐/SMAPI 命令实际使用的扬声器 IP |
| `coordinatorIP` | string | 当前动作作用到的 group coordinator IP |

说明：

1. `room` 和 `ip` 来自通用 helper，覆盖最广
2. `speakerIP` / `coordinatorIP` 只在部分命令补充
3. 当前不会输出 UUID 级 target schema；如果需要 UUID，请依赖额外业务字段

---

## `execution.request` 字段族

当前实现里的 request 字段大致分为以下几类。

### A. 目标选择 / 分页 / 位置

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `start` | int | `queue list` / `favorites list` |
| `limit` | int | `queue list` / `favorites list` / 搜索类命令 |
| `index` | int | `ncm play` / `smapi search` / `search spotify` |
| `pos` | int | `queue play` / `queue remove` |
| `all` | bool | `discover --all` / `group status --all` |
| `id` | string | `schedule remove` / `schedule run --id` |

### B. 音乐来源 / 搜索

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `service` | object | `smapi search` / `auth smapi begin` / `play spotify` |
| `query` | string | `ncm play` / `smapi search` / `search spotify` |
| `category` | string | `ncm play` / `smapi search` / `play spotify` |
| `type` | string | `search spotify` |
| `market` | string | `search spotify` |
| `selectionAction` | string | `search spotify --open/--enqueue` |
| `titleOverride` | string | `play spotify --title` |

### C. 来源切换 / 直接播放

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `uri` | string | `play-uri` |
| `ref` | string | `open` / `enqueue` |
| `title` | string | `play-uri --title` / `say --title` |
| `radio` | bool | `play-uri --radio` / `say --radio` |
| `from` | string | `linein --from` |
| `mode` | string | `mode shuffle` / `mode repeat-one` |
| `asNext` | bool | `open --next` / `enqueue --next` |
| `playNow` | bool | `open` / `enqueue` |

### D. 音量 / 静音 / 组音量

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `volume` | int | `volume set` / `group volume set` |
| `mute` | bool | `mute on/off` / `group mute on/off/set` |

### E. 配置 / 场景 / 定时任务

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `name` | string | `scene save/apply/delete` / `schedule add` |
| `key` | string | `config get/set/unset` |
| `value` | string | `config set` |
| `action` | string | `schedule add` |
| `payload` | string | `schedule add` |
| `at` | string | `schedule add` |
| `interval` | string | `schedule serve` |
| `once` | bool | `schedule serve --once` |

### F. TTS / 播报

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `text` | string | `say` 播报文本 |
| `inputMode` | string | `auto-tts` 或 `audio-uri` |
| `tempVolume` | int | 临时播报音量 |
| `lang` | string | `zh` / `yue` 等 |
| `voice` | string | 实际或指定 voice |
| `style` | string | 风格提示 |
| `rate` | int | 语速 |
| `holdSeconds` | int | 本地临时 HTTP 保持时长 |
| `usedAudioURI` | string | 实际用于播放的音频 URI |

### G. Group 操作

| 字段 | 类型 | 典型命令 |
| --- | --- | --- |
| `to` | object | `group join` / `group party` |
| `only` | string | `scene apply --only` |

说明：

`to` 当前是一个成员摘要对象，通常包含：

- `name`
- `ip`
- `uuid`

---

## 能力域与 request 对齐表

以下是当前已接入 envelope 的能力域，以及常见 request 字段。

| capability | operation | 常见 request 字段 |
| --- | --- | --- |
| `say` | `announce` | `text`, `inputMode`, `radio`, `tempVolume`, `lang`, `voice` |
| `scene` | `list`, `save`, `apply`, `delete` | `name`, `only` |
| `schedule` | `list`, `add`, `remove`, `run`, `serve` | `name`, `action`, `payload`, `at`, `id`, `interval`, `once` |
| `queue` | `list`, `clear`, `play`, `remove` | `start`, `limit`, `pos` |
| `discover` | `scan` | `all` |
| `config` | `path`, `get`, `set`, `unset` | `key`, `value` |
| `watch` | `event` | 无固定 request |
| `doctor` | `report` | 无固定 request |
| `transport` | `play`, `pause`, `stop`, `next`, `prev` | 无固定 request |
| `transport.status` | `get` | 无固定 request |
| `transport.mode` | `get`, `set` | `mode` |
| `transport.source` | `play-uri`, `linein`, `tv`, `music` | `uri`, `title`, `radio`, `from` |
| `transport.volume` | `get`, `set` | `volume` |
| `transport.mute` | `get`, `set`, `toggle` | `mute` |
| `group` | `status`, `join`, `unjoin`, `solo`, `party`, `dissolve` | `all`, `to` |
| `group.volume` | `get`, `set` | `volume` |
| `group.mute` | `get`, `set`, `toggle` | `mute` |
| `favorites` | `list`, `open` | `start`, `limit`, `index`, `title` |
| `music.netease` | `play`, `lucky` | `query`, `category`, `limit`, `index` |
| `music.smapi` | `search`, `browse` | `service`, `query`, `category`, `id`, `limit`, `index` |
| `music.spotify` | `open`, `enqueue`, `play`, `search`, `search_open`, `search_enqueue` | `ref`, `title`, `asNext`, `playNow`, `service`, `query`, `category`, `type`, `market`, `index`, `selectionAction` |
| `auth.smapi` | `begin`, `complete` | `service`, `code`, `linkDeviceID`, `wait` |

---

## 示例

### 1. `queue play 3`

```json
{
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
      "pos": 3
    }
  }
}
```

### 2. `schedule add --action say --payload "早晨好" --at 2026-03-20T09:30:00+08:00 --name morning`

```json
{
  "action": "schedule.add",
  "execution": {
    "version": "v1",
    "capability": "schedule",
    "operation": "add",
    "status": "completed",
    "target": {
      "room": "客厅"
    },
    "request": {
      "name": "morning",
      "room": "客厅",
      "action": "say",
      "payload": "早晨好",
      "at": "2026-03-20T09:30:00+08:00"
    }
  }
}
```

### 3. `ncm play 郑伊健 心照`

```json
{
  "action": "ncm.play",
  "execution": {
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
    }
  }
}
```

---

## 与旧草案的关系

`docs/json-schema-v1.md` 里提到的：

- `target`
- `source`
- `action`
- `strategy`
- `state`

仍然可以看作长期方向。

但截至 2026-03-19，当前仓库真正稳定实现的是：

1. `execution.target`
2. `execution.request`
3. capability/operation 驱动的扁平字段集

所以 v1 正式规范先以这套现实模型为准。

---

## 给 agent 的接入建议

1. 先根据顶层 `action` 决定命令类型
2. 再根据 `execution.capability` + `execution.operation` 选择解析器
3. `execution.target` 与 `execution.request` 按“字段存在才使用”的方式解析
4. 不要假设未来草案里的 `source/action/strategy` 已经实现

这样可以在当前仓库状态下，最小成本接入，又不会误判协议能力。
