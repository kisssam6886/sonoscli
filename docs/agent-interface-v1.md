# Sonos × 网易云能力层设计草案 v1

## 定位

`sonoscli-plus` 的目标不是做“自然语言理解器”，而是做一个 **Sonos × 网易云音乐 的稳定能力层 / control layer**。

任何 agent（OpenClaw、Claude、Codex、Telegram bot、Discord bot、Web agent）只要接入这个 repo，就可以把自然语言翻译成标准参数，再调用这里的能力完成：

- 选房间 / 选组
- 搜歌 / 榜单 / 歌手热门歌 / 歌单 / 专辑
- 插播 / 追加 / 替换队列
- 播放 / 暂停 / 跳歌
- 队列快照 / 恢复 / 撤销
- 稳定返回 JSON 结果

**原则**：
- 仓库负责“做得到、做得稳、做得准”
- agent 负责“听得明、讲人话、做意图翻译”

---

## 边界

### 仓库负责
- Sonos 对接
- 网易云 SMAPI 对接
- 队列编排
- metadata 修复
- 播放可靠性
- 标准化输入输出

### 仓库不负责
- 广东话/普通话/英文自然语言理解
- 聊天上下文推理
- Prompt 设计
- 针对某个 bot 的对话话术

---

## 核心模型

未来的能力设计尽量围绕五个维度展开：

### 1. target
播放目标。

示例：
- `room=浴室`
- `room=客厅`
- `room=主卧`
- `group=current`
- `coordinator=auto`

### 2. source
内容来源。

示例：
- `type=artist name=刘德华`
- `type=chart name=热歌榜`
- `type=chart name=网易云欧美热歌榜`
- `type=playlist id=...`
- `type=album name=忘情水`
- `type=search query=郑秀文`
- `type=track name=暗里着迷`

### 3. action
对队列 / 播放做什么动作。

示例：
- `play_now`
- `append`
- `insert_front`
- `insert_next`
- `replace_queue`
- `play_after_current`

### 4. strategy
如何挑选和编排。

示例：
- `limit=10`
- `mode=top`
- `mode=random`
- `dedupe=track`
- `avoid_live=true`
- `preserve_queue=true`
- `play_immediately=true`

### 5. state
与当前队列状态交互。

示例：
- `snapshot`
- `restore`
- `undo`
- `inspect`
- `heal_queue`

---

## 推荐接口形态

不建议只堆大量死命令，例如：
- `play-andy`
- `play-livingroom-top10`
- `insert-bathroom-sammi`

建议做成可组合的接口：

```json
{
  "target": { "room": "浴室" },
  "source": { "type": "chart", "name": "网易云欧美热歌榜" },
  "action": "insert_front",
  "strategy": {
    "limit": 10,
    "mode": "top",
    "dedupe": "track",
    "playImmediately": false
  }
}
```

或 CLI 等价形式：

```bash
sonos ncm queue \
  --room "浴室" \
  --source chart \
  --name "网易云欧美热歌榜" \
  --action insert-front \
  --limit 10
```

---

## 第一批优先能力

### A. Artist flows
- 按歌手取热门歌
- 按歌手随机取 N 首
- 插前 / 插后 / 替换

### B. Chart flows
- 热歌榜前 N
- 新歌榜前 N
- 飙升榜前 N
- 欧美热歌榜前 N
- 随机 N / top N

### C. Queue manipulation
- append
- insert-front
- insert-next
- replace-queue
- queue inspect

### D. Queue resilience
- metadata heal
- play retry after transition
- dedupe

### E. State management
- snapshot
- restore
- undo

---

## 值得提前预留的扩展能力

### interleave
每隔 N 首插 1 首某歌手/某榜单。

### seed + expand
先指定一首歌，再扩成 5~10 首相近内容。

### smart insert
支持：
- 立即插到当前后
- 当前歌播完后开始插
- 插播后恢复原队列顺序

### queue snapshot / restore
先保存当前队列，插播一轮，播完恢复。

### agent-friendly JSON
每个命令尽量提供稳定 JSON 输出，例如：

```json
{
  "ok": true,
  "target": { "room": "浴室" },
  "action": "insert_front",
  "source": { "type": "chart", "name": "热歌榜" },
  "inserted": 10,
  "items": [ ... ]
}
```

---

## 当前已经验证可行的方向

以下能力已经在当前仓库里被实测证明有价值：

- 网易云歌手搜索 + 队列重建
- 插播到前面 / 队尾追加
- 排行榜浏览（TOPBOARD）
- 队列 metadata 修复
- 特殊标题（Medley / Live / feat）稳定显示
- `TRANSITIONING` 时补 `Play()` 提高起播成功率

---

## 结论

`sonoscli-plus` 应继续沿着 **可被任何 agent 接入的 Sonos × 网易云能力层** 方向迭代。

短期优先：
1. 固化 queue/action/source/target 模型
2. 把已验证能力整理成正式接口
3. 保持 JSON 输出稳定
4. 让上层 agent 只需做自然语言翻译，不必理解 Sonos/SMAPI 细节
