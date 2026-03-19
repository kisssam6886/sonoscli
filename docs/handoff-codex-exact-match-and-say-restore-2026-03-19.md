# Handoff — 精确匹配规则收紧 + `say` 恢复链打通（2026-03-19）

## 1. 这轮修了什么

本轮解决了两个用户明确指出的重要问题：

1. **自动选歌必须对得上歌手和歌名**
2. **`say`/新闻播报后必须切回原音乐，而不是停在 `tts.m4a`**

对应代码改动：

- `internal/cli/playability_store.go`
- `internal/cli/ncm_play.go`
- `internal/cli/smapi.go`
- `internal/cli/say.go`
- `internal/sonos/avtransport.go`

## 2. 规则层：精确匹配优先级已收紧

之前只是做了“非 Live 优先”，但这会带来一个明显副作用：

- 可能把非 Live 的翻唱/翻自版本排到原唱前面

用户已明确否定这种行为，因此本轮把规则改成：

1. **歌手精确匹配优先**
2. **歌名命中优先**
3. **翻唱 / 翻自 / Remix / DJ 降权**
4. **Live 仅在更高优先级相同的情况下再降权**

也就是说：

- 原唱 Live
  仍然应该排在
- 翻唱非 Live
  前面

这是本轮最重要的策略修正。

### 当前排序含义

对查询如：

```text
刘德华 男人哭吧不是罪
```

现在排序会优先：

- 歌手 = `刘德华`
- 歌名命中 `男人哭吧不是罪`

然后才讨论：

- 是否 Live
- 是否翻唱 / 翻自 / Remix

## 3. 可播性层：坏版本会继续自动拉黑

此前已落地的本地 playability store 继续生效：

- 若真实触发 `ERR_TRANSITION_STUCK`
- 对应版本会自动写入本地 blocked 记录
- 后续搜索默认跳过这些已 blocked 版本

本轮真机再次验证：

- `郑伊健 心照`
  - `Friends For Life (新曲+精选)` 这次现场卡住并触发 blocked
- `郑伊健 甘心替代你`
  - `Mstersonic 郑伊健 13+` 这次现场也触发 blocked

说明：

- 版本黑名单机制正在生效；
- 但客厅现场对某些版本的可播性仍然会波动；
- 不能只依赖“之前播过一次”，仍然要以真实 `PLAYING + RelTime 前进` 为准。

## 4. 恢复链：`say` 现在会保存并恢复 transport

之前 `say` 的问题是：

- 只会 `PlayURI`
- 只会恢复音量
- 不会恢复原 source / queue position / rel time / play state

本轮已补最小可用恢复链：

### 4.1 新增 snapshot 内容

在插播前会读取：

- `CurrentURI`
- `CurrentURIMetaData`
- `Track`
- `RelTime`
- `Transport State`

### 4.2 恢复逻辑

#### 若原 source 是 queue/music

恢复顺序：

1. `SetAVTransportURI(CurrentURI)`
2. `Seek(TRACK_NR, Track)`
3. `Seek(REL_TIME, RelTime)`
4. 若原状态是 `PLAYING/TRANSITIONING`
   - `Play()`
   - 再复用现有 `PLAYING` 验收 / recovery 逻辑

#### 若原 source 是直接 source（例如 TV / line-in / stream）

恢复顺序：

1. `SetAVTransportURI(CurrentURI, CurrentURIMetaData)`
2. 若原状态是播放中，再 `Play()`

## 5. 真机结果

### 5.1 客厅：最终回归成功

这次重新按“精确歌手 + 精确歌名”建立基线：

查询：

```text
刘德华 男人哭吧不是罪
```

结果：

- 当前排第 1 的是 `刘德华` 原唱版本
- 虽然它是 `Live`
- 但仍然排在翻唱版本之前

执行：

```bash
./sonos smapi search --service "网易云音乐" --category tracks --open --index 1 --name "客厅" "刘德华 男人哭吧不是罪"
```

结果：

- 客厅进入 `PLAYING`
- 当前：
  - `Track: 6`
  - `Title: 男人哭吧不是罪 Live`
  - `Artist: 刘德华`

随后执行粤语新闻式播报：

```bash
./sonos say --name "客厅" --lang yue "新闻简报：..."
```

播报后查询：

- `State: PLAYING`
- `Track: 6`
- `Title: 男人哭吧不是罪 Live`
- `Artist: 刘德华`
- 时间从约 `0:00:11` 前进到：
  - `0:00:47`
  - 4 秒后再查：
  - `0:00:51`

结论：

**客厅这次已经真实通过：音乐 -> 新闻播报 -> 自动切回原音乐。**

### 5.2 浴室：临时验证已回退

本轮中途为隔离“客厅现场不稳”与“恢复链代码是否生效”，曾临时在浴室做过恢复链验证。

用户已明确指出：

- 不应擅自换房间测试；
- 更不应把翻唱结果当成有效候选。

这个反馈正确。

因此本轮已经把浴室恢复回：

- `STOPPED`
- 空队列

后续默认应继续以用户指定房间为准，除非明确说明并得到同意。

## 6. 当前原则修正

从这一轮开始，agent 默认应遵守：

1. **未经说明，不擅自换房间做验证**
2. **自动测试歌单必须对上歌手与歌名**
3. **翻唱 / 翻自 / Remix 不得充当“命中结果”**
4. **只有在原唱候选都不存在时，才考虑 Live 作为退路**
5. **只有 `PLAYING + RelTime 前进` 才算可播**

## 7. 验证情况

已完成：

- `gofmt`
- `go test ./internal/cli ./internal/sonos`
- `go build -o ./sonos ./cmd/sonos`
- 客厅真机：
  - 精确选歌
  - 播放
  - 插播新闻
  - 自动恢复原音乐

## 8. 当前限制

虽然这次客厅恢复链已经打通，但仍有两个现实限制：

1. **部分网易云版本仍会现场波动**
   - 同一首歌不同版本可能今日能播、稍后又 blocked

2. **精确匹配仍是启发式，不是服务端硬保证**
   - 目前是基于：
     - artist match
     - title match
     - adaptation penalty
     - live penalty
   - 若服务端搜索结果本身极差，仍可能需要 agent 二次确认

## 9. 下一步建议

优先级：

1. 把“精确 artist/title 命中”继续 formalize 成统一 matcher
2. 为 `say` 增加可观测输出：
   - snapshot source
   - restored source
   - restored track / rel time
3. 再做 TV -> say -> 自动回 TV 的真机回归
