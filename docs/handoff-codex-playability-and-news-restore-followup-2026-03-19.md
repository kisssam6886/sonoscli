# Handoff — 可播性规则落地与“新闻后恢复音乐”复测（2026-03-19）

## 1. 这轮改了什么

本轮把之前只存在于文档里的两条规则，开始落到实际代码层：

1. **本地可播性注册表（playability store）**
   - 新增本地文件存储：
     - `~/.config/sonoscli/playability.json`
   - 当某个版本在 `SMAPI open / ncm play` 中真实触发：
     - `ERR_TRANSITION_STUCK`
   - 会自动把该版本记录为：
     - `status=blocked`
     - `reason=transition_stuck`

2. **默认避开已 blocked 的版本**
   - `smapi search`
   - `ncm play`
   - `ncm lucky`
   现在都会默认跳过本地已记录为 `blocked` 的曲目版本。

3. **新增“非 Live 优先”启发式**
   - 只要同一批候选里存在非 Live 版本，就会把明显的：
     - `Live`
     - `演唱会`
     - `音乐会`
     - `concert`
     - `巡回`
     - `现场`
     版本往后排。
   - `ncm lucky` 更进一步：
     - 若存在非 Live 候选，则仅在非 Live 集合里随机。

4. **修正 `smapi search --open/--enqueue` 的纯文本输出**
   - 之前即使已经实际执行入队/播放，纯文本模式仍然打印搜索结果表，容易让人误判“命令没执行”。
   - 现在会直接打印：
     - `已开始播放：...`
     - `已加入队列：...`

## 2. 这轮为什么重要

之前的“版本黑名单 / 白名单”只是操作经验，不是系统行为。

这会导致三个问题：

1. agent 每次都会重复踩已知坏版本；
2. 用户明明说过“不要用 Live、不要用坏版本”，但系统不会记；
3. `smapi search --enqueue` 的输出会误导排障，浪费时间。

本轮之后，项目第一次开始具备“记住现场失败版本并在下次避开”的能力。

## 3. 真机测试结论

### 3.1 版本替代验证

本轮已再次验证：

- `郑伊健 甘心替代你`
  - `古惑仔最强精选集` 版本此前出现过 `TRANSITIONING`
  - `Mstersonic 郑伊健 13+` 版本可播

- `郑伊健 心照`
  - `Discover` 版本此前出现过 `TRANSITIONING`
  - `Friends For Life (新曲+精选)` 版本可播

### 3.2 干净队列连播回归

用以下 3 首重新建了客厅干净队列：

1. `心照` — `Friends For Life (新曲+精选)`
2. `甘心替代你（电影《古惑仔3之只手遮天》插曲）` — `Mstersonic 郑伊健 13+`
3. `男人哭吧不是罪 Live` — `幻影中国巡回演唱会Live`

按以下流程复测：

1. `queue play 1`
2. `play`
3. `next`
4. `next`

结果：

- 第 1 首进入 `PLAYING`
- 第 2 首进入 `PLAYING`
- 第 3 首进入 `PLAYING`
- `RelTime` 均有持续前进

说明：

- 这次已经再次证明，问题并不是“客厅完全不能播”；
- 更像是“坏版本 + transport 恢复不完整”叠加。

### 3.3 “音乐 -> 新闻播报 -> 切回音乐”复测

这是本轮最重要的回归。

#### 播报前现场

客厅在播：

- `甘心替代你`
- `Track=4`
- `State=PLAYING`
- `Time=0:00:39`

#### 执行

执行：

```bash
./sonos say --name "客厅" --lang yue "新闻简报：..."
```

#### 播报后现场

播报结束后立即查询：

- `State=STOPPED`
- `URI=http://10.10.10.129:54200/tts.m4a`
- `Title=tts.m4a`

4 秒后再次查询，仍然是：

- `STOPPED`
- `tts.m4a`

#### 结论

**当前 `say` 仍然不会恢复原音乐 source。**

也就是说：

1. 它会插播成功；
2. 但播完后不会自动切回原队列音乐；
3. 更不会恢复到播报前的那一首、那个进度。

#### 额外恢复测试

随后手动执行：

```bash
./sonos music --name "客厅"
./sonos queue play 4 --name "客厅"
./sonos play --name "客厅"
```

现场表现说明：

- `music` 只能把 source 切回队列模式；
- 不能保证回到播报前那一首；
- 也不能恢复原进度；
- 还可能再次碰到一次 `TRANSITIONING`，需要再补 `play`。

最终客厅被手动拉回：

- `State=PLAYING`
- `Track=2`
- `Title=甘心替代你（电影《古惑仔3之只手遮天》插曲）`

## 4. 当前代码影响范围

本轮代码主要涉及：

- `internal/cli/playability_store.go`
- `internal/cli/ncm_play.go`
- `internal/cli/smapi.go`

测试新增：

- `internal/cli/playability_store_test.go`

已验证：

- `gofmt`
- `go test ./internal/cli`
- `go build -o ./sonos ./cmd/sonos`

## 5. 当前仍未解决的问题

最关键未解决项仍然是：

1. **`say` 没有 snapshot / restore transport source**
   - 不会恢复 TV / 音乐 / line-in

2. **`say` 不会恢复播放上下文**
   - 不会恢复：
     - 原 track
     - 原 queue position
     - 原 rel time
     - 原 play state

3. **即使手动切回 music，也可能再次遇到 `TRANSITIONING`**
   - 说明恢复路径必须复用现有 queue playback recovery 逻辑；
   - 不能只做一个简单的 `music + play`。

## 6. 下一步建议

优先级应为：

1. 给 `say` 增加最小可用 `transport snapshot / restore`
2. 恢复音乐时复用 `queue_playback_executor` 的 recovery
3. 再做真正的“音乐 -> 新闻 -> 自动切回原歌原进度”验收

在这一步完成前，不能宣称“新闻播报后恢复音乐”已经可用。
