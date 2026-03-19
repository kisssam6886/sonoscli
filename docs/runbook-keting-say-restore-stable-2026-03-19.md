# 客厅 `say` 恢复链稳定版记录（2026-03-19）

## 结论

2026-03-19 这轮客厅真机回归通过，当前可以严格记住以下稳定结论：

1. `say` 恢复链必须按播报前原始 source 自动恢复，不是固定回音乐。
2. 如果播报前是 TV，播报后应回 `x-sonos-htastream:*:spdif`。
3. 如果播报前是网易云音乐单曲播放，播报后应回同一首、同一条播放 URI，并继续推进时间。
4. 当前客厅现场已经验证：
   - `TV -> say -> 自动回 TV` 通过
   - `音乐 -> say -> 自动回原曲` 通过
   - `音乐继续播放 1 分钟 -> say 新闻 -> 自动回原曲` 通过

## 当前稳定逻辑

当前 `say` 恢复链按 `CurrentURI` 分两类：

- 队列源：`x-rincon-queue:*`
  - 先记住 `Track` 和 `RelTime`
  - 恢复时优先走 `PlayQueuePosition(track)`，再 `SeekRelTime(relTime)`
  - 最后执行恢复播放检查
- 直接源：例如 `x-sonos-htastream:*` / 其他直接 URI
  - 直接 `SetAVTransportURI(CurrentURI, CurrentMeta)`
  - 如果播报前是 `PLAYING` 或 `TRANSITIONING`，恢复后继续 `Play`

这个分支判断就是本次恢复稳定的核心。以后不要把“回 TV”和“回音乐”当成两套互不相关逻辑，它们本质上都属于“按原 source 恢复”。

## 本次客厅真机验证

### 1. TV -> say -> 自动回 TV

先切客厅到 TV：

```bash
./sonos tv --name "客厅"
./sonos status --name "客厅" --format json
```

当时确认状态：

- `uri = x-sonos-htastream:RINCON_B8E93741A09801400:spdif`
- `state = PLAYING`

然后执行：

```bash
./sonos say --name "客厅" --lang yue --hold-seconds 10 "测试播报：如果恢复正常，播完之后应该自动返回电视输入。"
```

播报后再次确认：

- `uri` 仍然是 `x-sonos-htastream:RINCON_B8E93741A09801400:spdif`
- `state = PLAYING`

结论：TV 恢复链通过。

### 2. 音乐 -> say -> 自动回原曲

先建立单曲洁净基线，只用严格对得上的原唱非 Live：

```bash
./sonos queue clear --name "客厅"
./sonos smapi search --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 18s "郑伊健 甘心替代你"
./sonos status --name "客厅" --format json
```

当时确认状态：

- `title = 甘心替代你`
- `artist = 郑伊健`
- `state = PLAYING`
- `RelTime = 0:00:01`

然后执行：

```bash
./sonos say --name "客厅" --lang yue --hold-seconds 8 "测试播报：播完请自动返回刚才那首歌。"
./sonos status --name "客厅" --format json
```

播报后确认：

- 仍然是 `郑伊健 - 甘心替代你`
- `state = PLAYING`
- `RelTime` 从 `0:00:01` 推进到 `0:00:23`

结论：音乐恢复链通过。

### 3. 音乐继续播放 1 分钟 -> say 新闻 -> 自动回原曲

本次继续让客厅维持同一首歌播放，不立刻打断，先等待 1 分钟，再播一条短新闻：

基线状态：

- `title = 甘心替代你`
- `artist = 郑伊健`
- `RelTime = 0:02:09`

延时测试命令：

```bash
sleep 60; ./sonos say --name "客厅" --lang yue --hold-seconds 8 "新闻测试：苹果继续推进人工智能功能整合，市场继续关注后续产品节奏。"
```

随后确认状态：

- 仍然是 `郑伊健 - 甘心替代你`
- `state = PLAYING`
- `RelTime = 0:03:12`

结论：延时插播新闻后自动回原曲通过。

## 当前必须遵守的操作规则

1. 一律使用 repo 编译版：

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos ...
```

不要改用 `/opt/homebrew/bin/sonos`。

2. 真机测试只测用户指定房间。

这轮用户明确要求只测客厅，不要再擅自切去浴室。

3. 音乐测试必须严格对得上“歌手 + 歌名”。

- 优先原唱
- 优先非 Live
- 找不到原唱非 Live，才退而求其次
- 不要选翻唱、翻自、Remix、DJ 版本

4. 如果要测 `say` 恢复链，先把播放基线建立稳。

不要在脏队列、错误房间、错误版本歌曲上直接测恢复，否则很容易把“源内容不可播”和“恢复逻辑失败”混在一起。

## 代码侧对应变更

当前恢复稳定依赖以下点：

- `internal/cli/say.go`
  - 队列恢复主路径改为 `PlayQueuePosition(track)` + `SeekRelTime(relTime)`
  - 对直接源保留 `SetAVTransportURI(...) + Play()`
- `internal/cli/say_test.go`
  - 已补队列恢复测试
  - 已补 TV source 恢复测试，防止以后回归

## 后续建议

1. 以后每次做新闻播报回归，优先重复本文件第 3 节这条延时测试。
2. 如果未来再出现“播报后回不去”，先看恢复前的 `CurrentURI` 是队列源还是 TV 源，不要一上来只盯着音乐。
3. 如果未来再出现“歌进得去但播不了”，先当成曲目可播性问题，不要立刻判定 `say` 恢复链又坏了。
