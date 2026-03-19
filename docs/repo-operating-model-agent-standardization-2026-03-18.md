# Repo Operating Model — 从家用现场走向 Agent 标准化能力层

时间：2026-03-18  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 1) 一句话结论

商业上不建议你现在同时深做“国内 + 国外所有音乐平台”，  
但技术上必须从现在开始把仓库做成：

> **中国区优先、架构全球可扩展的 Sonos 音乐控制层。**

更具体地说：

- **产品聚焦**：先把中国区的 3 个高价值服务打稳
  - 网易云音乐
  - QQ 音乐
  - 酷我音乐
- **架构要求**：底层设计不能写死在 `ncm`
  - 必须让 Spotify / Apple Music / YouTube Music / Tidal 等后续能接入同一套能力模型

---

## 2) 为什么不建议现在“两边一起深做”

### 原因 A：你当前最强的真实场景在中国区
你现在已经反复验证并修出真实价值的，是：

- Sonos 家庭影院现场
- 网易云音乐 SMAPI
- 中文歌手、榜单、队列、插播、恢复

这部分不是 PPT，而是已经有可复用经验和真实问题库。

### 原因 B：商业化早期更怕“什么都支持，但都不稳”
如果你现在同时深做：

- 网易云
- QQ 音乐
- 酷我
- Spotify
- Apple Music
- YouTube Music

很容易出现：

- 看起来覆盖很广
- 实际上每个平台都只有 60 分
- agent 很难得到稳定结果

对仓库商业化最伤的，不是少几个平台，而是：

> **别人接进来以后发现“有时行、有时不行”。**

### 原因 C：官方支持范围很广，但你的竞争优势不在“列服务名”
按 Sonos 官方服务目录，当前中国站服务页同时列出了：

- `网易云音乐`
- `QQ音乐`
- `酷我音乐`
- `Spotify`
- `苹果音乐`
- `YouTube音乐`

Sonos 官方中国站服务总页当前显示 `131个结果`。  
这说明“理论上可支持很多服务”没问题，真正难的是：

- 哪些服务能稳定搜
- 哪些曲目能稳定播
- 哪些队列策略不容易卡 transport

也就是你现在已经开始摸到的这一层。

