# Sonos 商业化完整 Handoff（给其他 Agent）

> 版本：v1  
> 更新时间：2026-03-18  
> 仓库：`sonoscli-plus`

## 1) 这项目是什么（不是“只做音乐”）
`sonoscli-plus` 的定位是 **Sonos 控制能力层（control layer）**，音乐只是其中一条已打通的服务链路。  
目标是让任何 Agent（OpenClaw / Telegram Bot / Discord Bot / Web Agent）都能用标准参数和 JSON 输出调用 Sonos 能力，而不需要理解 UPnP/SMAPI 底层细节。

---

## 2) 当前能力全景（商业化可讲的功能面）

### A. 设备与系统控制能力
- 设备发现（discover）
- 房间状态查询（status/now）
- 实时事件订阅（watch）
- 组网控制（group status/join/unjoin/solo/party/dissolve）
- 场景预设（scene save/apply/list/delete）

### B. 播放控制能力
- 基础控制：play/pause/stop/next/prev
- 输入源切换：`tv` / `music`
- 外部 URI 播放：`play-uri`
- Line-in 输入：`linein`
- 队列控制：queue list/play/remove/clear
- 播放模式：`mode get/shuffle/shuffle-norepeat/repeat/repeat-one/normal`

### C. 内容来源与服务接入能力
- Sonos Favorites（list/open）
- Spotify（Sonos linked service / SMAPI 搜索）
- 网易云（`ncm categories/browse/search/play/lucky/auth`）
  - 已修复队列 metadata 显示
  - 已修复 TRANSITIONING 卡住不播
  - `play/lucky` 支持重建队列并播

### D. Agent 可编排能力（最关键）
- `plain/json/tsv` 输出（脚本化友好）
- 参数化调用（room/source/action/policy 不写死）
- 已有接口文档 + schema 草案 + quickstart，方便多 Agent 接入

---

## 3) 已验证价值（可对外讲的“可用证据”）
- 网易云链路已完成真机验证
- 热门歌单/榜单插播场景可稳定执行
- 队列标题/歌手/专辑 metadata 展示已恢复
- 关键播放可靠性问题（亮了但不播）已修复

---

## 4) 商业化叙事建议（给其他 Agent 的统一口径）
不要把它描述成“音乐播放器工具”，而是：

> “一个可复用的家庭音频控制基础设施层（Home Audio Control Layer）。”

可扩展方向：
1. 家庭自动化编排（场景 + 时段 + 设备组）
2. 多源内容统一调度（NCM / Spotify / Radio / URI）
3. Bot/Agent 托管控制（聊天触发、规则触发）
4. B2B 场景（酒店/门店/样板间的音频场景自动化）

---

## 5) 边界（避免团队跑偏）
当前阶段明确：
- ✅ 做：稳定能力层、标准接口、可编排、可观测
- ❌ 不做：自然语言理解仓库、重 UI App、硬件代理重资产方向

原因：先把能力层做厚，后续任何前端/Agent 都能复用。

---

## 6) 下一阶段执行顺序（其他 Agent 接手时按这个做）
1. 固化 queue/source/action 正式 CLI 接口
2. 统一 source 模型（artist/chart/playlist/album/search）
3. 把 `ncm_*` 渐进迁移到 ServiceAdapter 抽象
4. 补接口稳定性与错误码语义
5. 再评估 `sonos say` / `sonos schedule`

---

## 7) 建议给其他 Agent 的阅读顺序
1. 本文：`docs/handoff-agent-commercial-v1.md`
2. 方向文档：`docs/project-direction.md`
3. 接口文档：`docs/agent-interface-v1.md`
4. 适配器文档：`docs/service-adapter-v1.md`
5. 请求模型：`docs/queue-request-v1.md` + `docs/json-schema-v1.md`

---

## 8) 一句话交接模板（可直接转发）
`sonoscli-plus` 不是“只做音乐”，而是 Sonos 控制能力层。音乐服务（网易云/Spotify）只是内容适配器。请按“能力层标准化优先”推进：先统一 queue/source/action 接口，再做 adapter 抽象，最后才上调度与语音等上层能力。
