# NCM「写入曲目到 Sonos 队列」复用方法 + 回归测试手册

> 适用仓库：`sonoscli-plus`  
> 目标：复测“搜索后写入队列并稳定起播”的能力（你提到之前已修好的点）

---

## 1) 已修好且可复用的核心方法

对应代码：`internal/cli/ncm_play.go`

### 1.1 命令入口
- `sonos ncm play <query> [--category tracks] [--limit N] [--index K]`
- `sonos ncm lucky <query> [--limit N]`

### 1.2 参数语义
- `query`：搜索词（支持别名归一化，如 `Sammi/Eason/Jay/GEM`）
- `--category`：默认 `tracks`（建议回归固定用 `tracks`）
- `--limit`：搜索返回上限（`play` 默认 10，`lucky` 默认 20）
- `--index`：`play` 时选择第几首可播结果（1-based）

### 1.3 关键流程（写入曲目）
`rebuildNCMQueueNative(...)` 的稳定流程：
1. 搜索并过滤可播曲目（只保留 `SONG:` 前缀）
2. 去重（`dedupeNCMTracks`）
3. **清空队列**：`ClearQueue`
4. 循环写入：`AddURIToQueue(uri, meta, ...)`
5. 定位并播放选中位置：`PlayQueuePosition`
6. 补发 `Play()`
7. 若状态仍是 `TRANSITIONING`，等待约 1.2s 再补一次 `Play()`

> 上述第 7 步就是“亮了但不播”的关键修复点。

### 1.4 metadata 修复点
- 写入时使用 `buildNCMQueueTrackMeta(...)` 生成 DIDL metadata
- 避免 Sonos 队列出现标题/艺人/专辑空白（骨架项）

---

## 2) 直接可执行回归测试（建议顺序）

> 前置：机器与 Sonos 同网段；目标房间名正确（例：`客厅`）

### Step 0：基础连通
```bash
sonos discover --format json
sonos status --name "客厅" --format json
```
**预期**
- discover 能看到设备
- status 返回当前 transport 信息

**失败排查**
- 看不到设备：检查同网段/防火墙/`--timeout` 拉长到 `10s`
- 房间名不对：先用 discover 返回的 Name 原样填 `--name`

---

### Step 1：NCM 搜索能力
```bash
sonos ncm search "Eason" --category tracks --limit 8 --name "客厅" --format json
```
**预期**
- 返回 `items` 非空
- `id` 多数为 `SONG:*`

**失败排查**
- 授权问题：执行 `sonos ncm auth` / `sonos auth smapi begin|complete`
- 服务未就绪：`sonos smapi services --name "客厅"` 检查网易云是否存在

---

### Step 2：写入队列 + 按索引播放（关键）
```bash
sonos ncm play "Eason" --category tracks --limit 8 --index 1 --name "客厅" --format json
```
**预期**
- JSON 含：`playableHits`、`queuedTracks`、`selected`、`uri`
- `queuedTracks >= 1`
- Sonos 实际开始播放

**失败排查**
- `no playable tracks`：搜索词换中文或提高 `--limit`
- `--index out of range`：把 `--index` 设到结果范围内
- 命令成功但不出声：立即执行 `sonos play --name "客厅"`；再看是否 TV 输入源占用（先 `sonos music --name "客厅"`）

---

### Step 3：核对“写入曲目”是否真入队
```bash
sonos queue list --name "客厅" --format json
```
**预期**
- 队列非空
- 至少前几首是本次查询相关曲目
- 标题不应全为空白

**失败排查**
- 队列为空：检查上一步是否报错；重跑 Step 2
- 标题空白：记录返回样本 + 当前 commit，回传（可能 metadata 回归）

---

### Step 4：随机起播路径（lucky）
```bash
sonos ncm lucky "Jay" --limit 12 --name "客厅" --format json
```
**预期**
- 返回 `selected`
- 开始播放且队列被重建

**失败排查**
- 与 Step 2 一样，优先看授权、输入源、播放状态

---

### Step 5：TRANSITIONING 回归专项
```bash
sonos status --name "客厅" --format json
```
在 Step 2/4 后立即连续执行 2~3 次。

**预期**
- 状态应从 `TRANSITIONING` 进入 `PLAYING`
- 若短暂 `TRANSITIONING`，应很快被补 `Play()` 拉起

**失败排查**
- 长时间停在 `TRANSITIONING`：
  1) `sonos play --name "客厅"`
  2) 若仍失败，`sonos music --name "客厅"` 后再 `sonos play`
  3) 抓 `--debug` 日志回报

---

## 3) 推荐一键回归命令集（可直接复制）

```bash
sonos discover --format json
sonos status --name "客厅" --format json
sonos ncm search "Eason" --category tracks --limit 8 --name "客厅" --format json
sonos ncm play "Eason" --category tracks --limit 8 --index 1 --name "客厅" --format json
sonos queue list --name "客厅" --format json
sonos ncm lucky "Jay" --limit 12 --name "客厅" --format json
sonos status --name "客厅" --format json
```

---

## 4) 注意事项（避免误判）
1. **`ncm play/lucky` 会重建队列**（先清空再写入），不是追加。
2. `--index` 是 1-based，不是 0-based。
3. 房间在 TV 输入源时，音乐链路可能被覆盖，先切 `sonos music`。
4. 同名歌手多结果时，建议先 `ncm search` 看结果再 `play --index`。
5. 发生偶发不播时，优先确认 transport 状态是否卡在 `TRANSITIONING`。

---

## 5) 回报模板（你可以直接发 Sam）

- 回归时间：
- 目标房间：
- Step2（写入+播放）结果：成功/失败
- `queuedTracks`：
- 队列 metadata 展示：正常/异常
- `TRANSITIONING -> PLAYING`：正常/异常
- 异常日志摘要（若有）：
- 结论：可回归通过 / 需修复