来源：
- [Sonos内置服务](https://support.sonos.com/zh-cn/services)
- [Sonos上的网易云音乐](https://support.sonos.com/zh-cn/services/netease-cloud-music)
- [Sonos上的QQ音乐](https://support.sonos.com/zh-cn/services/qq-music)
- [Sonos上的酷我音乐](https://support.sonos.com/zh-cn/services/kuwo-music)

---

## 3) 建议的产品路线

### 阶段 1：把“中国区 Sonos 音乐控制层”做成可卖能力
优先顺序建议：

1. Sonos 核心层稳定
   - 房间发现
   - 队列写入
   - 开播
   - 恢复
   - 状态读取
   - 机器可读错误

2. 网易云做成标准能力
   - 搜索
   - 歌手热歌
   - 榜单
   - 版本选择
   - 黑名单/白名单
   - 先播后补

3. QQ 音乐 / 酷我做 feasibility 与最小打通
   - 不是马上追求“全功能”
   - 而是先验证：
     - 是否能稳定搜索
     - 是否能稳定起播
     - 是否有一致 metadata
     - 是否有可自动化授权流程

### 阶段 2：抽象为多服务 adapter
当网易云足够稳后，再把以下部分彻底拆出来：

- `service adapter`
- `candidate selection`
- `queue playback executor`
- `recovery policy`
- `request/response schema`

这样未来接：

- QQ 音乐
- 酷我
- Spotify
- Apple Music
- YouTube Music

主要是接新 adapter，不是重写整套系统。

### 阶段 3：国际服务作为“同架构扩展”，不是当前主战场
国际服务建议：

- 架构上现在就兼容
- 商业上先不作为主卖点

原因很简单：
- 你今天最真实的用户问题和验证深度都在中国区
- 先把中国区做成标杆，更容易形成差异化

---

## 4) 仓库不应该再像“只在你家里能跑”

以后每次修复都应该沉淀为 4 个层次，而不是只改代码或只留对话记录。

### A. Incident
记录：

- 现象
- 根因
- 修复动作
- 验证结果
- 边界条件

### B. Runbook
记录：

- 别的 agent 如何复现
- 如何判断成功
- 如何兜底

### C. Compatibility / Policy
记录：

- 哪些版本优先
- 哪些版本避开
- 哪些房间拓扑要特别处理
- 哪些服务有特殊限制

### D. GitHub 留痕
至少保留一种：

- docs commit
- PR 说明
- issue / discussion

目标不是“写很多字”，而是让第三方 agent 能接手，不必翻聊天记录。

---

## 5) 建议固定的文档纪律

以后每解决一个现场问题，固定输出下面 3 份产物：

1. `docs/incident-*.md`
2. `docs/runbook-*.md` 或更新现有 runbook
3. `docs/github-note-*.md`

如果问题会影响 agent 行为，再额外补：

4. `docs/policy-*.md` 或 `compatibility-*.md`

### 每份至少要有的字段

#### Incident
- 时间
- 环境
- 现象
- 根因
- 修复
- 验证
- 当前限制

#### Runbook
- 适用前提
- 操作步骤
- 成功标准
- 失败兜底

#### GitHub Note
- 一段摘要
- 为什么重要
- 影响范围
- 对其他 agent 的执行建议

---

## 6) 建议固定的数据模型

如果目标是“让所有 Agent、所有聊天工具、所有自动化平台都能接”，
那仓库应该逐步围绕统一 request / response 模型，而不是继续堆自然语言特例。

推荐继续固定这几个核心维度：

- `target`
  - 例：`room`, `ip`, `group`
- `service`
  - 例：`netease`, `qqmusic`, `kuwo`, `spotify`, `applemusic`
- `source`
  - 例：`artist`, `album`, `playlist`, `chart`, `search`, `track`
- `action`
  - 例：`play_now`, `append`, `replace_queue`, `insert_next`
- `strategy`
  - 例：`limit`, `dedupe`, `avoid_live`, `version_policy`
- `reliability`
  - 例：`retry_play`, `require_playing`, `fallback_candidate`, `blacklist_policy`

这样：
- Telegram bot 能接
- Discord bot 能接
- Web agent 能接
- 本地 CLI 能接
- 任何 LLM agent 也能接

因为上层只负责把自然语言翻译成这套标准参数。

---

## 7) 现在最值得做的工程动作

不是马上扩很多平台，而是先补齐下面 5 件事：

1. **版本选择层**
   - 不再默认拿第 1 条搜索结果
   - 支持专辑优先级 / 黑名单 / 白名单

2. **播放验收层**
   - 只有进入 `PLAYING` 且 `RelTime` 前进才算成功

3. **环境指纹记录**
   - 房间
   - 拓扑
   - 网线位置
   - Sonos 版本
   - 服务
   - binary 路径

4. **统一机器可读错误**
   - 例如 `ERR_TRANSITION_STUCK`
   - 继续往 transport / search / auth / topology 分类

5. **服务适配层抽象**
   - 先把网易云逻辑从 `ncm_*` 继续抽松
   - 为 QQ 音乐 / 酷我预留一致接口

---

## 8) 关于“国内优先还是全球一起做”的最终建议

最终建议很明确：

### 商业定位
先做：

> **Sonos × 中国区音乐服务的可靠控制层**

这是你最有机会做出明显差异化的地方。

### 技术定位
同时坚持：

> **service-agnostic core + adapter-based expansion**

也就是：
- 核心层从现在开始就不能写死在网易云
- 但产品和验证资源，先集中在中国区

### 简单说
- **不要产品上贪大求全**
- **不要架构上把自己锁死**

这两句一起成立，才是最适合你当前阶段的路线。
