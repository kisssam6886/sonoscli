# Sonos 项目完整 Handoff（2026-03-17 → 2026-03-18）

> 交接对象：Codex  
> 仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`  
> 分支：`main`  
> 最新文档提交：`4b46330`

---

## 0. 一句话总结（先给结论）

这两天已经把 `sonoscli-plus` 从“单点可用脚本”推进到“可被 Agent 接入的 Sonos 执行层雏形”：
- **功能面**：网易云链路（搜索/入队/播放/模式/输入源切换）打通并增强；
- **稳定性**：修复了网易云 `SONG:*` 在 `--open` 的错误路由，以及队列 metadata/播放可靠性问题；
- **产品面**：已形成“执行层 vs 对话层”边界、商业化方向、接口草案和交接文档体系；
- **当前状态**：代码主问题已修，仍存在特定场景下 `TRANSITIONING` 的现场波动，需要 Codex 接手做系统化兜底与自动恢复完善。

---

## 1. 项目定位与商业化方向（已定调）

### 已定方向
`sonoscli-plus` 不定位为自然语言仓库，而是：
- **Sonos × 中国音乐服务的执行层（control/execution layer）**
- 给 OpenClaw/Codex/Claude 这类上层 Agent 调用
- 上层做理解（NLU），本仓库做可执行动作与稳定输出

### 商业化方向（已文档化）
优先不是“卖点歌”，而是卖“稳定编排与恢复能力”：
1. 高级编排（interleave/smart insert/seed-expand）
2. 状态恢复（snapshot/restore/undo/auto-heal）
3. 托管体验（Hosted bot/Web Console/场景自动化）
4. 多来源统一能力（Netease/QQ/Kuwo adapter）

参考：
- `docs/project-direction.md`
- `docs/handoff-agent-commercial-v1.md`

---

## 2. 2026-03-17 到 2026-03-18 动作全量时间线（按 commit）

> 命令来源：`git log --since '2026-03-17' --until '2026-03-18 23:59'`

### 2026-03-17
- `3f70c3e` feat: add sonos play mode controls
- `d9ef088` docs: rename local fork to sonoscli-plus
- `99deaaa` feat: add netease shortcuts and auth fallback
- `90aa83b` feat: add netease play and lucky commands
- `43e220a` feat: add netease alias normalization
- `de1dea2` feat: add tv/music source switching
- `56c6e90` feat: queue multiple netease tracks for artist playback
- `3403677` fix: decode double-encoded favorite URIs safely
- `1aa7670` fix: preserve NCM queue metadata on Sonos
- `3ddee12` fix: make NCM queue items display and start reliably
- `0860293` docs: define agent-friendly Sonos x Netease direction
- `162c5fa` docs: add project direction and commercialization notes
- `a459fb5` docs: add MVP closure checklist
- `14ab2f3` docs: add agent schema and quickstart
- `7a3c368` docs: add queue request and service adapter drafts
- `bd0ef9c` docs: add phase handoff

### 2026-03-18
- `339db43` docs: add full commercial handoff for multi-agent onboarding
- `3dceeba` docs: include sonos say/schedule and qqmusic-kuwo feasibility track
- `95f22af` feat: scaffold sonos say and schedule MVP commands
- `69220a8` docs: add qqmusic and kuwo feasibility validation templates
- `9fc4e4c` feat: add schedule run executor and document say/schedule MVP
- `e534ba4` docs: add reusable ncm queue-write method and regression playbook
- `71534aa` feat: add schedule serve loop for periodic due-job execution
- `9bd9130` feat: add zh/yue voice-aware sonos say auto-TTS on macOS
- `ff49da0` fix: support SMAPI open/enqueue for Netease SONG refs via queue path
- `4b46330` docs: add end-to-end runbook for smapi netease unable-to-play instability

---

## 3. 关键问题与修复闭环（技术层）

## 3.1 已修复：`smapi search --open` 报 Spotify ref 错

### 现象
- 搜索网易云返回 `SONG:*` 正常
- 但 `--open` 报：`selected result is not a supported Spotify ref`

### 根因（已定位）
旧逻辑强制走 `ParseSpotifyRef`，把网易云 `SONG:*` 当非法输入。

### 修复（`ff49da0`）
在 `internal/cli/smapi.go` 新增 `openOrEnqueueSMAPIItem(...)`：
- Spotify 继续原路径
- 网易云（service id=165 + `SONG:*`）走：
  - `AddURIToQueue`
  - `PlayQueuePosition`
  - `Play()`
  - 若 `TRANSITIONING` 再补一次 `Play()`

### 证据
- 旧 binary `/opt/homebrew/bin/sonos` 可复现该错误
- repo binary `./sonos` 不再报该错误

参考：`docs/incident-smapi-netease-open-fix-2026-03-18.md`

---

## 3.2 现场仍可能出现：`TRANSITIONING` 持续

### 当前观测
- 设备发现正常
- 网易云服务存在（id=165）
- 可搜索 + 可入队
- 但个别现场仍会卡在 `TRANSITIONING`（未到 `PLAYING`）

### 已沉淀 runbook
`docs/runbook-smapi-netease-unable-to-play-2026-03-18.md`
覆盖：
- 根因分层（旧 binary / 播放路径 / 输入源与静音）
- 一键命令
- 兜底流程
- 验收标准

---

## 4. 功能版图（截至现在）

### 4.1 Sonos基础增强
- 播放模式：`mode get/shuffle/shuffle-norepeat/repeat/repeat-one/normal`
- 输入源切换：`tv` / `music`
- 队列稳定性增强与 metadata 修复

### 4.2 网易云能力
- 快捷命令与 auth fallback
- `play` / `lucky`
- 歌手别名归一化
- 艺人多首入队
- `smapi search/browse --open/--enqueue` 兼容 `SONG:*`

### 4.3 Agent 接入文档
- 接口草案、schema、request 模型、quickstart、service adapter 草案

---

## 5. 文档资产清单（可直接交 Codex）

核心建议先读顺序：
1. `docs/project-direction.md`
2. `docs/handoff-agent-commercial-v1.md`
3. `docs/agent-interface-v1.md`
4. `docs/service-adapter-v1.md`
5. `docs/queue-request-v1.md`
6. `docs/json-schema-v1.md`
7. `docs/runbook-smapi-netease-unable-to-play-2026-03-18.md`
8. `docs/incident-smapi-netease-open-fix-2026-03-18.md`
9. `docs/regression-ncm-queue-write-playbook.md`
10. `docs/mvp-closure-checklist.md`

补充资料：
- `docs/feasibility-qqmusic.md`
- `docs/feasibility-kuwo.md`
- `docs/plan-execution-v1.md`
- `docs/handoff-agent-commercial-v1.md`

---

## 6. 当前风险与未完成事项（Codex 接手重点）

### P0（先做）
1. **把 `TRANSITIONING` 波动收敛为确定性恢复**
   - 增加状态机式重试策略（定次、定间隔、定退出条件）
   - 输出机器可读错误码（例如 `ERR_TRANSITION_STUCK`）

2. **统一“播放执行路径”并封装可复用执行器**
   - 所有网易云 track 播放均走 queue-path
   - 避免命令层重复分散实现

3. **二进制版本防误用机制**
   - 明确告警：检测到旧 binary 时输出强提示
   - 可选：`sonos doctor` 增加版本与功能自检

### P1（随后）
4. 把 `target/source/action/strategy/state` 模型落成稳定命令/API
5. 完成 QQ/Kuwo adapter feasibility 的第一轮代码验证
6. 补自动化回归：
   - `smapi open SONG:*`
   - queue metadata
   - transition recovery

---

## 7. 现场可复现/验证命令（给 Codex 快速上手）

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus

# 编译
 go build -o ./sonos ./cmd/sonos

# 设备与服务检查
./sonos discover --format json
./sonos smapi services --name "客厅" --format json

# 核心回归：网易云 SONG:* open
./sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 12s --format json

# 状态与队列验证
./sonos status --name "客厅" --format json
./sonos queue list --name "客厅" --format json
```

