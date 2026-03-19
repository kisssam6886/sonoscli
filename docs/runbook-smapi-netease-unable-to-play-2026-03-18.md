# Runbook：SMAPI 网易云“无法播放 / TRANSITIONING”完整可落地方案

更新时间：2026-03-18  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`  
对应修复提交：`ff49da0`

---

## 1) 根因闭环（为什么你环境可、主会话不可）

### 根因 A：调用了旧二进制（最常见）
- 系统默认 `sonos` 通常来自：`/opt/homebrew/bin/sonos`
- 新修复在仓库代码（commit `ff49da0`）里；如果主会话继续跑旧 binary，就会继续触发旧逻辑：
  - `selected result is not a supported Spotify ref`

### 根因 B：播放路径不对
- 网易云 `SONG:*` 直接 `play-uri` 往往不稳定，容易 `TRANSITIONING -> STOPPED`
- 正确路径应是“入队后播”：
  1. `AddURIToQueue`
  2. `PlayQueuePosition`
  3. `Play()`
  4. 若仍 `TRANSITIONING`，补一次 `Play()`

### 根因 C：输入源/静音/音量干扰
- 当前在 TV 输入源、静音、音量过低，会被误判成“无法播放”

---

## 2) 最终可执行步骤（从零到可播）

### Step 0：固定使用修复版 CLI（关键）
```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos --version
```

### Step 1：前置检查（设备 + 服务）
```bash
./sonos discover --format json
./sonos smapi services --name "客厅" --format json
```
确认服务列表里有“网易云音乐”（service id 165）。

### Step 2：把输入源切回音乐并解除静音
```bash
./sonos music --name "客厅"
./sonos mute off --name "客厅"
./sonos volume set --name "客厅" 30
```

### Step 3：直接搜索并播放（修复路径）
```bash
./sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 10s --format json
```

### Step 4：验证状态与队列
```bash
./sonos status --name "客厅" --format json
./sonos queue list --name "客厅" --format json
```

---

## 3) 需要使用的二进制/路径

**建议必须使用仓库内修复版：**
- 目录：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`
- 命令：`./sonos ...`（或 `go run ./cmd/sonos ...`）

**不建议直接用** `/opt/homebrew/bin/sonos` 进行此次问题复测，除非确认它已包含 `ff49da0` 修复。

---

## 4) 若仍卡 `TRANSITIONING` 的兜底流程

按顺序执行：
```bash
./sonos play --name "客厅"
./sonos music --name "客厅"
./sonos play --name "客厅"
./sonos status --name "客厅" --format json
```

若仍失败，再做一次最小重试：
```bash
./sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 2 --name "客厅" --timeout 10s --format json
./sonos status --name "客厅" --format json
```

---

## 5) 一键命令（可直接复制）

> 一条命令从“编译修复版 -> 切音乐输入 -> 播放 -> 校验”全走完。

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus && \
go build -o ./sonos ./cmd/sonos && \
./sonos music --name "客厅" && \
./sonos mute off --name "客厅" && \
./sonos volume set --name "客厅" 30 && \
./sonos smapi search "斗破苍穹" --service "网易云音乐" --category tracks --open --index 1 --name "客厅" --timeout 10s --format json && \
./sonos play --name "客厅" && \
./sonos status --name "客厅" --format json && \
./sonos queue list --name "客厅" --format json
```

---

## 6) 验证标准（什么算成功）

判定“成功”需同时满足：
1. `smapi search ... --open` 不再报 Spotify ref 错误
2. `status` 里 transport state 最终为 `PLAYING`（而非长期 `TRANSITIONING/STOPPED`）
3. `queue list` 可看到已写入曲目（至少 1 条）
4. 听感上实际有声音输出（非静音、音量>0）

若 1 成功但 2 失败，优先按“兜底流程”执行后再判定。
