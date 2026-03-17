# JSON Schema Draft v1

## 目的

为 `sonoscli-plus` 未来的 agent 接入提供稳定、可预期、可 machine-parse 的输入/输出结构。

当前阶段先以 **草案** 形式固定字段思路，不要求一次性完全实现。

---

## 1. Request Model

建议未来统一围绕以下顶层结构：

```json
{
  "target": {
    "room": "浴室",
    "group": "current"
  },
  "source": {
    "service": "netease",
    "type": "chart",
    "name": "热歌榜",
    "query": null,
    "id": null
  },
  "action": {
    "type": "insert_front",
    "playImmediately": false
  },
  "strategy": {
    "limit": 10,
    "mode": "top",
    "dedupe": "track",
    "avoidLive": false,
    "preserveQueue": true
  }
}
```

### target
- `room`: Sonos 房间名
- `group`: 可选，当前组 / 指定组 / auto

### source
- `service`: `netease` / `qqmusic` / `kuwo` / ...
- `type`: `artist` / `chart` / `playlist` / `album` / `search` / `track`
- `name`: 人类可读名，例如“刘德华”“欧美热歌榜”
- `query`: 用于 search
- `id`: 服务内部 ID

### action
- `type`: `play_now` / `append` / `insert_front` / `insert_next` / `replace_queue`
- `playImmediately`: 是否立即切播

### strategy
- `limit`: 数量
- `mode`: `top` / `random` / `mixed`
- `dedupe`: `none` / `track` / `album`
- `avoidLive`: 是否尽量避开 live
- `preserveQueue`: 是否保留当前队列

---

## 2. Success Response Model

```json
{
  "ok": true,
  "target": {
    "room": "浴室",
    "speakerIP": "10.10.10.34"
  },
  "source": {
    "service": "netease",
    "type": "chart",
    "name": "热歌榜"
  },
  "action": {
    "type": "insert_front",
    "playImmediately": false
  },
  "result": {
    "inserted": 10,
    "queueLength": 25,
    "startedPlaying": false,
    "items": [
      {
        "id": "SONG:...",
        "title": "海屿你",
        "artist": "马也_Crabbit",
        "album": "..."
      }
    ]
  }
}
```

### 约定
- 成功时统一返回 `ok: true`
- `items` 尽量返回最终实际入队条目，而不是原始搜索候选
- `startedPlaying` 用来区分“只是排队”还是“已切播”

---

## 3. Error Response Model

```json
{
  "ok": false,
  "error": {
    "code": "SOURCE_NOT_FOUND",
    "message": "chart not found: 欧美热歌榜",
    "retryable": false,
    "details": {
      "service": "netease",
      "sourceType": "chart",
      "name": "欧美热歌榜"
    }
  }
}
```

### 建议错误码（第一批）
- `ROOM_NOT_FOUND`
- `COORDINATOR_NOT_FOUND`
- `SOURCE_NOT_FOUND`
- `SERVICE_NOT_AVAILABLE`
- `SERVICE_AUTH_REQUIRED`
- `NO_PLAYABLE_TRACKS`
- `QUEUE_WRITE_FAILED`
- `PLAYBACK_START_FAILED`
- `INVALID_ARGUMENT`
- `NOT_IMPLEMENTED`

---

## 4. 当前最值得优先统一的 JSON 输出

先不要求所有命令一步到位，但建议优先统一以下几类：

1. `artist -> append/insert-front/replace`
2. `chart -> append/insert-front/replace`
3. `queue inspect`
4. `status`
5. 错误输出

---

## 5. 实施建议

### 第一阶段
- 保持现有命令不大改
- 先把现有 `--format json` 输出风格逐步收敛
- 新增命令时优先按本草案输出

### 第二阶段
- 增加一个更统一的抽象入口，例如：

```bash
sonos ncm queue \
  --room "浴室" \
  --source chart \
  --name "热歌榜" \
  --action insert-front \
  --limit 10 \
  --format json
```

---

## 结论

这份 schema 草案的目标不是一次性定死实现，而是让后续功能扩展时：
- 字段命名尽量一致
- agent 接入尽量低摩擦
- 错误处理尽量标准化
