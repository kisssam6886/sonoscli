# Queue Request Draft v1

## 目标

把已经验证可行的队列操作（append / insert-front / insert-next / replace）抽象成统一 request model，方便：

- CLI 使用
- agent 调用
- 后续 service adapter 对接

---

## 1. Request Shape

```json
{
  "target": {
    "room": "浴室"
  },
  "source": {
    "service": "netease",
    "type": "artist",
    "name": "刘德华"
  },
  "action": {
    "type": "insert_front",
    "playImmediately": false
  },
  "strategy": {
    "limit": 5,
    "mode": "top",
    "dedupe": "track"
  }
}
```

---

## 2. Action 语义定义

### append
加到队尾，不影响当前播放。

### insert_front
插到当前播放后面的前排位置。

### insert_next
插到下一首位置，通常数量较少，强调“下一首先播这个”。

### replace_queue
清空当前队列，用新内容替换。

### play_now
立即切播，并可视情况重建队列。

---

## 3. Strategy 字段建议

### limit
取多少首。

### mode
- `top`
- `random`
- `mixed`

### dedupe
- `none`
- `track`
- `album`

### preserveQueue
对 `append / insert_*` 默认为 true。

### avoidLive
尽量避开 live / remix 等特殊版本。

---

## 4. 建议返回结果

```json
{
  "ok": true,
  "action": "insert_front",
  "target": {
    "room": "浴室",
    "speakerIP": "10.10.10.34"
  },
  "source": {
    "service": "netease",
    "type": "artist",
    "name": "刘德华"
  },
  "result": {
    "inserted": 5,
    "startedPlaying": false,
    "queueLength": 15,
    "items": [
      {
        "title": "暗里着迷",
        "artist": "刘德华",
        "album": "经典重现"
      }
    ]
  }
}
```

---

## 5. 为什么要先文档化

当前仓库虽然已经实测支持：
- 歌手热门歌插播
- 热歌榜前 10 插播
- 欧美热歌榜前 10 插播
- 队尾追加

但这些能力目前还偏“流程逻辑”，不是统一接口。

先把 queue request 结构定下来，可以避免后面新增功能时：
- 参数命名越来越乱
- append / insert / replace 语义不一致
- agent 集成成本升高

---

## 6. 当前最务实的实现路径

### 第一步
继续保留现有命令，但在内部逐步向统一 queue request 结构靠拢。

### 第二步
当 artist/chart/search 都能走统一 resolve → queue action 链路后，再考虑暴露一个更统一的入口，例如：

```bash
sonos queue request --file request.json
```

或：

```bash
sonos ncm queue \
  --room 浴室 \
  --source artist \
  --name 刘德华 \
  --action insert-front \
  --limit 5
```

---

## 7. 结论

Queue request 抽象不是为了换皮，而是为了把当前已经验证有效的操作：
- append
- insert-front
- insert-next
- replace

沉淀成未来任何 agent 都能稳定调用的标准队列动作层。
