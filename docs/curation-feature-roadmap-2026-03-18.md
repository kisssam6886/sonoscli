# Curation Roadmap — 把“播音乐”扩成 Sonos Agent 执行层，但不做成杂货铺

时间：2026-03-18  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`  
参考输入：
- `docs/sonos-business-plan-feature-handoff-2026-03-18.md`
- `docs/handoff-agent-commercial-v1.md`
- `docs/repo-operating-model-agent-standardization-2026-03-18.md`

## 1) 先讲结论

这份 handoff 里面的大方向是对的：

- 不应把仓库只理解成“点歌工具”
- 音乐只是第一条已验证能力线
- TTS / 定时 / scene / 无手操作场景，都应该进入主线视野

但也必须做一层取舍，否则仓库很快会变成：

- 既做播放器
- 又做聊天机器人
- 又做语音助手
- 又做报表系统

最后什么都沾一点，但执行层不够硬。

所以最终判断是：

> **这个仓库要扩，但要围绕“Agent Execution Layer”扩，不是围绕“功能越多越好”扩。**

---

## 2) 我建议正式收进主线的能力

这些能力值得继续做，而且和仓库定位一致。

### A. 播放可靠性层
这是当前最高优先级，必须继续做深。

包括：
- 版本选择策略
- `ERR_TRANSITION_STUCK` 恢复
- 首播验收
- 队列去重
- safe append
- 黑名单 / 白名单

原因：
- 这是真正的差异化能力
- 也是以后所有 agent 最依赖的底层保证

### B. `scene`
这个应该保留并继续强化。

原因：
- 它不是聊天能力
- 是典型执行层能力
- 对家庭影院、多房间、场景切换很有价值

当前仓库现状：
- 已有 `scene save/apply/list/delete`
- 已有对应实现与测试

建议后续：
- 增加 scene apply 的验证输出
- 增加 scene rollback / dry-run
- 把 scene 纳入统一 request schema

### C. `schedule`
这个值得做，但要明确定义成“执行器调度”，不是“全能自动化平台”。

当前仓库现状：
- 已有本地 `schedule add/list/remove/run/serve`
- 当前是本地 JSON store + 轮询 worker 的 MVP

建议后续：
- 先把 schedule 做稳
- 重点是：
  - 机器可读结果
  - 失败重试策略
  - 幂等性
  - 日志/事件输出

不要一开始就做很重的 UI 或复杂规则引擎。

### D. `say` / TTS 播报
这个应该纳入主线，但要明确边界。

当前仓库现状：
- 已有 `sonos say`
- macOS 下可本地生成 TTS 并短暂托管给 Sonos 播放
- 支持 `zh` / `yue`

为什么值得做：
- 它直接提升无手操作体验
- 适合定时任务、提醒、执行确认、场景播报
- 也是商业化里比较好展示的能力

但要明确：
- **TTS 是执行层能力**
- **STT / 语音识别 / 自然语言理解不是本仓主线**

### E. “不方便操作”场景
这个值得保留，但要翻译成执行层语言，不要写成产品文案集合。

应该落成：
- 夜间低音量策略
- 先播后补策略
- 紧急静音高优先级
- 不打断当前房间策略
- 手忙场景的一句话模板

换句话说：

> “不方便操作”是产品场景描述，真正进仓库时要落为策略和 preset。

---

## 3) 我建议保留在上层 agent，而不是放进本仓主线的能力

### A. STT / 语音识别
例如：
- Telegram 语音转文字
- 本地麦克风语音识别
- 第三方语音服务接入

这些不建议放进本仓主线。

原因：
- 平台耦合太强
- 成本和依赖复杂
- 不同入口实现差异很大

建议：
- 本仓只提供标准执行接口
- STT 在外层 agent / gateway 做

### B. NLU / 多轮对话补全
例如：
- “播歌”自动补客厅
- “加十首”自动理解 append
- “讲得唔清楚”给用户 3 个候选

这些逻辑可以有，但不应作为本仓核心代码。

建议：
- 留在 `docs/agent-interface-v1.md` 一类接口文档里
- 作为外层 agent 参考规范
- 不把它写死成 CLI 主逻辑

### C. 聊天平台适配
例如：
- Telegram bot
- Discord bot
- 微信 / 钉钉 / 飞书机器人

这些应是外围项目，不应塞进执行层仓库。

---

## 4) 我建议“先做但不要过度做”的能力

### A. 指标与运营分析
值得做，但先做轻量版。

不要现在就做：
- 大型 dashboard
- 很重的 BI 系统

先做：
- 首播耗时
- 恢复次数
- 成功率
- 常见错误码统计

原因：
- 这些是调优和商业 demo 都能直接用的底层数据

### B. voice-intent bridge
这个概念是对的，但不建议现在做成一个“内置语音理解系统”。

更合适的形态是：
- 一个薄的 mapping spec
- 几个标准例子
- 甚至一个单独的 demo adapter

而不是把语音理解塞进主仓库 CLI。

---

## 5) 基于当前代码现实，应该怎么排优先级

不是从零开始，因为仓库已经有一些基础：

### 已存在，可继续产品化
- `scene`
- `schedule`
- `say`
- `queue playback recovery`
- `machine-readable errors`

### 已明确需要继续补强
- 版本选择策略
- 可播性验收
- safe append
- queue dedupe
- doctor / one-shot diagnosis

### 尚不建议进入主仓
- STT
- NLU
- 对话记忆
- 平台 bot 适配

---

## 6) 建议的路线图（修正版）

### Phase 1：执行层打稳
目标：
- 先把“播得稳”做硬

包含：
- queue reliability
- version policy
- doctor
- queue dedupe
- safe append
- `PLAYING + RelTime` 验收

### Phase 2：执行层扩成功能层
目标：
- 让仓库不止播音乐，但仍然是执行层

包含：
- `scene` 产品化
- `schedule` 产品化
- `say` 产品化
- hands-free presets
- 轻量 metrics

### Phase 3：多服务 adapter
目标：
- 从网易云成功线扩到中国区多服务

优先：
- QQ 音乐
- 酷我音乐

### Phase 4：外围接入生态
目标：
- 让更多 agent / bot / automation 更容易接入

包含：
- 正式 request schema
- 正式 response / error schema
- webhook / local API wrapper（如需要）
- 示例 adapter repo

---

## 7) 以后每个功能用什么标准判断要不要做

建议固定这 5 条：

1. 它是不是执行层能力？
2. 它会不会提升稳定性、恢复性或可编排性？
3. 它能不能被任何 agent 复用？
4. 它会不会把平台耦合或 NLU 逻辑硬塞进仓库？
5. 它能不能留下机器可读结果与可验证行为？

如果：
- `1/2/3/5` 多数是“是”
- `4` 是“否”

就值得做。

---

## 8) 我对这份 handoff 的最终取舍

### 我明确保留的
- 商业化方向：对
- “不只播音乐”的方向：对
- TTS：保留
- 定时任务：保留
- scene / 场景：保留
- 无手操作场景：保留，但转成 preset/policy

### 我会重写表述方式的
- 自然语言能力
- 语音入口
- 多轮补全

这些不删，但会从“仓库内功能”改写为“仓库外接入规范”。

### 我明确不建议现在重投入的
- 仓库内置 STT
- 仓库内置聊天层
- 仓库内置各平台 bot
- 很重的 dashboard/运营后台

---

## 9) 对你当前商业化最有利的做法

你现在最有价值的路线不是：

> “我这个仓库什么都能做”

而是：

> “任何 agent 接进来，都能稳定执行 Sonos 场景任务，而且中国区音乐服务特别强。”

这句话更容易卖，也更容易建立护城河。

所以我建议以后对外统一讲法是：

1. 核心卖点：可靠执行
2. 特色卖点：中国区音乐服务 + 家庭影院现场经验
3. 扩展卖点：scene / schedule / say / recover
4. 接入卖点：任何 agent / bot / automation 都能调

---

## 10) 下一步最值得做的两件事

### A. 把 `say / schedule / scene` 纳入正式 roadmap，而不是散命令
要补：
- 状态模型
- 结果 schema
- 错误码
- 验收标准

### B. 做一个统一“执行请求”入口草案
例如：
- `service`
- `target`
- `source`
- `action`
- `strategy`
- `reliability`
- `feedback`

这样未来：
- 播音乐
- TTS 播报
- scene apply
- 定时任务执行

都能走同一套执行模型，而不是每块各说各话。
