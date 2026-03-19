# Incident Record — SMAPI 网易云 `--open` 失败与 `TRANSITIONING -> STOPPED`

时间：2026-03-18  
仓库：`sonoscli-plus`

## 问题现象
1. `sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks` 能返回 `SONG:*`。
2. 但加 `--open` 报错：`selected result is not a supported Spotify ref`。
3. 手动 `play-uri x-sonos-http:SONG...sid=165&flags=8232&sn=5` 后，状态常见 `TRANSITIONING -> STOPPED`，未稳定起播。

## 根因
### 根因 1（CLI 路由错误）
`smapi search/browse --open|--enqueue` 旧逻辑只支持 Spotify：
- 强制 `ParseSpotifyRef`
- 非 Spotify 直接报错

因此网易云 `SONG:*` 虽然可搜到，但无法走 `--open`。

### 根因 2（播放路径错误）
对网易云 `SONG:*` 直接使用 `play-uri`（SetAVTransportURI）不稳定，容易出现 `TRANSITIONING -> STOPPED`。

更稳定路径应为：
- `AddURIToQueue`（带正确 metadata）
- `PlayQueuePosition`
- `Play()`（必要时对 `TRANSITIONING` 再补一次 `Play()`）

## 修复方案（已落地）
在 `internal/cli/smapi.go` 新增统一分发：`openOrEnqueueSMAPIItem(...)`

- Spotify ref：保持原有 `EnqueueSpotify` 路径
- 网易云（service id = `165` 且 `SONG:*`）：
  - 使用 `buildNCMTrackURI(...)`
  - 使用 `buildNCMQueueTrackMeta(...)`
  - `AddURIToQueue(...)`
  - 若 `--open`：`PlayQueuePosition(...)` + `Play()` + `TRANSITIONING` 补 `Play()`

## 当前可执行命令（修复后）
```bash
# 1) 搜索 + 直接播放（修复重点）
sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --format json

# 2) 仅入队不立即播放
sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --enqueue --index 1 --name "客厅" --format json

# 3) 验证状态
sonos status --name "客厅" --format json

# 4) 验证队列
sonos queue list --name "客厅" --format json
```

## 注意事项
1. `--open/--enqueue` 需要 `--name` 或 `--ip`。
2. 若目标处于 TV 输入源，先切回音乐队列：
   ```bash
   sonos music --name "客厅"
   ```
3. 若仍偶发不播，可手动补一枪：
   ```bash
   sonos play --name "客厅"
   ```
4. `SONG:*` 是服务内 ID，不等于 Spotify ref，不应再走 Spotify 专用逻辑。

## 提交信息
- 代码修复 commit：见本次变更（`smapi.go`）
