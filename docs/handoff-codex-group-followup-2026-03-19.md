# Handoff — Codex 继续补齐 Group 能力线

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次把整条 `group` 能力线接进统一 `execution` envelope：

### `group`
- `group status`
- `group join`
- `group unjoin`
- `group solo`
- `group party`
- `group dissolve`

### `group.volume`
- `group volume get`
- `group volume set`

### `group.mute`
- `group mute get`
- `group mute on`
- `group mute off`
- `group mute toggle`
- `group mute set`（隐藏兼容入口）

---

## 为什么改

前面几轮已经把：

- transport
- transport.mode
- transport.source
- status / volume / mute
- discover / config / watch / auth
- 各类 music 入口

陆续收进 execution envelope。

但如果 `group` 还保留旧输出风格，上层 agent 在做：

- 组网状态判断
- 加入房间
- 退组
- party 模式
- dissolve
- group 级音量与静音

时，仍然要为这整条能力线写单独解析逻辑。

这次补完之后，`group` 终于也进入同一条协议线了。

---

## 结果细节

### `group`

统一到：

- `capability = "group"`
- `operation = "status" / "join" / "unjoin" / "solo" / "party" / "dissolve"`

并保留原有业务字段：

- `groups`
- `joiner`
- `to`
- `member`
- `group`
- `results`
- `skipped`

也就是说，agent 现在既能拿稳定 envelope，也不会丢掉原本很有用的细节字段。

### `group.volume`

统一到：

- `capability = "group.volume"`
- `operation = "get" / "set"`

### `group.mute`

统一到：

- `capability = "group.mute"`
- `operation = "get" / "set" / "toggle"`

其中：

- `on`
- `off`
- `set`

都会归到 `operation = "set"`，实际值放在 request/result 里。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*Group'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `group status` JSON 输出带 `execution`
- `group join` JSON 输出带 `execution`
- `group unjoin` JSON 输出带 `execution`
- `group solo` JSON 输出带 `execution`
- `group party` JSON 输出带 `execution`
- `group dissolve` JSON 输出带 `execution`
- `group volume get` JSON 输出带 `execution`
- `group mute on` JSON 输出带 `execution`

---

## 当前限制

1. 这次改的是协议统一，不是 Sonos 组网策略本身变化
   - 即：agent 更容易调用和判断
   - 不代表 Sonos 底层 group 行为被改变

2. `doctor` 仍未收口
   - 当前工作树里它还是未跟踪文件
   - 为避免误碰其他 agent / 用户未提交内容，这轮仍然先跳过

3. 现在 execution envelope 覆盖已经很广
   - 下一步更值得做的是正式 schema 和恢复能力
   - 而不是继续无限堆零散命令

---

## 下一步建议

建议下一轮继续：

1. 视工作树状态决定是否把 `doctor` 收进 envelope
2. 正式写 `request/response/error schema`
3. 把播放与组网 runbook 逐步转成可执行恢复策略
4. 再推进 `snapshot / restore / undo / auto-heal`

---

## 给后续 agent 的一句话

> `group` 这条大能力线已经进入统一 execution 协议；后续重点不应再回到零散输出，而应转向正式 schema、恢复能力和自动修复。
