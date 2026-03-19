# Incident Record — 客厅网易云队列曲目可播性复测（2026-03-19）

## 1. 背景

用户反馈：

- “还是播不了，什么问题。”
- 当前客厅队列里，部分歌曲已经成功入队，但只有个别歌曲能实际播放。

本次复测目标不是继续加歌，而是把问题拆清楚：

1. 是整条客厅播放链路坏了；
2. 还是队列中存在“能搜到、能入队，但实际拉流失败”的具体曲目。

## 2. 复测环境

- 仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`
- 二进制：强制使用 repo 内 `./sonos`
- 房间：`客厅`
- 服务：网易云（`sid=165`）
- 现场队列：
  1. `甘心替代你`
  2. `心照`
  3. `男人哭吧不是罪 Live`

## 3. 现场证据

### 3.1 队列内容确认

执行：

```bash
./sonos queue list --name "客厅"
```

结果：

```text
POS  TITLE
1    甘心替代你
2    心照
3    男人哭吧不是罪 Live
```

说明：歌曲已经真实进入客厅队列，不是“没加进去”。

### 3.2 第 1 首：`甘心替代你` 明确卡死在 `TRANSITIONING`

执行：

```bash
./sonos queue play 1 --name "客厅"
./sonos status --name "客厅" --timeout 12s
```

状态显示：

- `State: TRANSITIONING`
- `Track: 1`
- `Title: 甘心替代你`
- `Artist: 郑伊健`
- `Time: 0:00:00 / 0:03:17`

结论：

- Sonos 已经切到第 1 首；
- 但 transport 没有进入 `PLAYING`；
- `RelTime` 停在 `0:00:00`，说明并未成功开始拉流播放。

### 3.3 第 2 首：`心照` 同样卡死在 `TRANSITIONING`

执行：

```bash
./sonos queue play 2 --name "客厅" --timeout 12s
./sonos status --name "客厅" --timeout 12s
```

状态显示：

- `State: TRANSITIONING`
- `Track: 2`
- `Title: 心照`
- `Artist: 郑伊健`
- `Time: 0:00:00 / 0:03:17`

结论：

- 第 2 首与第 1 首表现一致；
- 不是单一歌曲偶发问题，而是“当前命中的这类网易云曲目”在客厅存在实际开播失败。

### 3.4 第 3 首：`男人哭吧不是罪 Live` 可恢复到正常播放

执行：

```bash
./sonos queue play 3 --name "客厅" --timeout 12s
./sonos play --name "客厅" --timeout 12s
./sonos status --name "客厅" --timeout 12s
```

最终状态：

- `State: PLAYING`
- `Track: 3`
- `Title: 男人哭吧不是罪 Live`
- `Artist: 刘德华`
- `Time` 持续前进（复测时看到 `0:00:12`、`0:00:15`）

结论：

- 客厅设备本身没有完全失效；
- 同一个队列里，至少第 3 首可以被拉起并稳定播放；
- 补一次 `play` 仍然是必要恢复动作。

## 4. 本次结论

本次已经可以明确排除“只是 CLI 没有把歌加进去”这一条。

更准确的结论是：

1. **入队成功 != 实际可播**
   - 第 1、2 首都已经真实在队列中；
   - 但切过去后长期停在 `TRANSITIONING`，`RelTime=0:00:00`。

2. **`canPlay=true`、能搜到、能入队，都不能当作验收通过**
   - 真正的播放验收仍然必须以：
     - `State=PLAYING`
     - `RelTime` 持续前进
     为准。

3. **当前问题更像“曲目版本/服务侧可播性”叠加“Sonos transport 对失败项恢复不够稳”**
   - 第 1、2 首像是 Sonos 能拿到元数据、能切轨，但实际拉流失败；
   - 第 3 首则可以在补 `play` 后进入稳定播放。

4. **因此，后续不能再用“只要搜得到”的歌做任何自动化恢复测试**
   - 否则会把“恢复逻辑失败”和“歌本身不可播”混在一起。

## 5. 对产品与仓库的直接影响

这次复测进一步确认，仓库下一步应优先做两件事：

1. **建立“稳定可播基线歌单”**
   - 只使用用户现场已验证 `PLAYING + RelTime 前进` 的歌曲做自动化测试；
   - 把“可搜到但不可播”的候选曲目标记为不合格测试样本。

2. **把现有 transport recovery 标准化**
   - 对 `queue play`、插播恢复、任务回播统一复用同一套 `TRANSITIONING -> 补 play -> 再验收` 流程；
   - 不能只在个别命令上做补救。

## 6. 当前安全状态

复测结束后，已将客厅恢复为：

- `Track: 3`
- `Title: 男人哭吧不是罪 Live`
- `State: PLAYING`
- `Volume: 36`

## 7. 下一步建议

按优先级：

1. 先把“稳定可播基线歌单”固化到文档；
2. 再做 `say/news task` 的 `snapshot/restore`；
3. 最后才继续做“大批量加歌 + 插播 + 自动恢复”的整链路测试。