对比旧 binary 问题：
```bash
/opt/homebrew/bin/sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 12s --format json
```

---

## 8. 给 Codex 的明确执行边界（避免跑偏）

1. **不要把仓库改成 NLP/聊天逻辑仓库**，保持 execution-layer 定位。  
2. **先解决可靠性再扩功能**，先把播放链路与恢复机制做实。  
3. **所有新增功能必须有可回归命令与成功判据**。  
4. **文档先行**：每个 P0 改动都要同步 runbook/incident。  
5. **兼容现有 CLI 体验**：不要大破坏命令习惯。

---

## 9. 建议直接给 Codex 的任务单（可复制）

> 在 `sonoscli-plus` 仓库完成 P0 稳定性收敛：
> 1) 设计并实现统一 `queue-path playback executor`（含 transition 恢复策略）；
> 2) 给 `smapi open/enqueue`、`ncm play/lucky` 统一接入该 executor；
> 3) 增加机器可读错误码与日志字段；
> 4) 增加最小回归测试/脚本（重点覆盖 `SONG:* open` 与 `TRANSITIONING` 恢复）；
> 5) 更新 docs（runbook + incident + quickstart）并附可复制验证命令。

---

## 10. 交接时点状态（现在）

- 代码主线可用：✅
- 网易云 `SONG:*` `--open` 逻辑 bug 已修：✅
- 商业化方向与架构方向已明确文档化：✅
- 现场“稳定播放”仍有波动，需做系统化恢复：⚠️
- 已有可执行 runbook，能快速定位与操作：✅

---

## 11. 你可直接发给 Sam/Codex 的路径

**主交接文档（本文件）：**
`/Users/sam/.openclaw/workspace-dev/sonoscli-plus/docs/handoff-2026-03-17-to-2026-03-18-for-codex.md`

**必须附带两份问题文档：**
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/docs/incident-smapi-netease-open-fix-2026-03-18.md`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/docs/runbook-smapi-netease-unable-to-play-2026-03-18.md`
