# Execute Entry PoC v1

时间：2026-03-19  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 目标

这份文档说明当前仓库新加的最小统一入口：

- `sonos execute`

它的定位不是“最终版统一 API”，而是一个可被 agent / automation 直接调用的 PoC：

1. 用一个小 JSON 请求触发已有能力
2. 先覆盖一批高价值 capability
3. 不大改现有命令实现

---

## 请求结构

当前 `execute` 接受如下 JSON：

```json
{
  "capability": "transport.mode",
  "operation": "set",
  "target": {
    "room": "客厅"
  },
  "request": {
    "mode": "repeat-one"
  }
}
```

### 顶层字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `action` | string | 可选别名；对已支持动作可自动映射到 capability/operation |
| `capability` | string | 能力域 |
| `operation` | string | 动作 |
| `target` | object | 可选目标 |
| `request` | object | 可选业务参数 |

### `target`

当前支持这些字段：

- `room`
- `name`
- `ip`
- `speakerIP`
- `coordinatorIP`

其中：

1. `room` / `name` 会映射到 `--name`
2. `ip` / `speakerIP` / `coordinatorIP` 会映射到 `--ip`
3. request 里的 target 会覆盖命令行上已有的 `--name/--ip`

---

## 调用方式

### 1. 直接传 inline JSON

```bash
sonos execute --format json --data '{
  "capability":"doctor",
  "operation":"report"
}'
```

### 2. 从文件读取

```bash
sonos execute --format json --file ./request.json
```

### 3. 从 stdin 读取

```bash
cat request.json | sonos execute --format json --file -
```

---

## 当前已支持的 capability

### Core / Inspect

- `doctor/report`
- `discover/scan`
- `transport.status/get`

### Transport

- `transport/play`
- `transport/pause`
- `transport/stop`
- `transport/next`
- `transport/prev`

### Mode / Volume / Mute

- `transport.mode/get`
- `transport.mode/set`
- `transport.volume/get`
- `transport.volume/set`
- `transport.mute/get`
- `transport.mute/set`
- `transport.mute/toggle`

### Source

- `transport.source/play-uri`
- `transport.source/linein`
- `transport.source/tv`
- `transport.source/music`

### Queue / Favorites

- `queue/list`
- `queue/clear`
- `queue/play`
- `queue/remove`
- `favorites/list`
- `favorites/open`

### Group

- `group/status`
- `group/join`
- `group/unjoin`
- `group/solo`
- `group/party`
- `group/dissolve`
- `group.volume/get`
- `group.volume/set`
- `group.mute/get`
- `group.mute/set`
- `group.mute/toggle`

### Music

- `music.netease/play`
- `music.netease/lucky`

### Voice

- `say/announce`

---

## 已支持的 `action` 别名

为了让上层 agent 更容易复用现有响应里的 `action` 字段，当前 `execute` 也支持一批 action alias。

例如：

| action | capability / operation |
| --- | --- |
| `doctor` | `doctor/report` |
| `discover` | `discover/scan` |
| `status` | `transport.status/get` |
| `play` | `transport/play` |
| `pause` | `transport/pause` |
| `mode.repeat-one` | `transport.mode/set` |
| `volume.set` | `transport.volume/set` |
| `mute.on` | `transport.mute/set` |
| `play-uri` | `transport.source/play-uri` |
| `queue.play` | `queue/play` |
| `favorites.open` | `favorites/open` |
| `group.party` | `group/party` |
| `group.mute.off` | `group.mute/set` |
| `ncm.play` | `music.netease/play` |
| `say` | `say/announce` |

说明：

1. alias 只覆盖当前 PoC 已支持的动作
2. 如果 alias 自带默认 request，例如 `mode.repeat-one`，会自动补到 `request.mode`
3. 如果 `action` 与显式 `capability/operation` 冲突，会报 `ERR_INVALID_ARGUMENT`

---

## 例子

### 1. 切到 repeat-one

```json
{
  "action": "mode.repeat-one",
  "target": {
    "room": "客厅"
  }
}
```

### 2. 播放队列第 3 首

```json
{
  "capability": "queue",
  "operation": "play",
  "target": {
    "room": "客厅"
  },
  "request": {
    "pos": 3
  }
}
```

### 3. 播放网易云搜索结果

```json
{
  "capability": "music.netease",
  "operation": "play",
  "target": {
    "room": "客厅"
  },
  "request": {
    "query": "郑伊健 心照",
    "category": "tracks",
    "limit": 8,
    "index": 1
  }
}
```

### 4. 执行 TTS 播报

```json
{
  "capability": "say",
  "operation": "announce",
  "target": {
    "room": "客厅"
  },
  "request": {
    "text": "测试广播",
    "lang": "yue",
    "radio": true,
    "holdSeconds": 10
  }
}
```

---

## 返回结构

`sonos execute` 不重新发明一套新响应。

它会把请求映射到现有命令，然后直接返回底层命令原本已经标准化的 JSON 输出。

这意味着：

1. 成功响应继续使用现有的 `action + execution`
2. 错误响应继续使用现有的 `ok=false + error`
3. 上层 agent 不需要为 `execute` 再写一套新响应解析器

---

## 当前限制

1. 这还是 PoC，不是完整统一入口
   - 还没有覆盖全部 capability
   - 也没有正式冻结完整输入 schema

2. 目前只支持一层简单 dispatch
   - 还不是工作流/编排引擎
   - 也不包含恢复动作组合

3. 当前主要价值是“统一入口”
   - 不是“统一业务语义抽象已经完成”
   - 底层仍然是映射到现有命令

4. 还没有覆盖的方向包括：
   - `scene`
   - `schedule`
   - `config`
   - `auth.smapi`
   - `music.smapi`
   - `music.spotify`

---

## 下一步建议

建议按这个顺序继续：

1. 扩展 `execute` 到 `music.smapi` / `music.spotify`
2. 给 `execute` 增加 capability coverage 测试矩阵
3. 再考虑把 `snapshot / restore / undo / auto-heal` 接进统一入口
4. 最后才去讨论更大的 workflow DSL
