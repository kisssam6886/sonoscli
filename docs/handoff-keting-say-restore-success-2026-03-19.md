# Handoff：客厅 `say` 恢复链已成功（给下一位 Agent）

## 先讲结论

这条功能在 2026-03-19 已经做成，而且做过客厅真机回归：

1. `TV -> say -> 自动回 TV` 成功
2. `音乐 -> say -> 自动回原曲` 成功
3. `音乐继续播放 1 分钟 -> say 播一条新闻 -> 自动回原曲` 成功

这次不是只靠单测判断，而是已经在用户现场的 `客厅` 房间真实跑通。

## 代码和提交

- 工作目录：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`
- 分支：`codex/home-theater-topology-note`
- 关键提交：`7ff873c`
- 提交信息：`fix: stabilize say restore for tv and queue playback`

关键文件：

- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/say.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/say_test.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/docs/runbook-keting-say-restore-stable-2026-03-19.md`

## 这次真正修好的是什么

核心修复不是“让它回音乐”，而是：

`say` 播报前先记录原始播放 source，播报结束后按原 source 自动恢复。

也就是：

- 如果播报前是 TV：回 TV
- 如果播报前是队列/音乐：回原曲、原位置，继续播放

实现上分成两条路径：

1. 队列源：`x-rincon-queue:*`
   - 恢复时优先走 `PlayQueuePosition(track)`
   - 再 `SeekRelTime(relTime)`
   - 最后检查是否成功回到 `PLAYING`
2. 直接源：例如 `x-sonos-htastream:*`
   - 直接 `SetAVTransportURI(CurrentURI, CurrentMeta)`
   - 如播报前在播，就继续 `Play`

## 已经成功的现场样本

### 样本 A：TV 回切

客厅先切到 TV：

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos tv --name "客厅"
./sonos status --name "客厅" --format json
```

已确认成功状态：

- `uri = x-sonos-htastream:RINCON_B8E93741A09801400:spdif`
- `state = PLAYING`

播报：

```bash
./sonos say --name "客厅" --lang yue --hold-seconds 10 "测试播报：如果恢复正常，播完之后应该自动返回电视输入。"
```

播报后再次检查，仍然回到：

- `uri = x-sonos-htastream:RINCON_B8E93741A09801400:spdif`
- `state = PLAYING`

### 样本 B：音乐回原曲

建立干净单曲基线，只选严格对得上的原唱非 Live：

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos queue clear --name "客厅"
./sonos smapi search --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 18s "郑伊健 甘心替代你"
./sonos status --name "客厅" --format json
```

现场成功时的基线：

- `title = 甘心替代你`
- `artist = 郑伊健`
- `state = PLAYING`
- `RelTime = 0:00:01`

播报：

```bash
./sonos say --name "客厅" --lang yue --hold-seconds 8 "测试播报：播完请自动返回刚才那首歌。"
```

播报后检查结果：

- 仍然是 `郑伊健 - 甘心替代你`
- `state = PLAYING`
- `RelTime` 推进到 `0:00:23`

### 样本 C：先让音乐继续播 1 分钟，再插播新闻，再自动回原曲

成功时的播报前基线：

- `title = 甘心替代你`
- `artist = 郑伊健`
- `RelTime = 0:02:09`

延时插播：

```bash
sleep 60; ./sonos say --name "客厅" --lang yue --hold-seconds 8 "新闻测试：苹果继续推进人工智能功能整合，市场继续关注后续产品节奏。"
```

之后检查结果：

- 仍然是 `郑伊健 - 甘心替代你`
- `state = PLAYING`
- `RelTime = 0:03:12`

这条链路已经实测通过。

## 下一位 Agent 复测时必须遵守

1. 只用 repo 里的二进制，不要用 Homebrew 的：

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos ...
```

不要用 `/opt/homebrew/bin/sonos`。

2. 只测用户指定房间。

当前默认只测 `客厅`。不要再切去 `浴室` 做实验，除非用户明确同意。

3. 音乐测试必须严格对得上“歌名 + 歌手”。

- 优先原唱
- 优先非 Live
- 找不到原唱非 Live，才退而求其次
- 不要选翻唱、翻自、Remix、DJ 版本

4. 先建立干净基线，再测恢复链。

不要在脏队列、重复队列、错误歌曲上直接测 `say` 恢复，否则容易把“歌曲本身不可播”和“恢复逻辑坏了”混为一谈。

5. 不要再把“回 TV”和“回音乐”当成两件互不相关的功能。

它们本质上都是“按播报前原始 source 恢复”。

## 建议你让下一位 Agent 这样复测

### 复测 1：音乐 -> say -> 自动回原曲

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos queue clear --name "客厅"
./sonos smapi search --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 18s "郑伊健 甘心替代你"
./sonos status --name "客厅" --format json
./sonos say --name "客厅" --lang yue --hold-seconds 8 "测试播报：播完返回刚才那首歌。"
./sonos status --name "客厅" --format json
```

通过标准：

- 播报后仍然是 `郑伊健 - 甘心替代你`
- `state = PLAYING`
- `RelTime` 比播报前更大

### 复测 2：1 分钟延时新闻 -> 自动回原曲

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos status --name "客厅" --format json
sleep 60; ./sonos say --name "客厅" --lang yue --hold-seconds 8 "新闻测试：苹果继续推进人工智能功能整合，市场继续关注后续产品节奏。"
./sonos status --name "客厅" --format json
```

通过标准：

- 回到播报前同一首歌
- `state = PLAYING`
- `RelTime` 继续推进

### 复测 3：TV -> say -> 自动回 TV

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos tv --name "客厅"
./sonos status --name "客厅" --format json
./sonos say --name "客厅" --lang yue --hold-seconds 8 "电视模式播报测试。"
./sonos status --name "客厅" --format json
```

通过标准：

- `uri` 回到 `x-sonos-htastream:*:spdif`
- `state = PLAYING`

## 如果复测失败，先这样判断

1. 如果播报后 source 变了：
   - 先查 `CurrentURI`，看是 TV 源、队列源，还是别的直接源
2. 如果 source 没变，但歌曲本身播不动：
   - 优先怀疑该曲目版本可播性，不要先怪 `say`
3. 如果歌手、歌名不对：
   - 优先怀疑选曲逻辑或测试命令用错，不要先判定恢复逻辑坏了

## 一句话 handoff

这条功能现在可以视为“已成功、可复测”的状态；下一位 Agent 的任务不是从零排查，而是按本文件的步骤做回归确认，并在失败时严格区分“恢复链失败”与“曲目本身不可播”。
