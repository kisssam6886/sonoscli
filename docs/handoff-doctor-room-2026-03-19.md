# Handoff：`doctor room` 已落地（2026-03-19）

## 这次加了什么

新增命令：

```bash
./sonos doctor room --name "客厅"
./sonos doctor room --name "客厅" --format json
```

这个命令不是改播放，而是只读诊断，给其他 Agent 一眼看清楚当前房间是否适合做 `say` 恢复测试。

## 为什么要加

之前一个常见问题是：

1. Agent 看到房间“好像在播歌”
2. 就直接开始测 `say -> 自动切回`
3. 结果其实当前房间是 `STOPPED`、`TRANSITIONING`、脏队列、或者错误 source
4. 最后把“基线不稳”误判成“恢复逻辑又坏了”

`doctor room` 的目的就是先把这层前置判断标准化。

## 这个命令会告诉你什么

它会输出：

- 目标房间是谁
- 当前真正的 coordinator 是谁
- 当前 group 里有多少成员，多少是可见房间，多少是 bonded/invisible
- 当前 source 是什么类型
  - `tv`
  - `queue`
  - `line-in`
  - `radio`
  - `music-track`
  - `unknown`
- `say` 恢复会走哪条路径
  - `queue`
  - `direct`
  - `none`
- 当前 transport state
- 当前曲目、时间、音量
- 当前是否适合做 `say` 恢复测试
- 当前有哪些 warning

## 和 `say` 恢复链的关系

这里最关键的字段有两个：

1. `source.kind`
2. `source.restorePath`

恢复逻辑的统一原则是：

- `restorePath = queue`
  - 表示 `say` 播报后应按队列路径恢复
- `restorePath = direct`
  - 表示 `say` 播报后应按直接 source 恢复
- `restorePath = none`
  - 表示当前根本没有有效 source，不适合测恢复

也就是说，这个命令本质上是在告诉下一位 Agent：

“你现在测的到底是 `TV -> say -> 回 TV`，还是 `音乐 -> say -> 回原曲`，又或者现在根本不该测。”

## 当前客厅真机样本

今天真机跑出的一个真实例子：

```bash
./sonos doctor room --name "客厅" --format json
```

当时返回的关键信息是：

- `source.kind = queue`
- `source.restorePath = queue`
- `state = TRANSITIONING`
- `canTestSayRestore = false`
- warning: `transport is not PLAYING; start TV or music before testing say restore`

这个输出非常有价值，因为它直接说明：

当前客厅虽然表面上有队列和曲目，但**现在并不适合立刻测 `say` 回切**。

## 下一位 Agent 应该怎么用

### 1. 每次真机测 `say` 前先跑

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos doctor room --name "客厅" --format json
```

### 2. 看这几个字段

- `source.restorePath`
- `status.state`
- `source.canTestSayRestore`
- `warnings`

### 3. 通过标准

如果要开始测 `say` 恢复，最好满足：

- `source.restorePath` 不是 `none`
- `status.state = PLAYING`
- `source.canTestSayRestore = true`
- 没有阻止性 warning

### 4. 如果不满足

先恢复播放基线，再测 `say`，不要直接硬测。

## 这次代码位置

- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor_room.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor_room_test.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/execute.go`

## 给下一位 Agent 的一句话

以后别再凭感觉判断“现在适不适合测恢复链”，先跑 `doctor room`。它就是现场版的基线门禁。
