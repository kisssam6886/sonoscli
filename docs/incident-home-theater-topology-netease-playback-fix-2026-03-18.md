# Incident Report: Home Theater Topology / 网易云播放异常修复

- 日期：2026-03-18
- 位置：客厅 Sonos（10.10.10.31）
- 服务：网易云音乐（Sonos SMAPI sid=165）

---

## 1. 现象
- 入队成功，但播放经常卡在 `TRANSITIONING` 或 `STOPPED`
- 常见表现：第一首可播，后续自动连播失败
- `RelTime` 停在 `0:00:00`

---

## 2. 影响范围
- 主要影响客厅播放器（home theater 路径）
- 浴室播放器同时间段可正常 `PLAYING`，说明并非全局网络中断

---

## 3. 排查结论（根因）
综合实测，问题属于“播放上下文 + 队列构建策略不稳”导致：

1) 之前为了提速，使用了小样本首播 + 后台并发补队列，命中过“可入队但不可稳定起播”的条目。  
2) 在客厅设备上多次切换 TV/music + 并发入队后，播放上下文容易出现不稳定状态。  
3) `play-uri` 直推网易云链路不稳定，不适合作为常规路径。  

> 结论：应统一使用“SMAPI 入队后播 + 连播验收”的稳定流程。

---

## 4. 成功修复方案（已实测通过）

### 步骤 A：基线复位（客厅）
```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
./sonos discover
./sonos music --name "客厅"
./sonos mute off --name "客厅"
./sonos mode shuffle --name "客厅"
./sonos queue clear --name "客厅"
```

### 步骤 B：精确重建队列（约8首）
- 用 `smapi search "黎明 + 歌名"`
- 优先 `artist=黎明` + 歌名命中
- 使用 `--enqueue` 逐首入队

### 步骤 C：开始播放
```bash
./sonos queue play 1 --name "客厅"
./sonos play --name "客厅"
```

### 步骤 D：强制连播验收（关键）
```bash
./sonos status --name "客厅"
./sonos next --name "客厅"; sleep 4; ./sonos status --name "客厅"
./sonos next --name "客厅"; sleep 4; ./sonos status --name "客厅"
```

验收通过标准：
- S1 / S2 / S3 全部 `PLAYING`
- `RelTime` 持续前进（非 `0:00:00`）

---

## 5. 关键规则（防复发）
1. 固定使用 repo binary：`./sonos`（不要混用其他路径）
2. 网易云默认路径：**SMAPI 入队后播**，避免 `play-uri`
3. 先稳后快：先保证连播稳定，再做速度优化
4. 每次修复后必须执行 S1/S2/S3 连播验收

---

## 6. 关联文档
- `docs/handoff-codex-followup-2026-03-18.md`（交接执行方法）
- `/Users/sam/.openclaw/workspace-dev/ISSUE_LEARNINGS.md`（项目级问题沉淀）

---

## 7. 结论
本次故障已通过“基线复位 + 精确入队 + 连播验收”恢复，客厅连续播放恢复正常。后续按该 SOP 执行可显著降低同类故障复发率。
