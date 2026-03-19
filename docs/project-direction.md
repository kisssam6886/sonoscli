# Project Direction — Sonos × China Music Control Layer

## 1. 项目一句话定位

`sonoscli-plus` 将逐步发展为一个 **面向 Sonos 的中国大陆音乐服务控制层（control layer）**。

它的目标不是做“自然语言聊天机器人”，而是做一个：

- 可被任何 agent / bot / automation 调用
- 对 Sonos 房间、队列、播放编排稳定可靠
- 可扩展到多个中国大陆 Sonos 支持音乐服务
- 能输出稳定 JSON / CLI 结果

的底层能力仓库。

---

## 2. 为什么先走开源

当前阶段最重要的不是商业化，而是先把以下东西做稳：

1. **能力真的有用**：能播、能插播、能排队、能恢复
2. **能力真的稳定**：metadata 正常、队列正常、播放正常
3. **接口真的清晰**：agent 接起来不费劲
4. **扩展真的可行**：不只网易云，未来能接 QQ 音乐 / 酷我等
5. **社区反馈可获得**：看别人真正需要什么

所以第一阶段明确是：

> **先开源，先做能力层，先建立产品形状。**

---

## 3. 不要把仓库定位成“自然语言仓库”

### 仓库负责
- Sonos 对接
- 音乐服务对接
- 队列编排
- 插播/追加/替换
- 播放控制
- 状态读取
- 自愈/恢复
- 稳定 JSON 输出

### 上层 agent 负责
- 自然语言理解
- 粤语 / 普通话 / 英文等表达转换
- 意图识别
- 参数补全
- 对话交互

换句话说：

> **repo 是执行层，agent 是理解层。**

任何 agent 只要接了这个 repo，就能把自然语言转成标准参数，然后调用能力。

---

## 4. 为什么不能只做网易云

网易云是第一条已经打通、并且已经实测证明有价值的服务路径。

但长期看，只做网易云会太局限：

- 用户可能更常用 QQ 音乐
- 不同地区 / 用户绑定的 Sonos 服务不同
- 如果架构写死在 `ncm_*` 上，未来扩展成本会越来越高

所以正确方向是：

> **以网易云为第一条成功的 service adapter，逐步抽象成多服务支持架构。**

---

## 5. 长期服务范围（建议）

第一优先级：中国大陆 Sonos 已支持、且有用户价值的音乐服务。

### 第一梯队
- 网易云音乐
- QQ 音乐
- 酷我音乐

### 第二梯队（视 Sonos 中国区支持情况再评估）
- 酷狗音乐
- 喜马拉雅 / 音频内容服务
- 其他 Sonos 中国区可接入内容服务

### 原则
新增服务时，尽量不复制整套逻辑，而是：
- 复用 Sonos target / queue / playback / retry 能力
- 新增 service-specific source adapter

---

## 6. 核心能力模型（建议固定）

后续新增能力尽量围绕以下五个维度构建，而不是继续堆死命令。

### A. target
播放目标。

示例：
- `room=浴室`
- `room=客厅`
- `room=主卧`
- `group=current`

### B. source
内容来源。

示例：
- `type=artist name=刘德华`
- `type=chart name=热歌榜`
- `type=chart name=网易云欧美热歌榜`
- `type=playlist id=...`
- `type=album name=忘情水`
- `type=search query=郑秀文`

### C. action
对队列做什么。

示例：
- `play_now`
- `append`
- `insert_front`
- `insert_next`
- `replace_queue`
- `play_after_current`

### D. strategy
如何挑选和编排。

示例：
- `limit=10`
- `mode=top`
- `mode=random`
- `dedupe=track`
- `avoid_live=true`
- `preserve_queue=true`
- `play_immediately=true`

### E. state
与当前队列状态相关的操作。

示例：
- `snapshot`
- `restore`
- `undo`
- `inspect`
- `heal_queue`

---

## 7. 当前已经验证有价值的能力

这些不是纸上谈兵，而是已经被实际修过、验证过、证明有用：

### Sonos × 网易云基础能力
- 歌手搜索
- 播放匹配歌曲
- 自动排 5–10 首
- 队列追加
- 插播到前面
- 榜单浏览
- 热歌榜前 10 / 欧美热歌榜前 10 插播

### 稳定性能力
- 修复 Sonos 队列 metadata 不显示
- 修复灰色骨架屏
- 修复“亮了但唔播” / `TRANSITIONING`
- 支持特殊标题（Live / feat / Medley）

这说明 repo 已经开始形成真正的“能力层”，不是单次脚本。

---

## 8. Phase 路线图

## Phase 1 — 打稳网易云能力层
目标：把网易云相关能力从“能用”变成“标准能力”。

### 优先项
- `artist` 热门歌/随机歌
- `chart` 前 N / 随机 N
- `append / insert-front / insert-next / replace`
- JSON 输出标准化
- metadata / playback reliability 固化
- 清理临时探针，沉淀正式接口

产出：
- 一套稳定 CLI
- 一套 agent-friendly 输出
- README / docs 清晰

---

## Phase 2 — 抽象 service adapter 层
目标：不要再把全部逻辑绑定在 `ncm`。

### 方向
抽出：
- 通用 Sonos queue/playback layer
- 通用 source/action/target/strategy model
- service-specific adapter:
  - `netease`
  - `qqmusic`
  - `kuwo`

产出：
- 服务扩展不需要重写队列层
- 更容易接入新服务

---

## Phase 3 — 扩展到 QQ 音乐 / 酷我
目标：从“网易云控制层”升级为“中国大陆 Sonos 音乐服务控制层”。

### 评估内容
- Sonos 是否暴露对应服务 descriptor
- 是否支持 SMAPI 搜索 / browse
- 认证流程是什么
- 排行榜 / 歌单 / 搜索可用程度
- metadata 是否稳定

产出：
- 多服务统一体验
- agent 不必关心底层差异

---

## Phase 4 — Agent-first interface
目标：让任何 agent 更容易接。

### 可选方向
- 统一 JSON request schema
- 统一 JSON response schema
- 稳定 machine-readable errors
- `--format json` 覆盖更多命令
- 一个更抽象的入口（如 `queue request`）

产出：
- OpenClaw / Claude / Codex / bot / workflow 都更容易接入

---

## 9. 值得预留、但不急着马上做的能力

### 1) snapshot / restore / undo
极有价值。

示例：
- 保存当前队列
- 插播一轮
- 结束后恢复原队列

### 2) interleave
例如每隔 2 首插 1 首某歌手或某榜单。

### 3) smart insert
例如：
- 当前歌播完后再插
- 插播到下一首后开始
- 插播后自动回原流程

### 4) seed + expand
例如：
- 先指定一首歌
- 再扩成 5–10 首相近内容

### 5) queue heal
自动检测并修复：
- metadata 空白
- 只亮不播
- transport stuck

---

## 10. 开源与商业化的关系

### 当前阶段：以开源为主
原因：
- 先验证需求
- 先积累能力
- 先沉淀接口
- 先让其他 agent / 用户能接

### 商业化探索：第二阶段再认真评估
但可以现在先列方向。

#### 可能方向 A：托管 agent / bot
- 用户不想自己折腾
- 直接通过 bot / Web UI 控制 Sonos + 中国音乐服务

#### 可能方向 B：高级编排能力
- 智能插播
- AI DJ
- 情绪 / 场景编排
- 播放恢复
- 多房间混播

#### 可能方向 C：B2B / white-label
- 给其他 agent 平台做 Sonos China music backend
- 给商用空间做多区域音乐控制层

#### 可能方向 D：专业版 / hosted features
- 开源基础控制
- 收费高级编排 / 托管 / 历史 / 分析 / UI

### 当前不建议做的事
- 太早定义收费墙
- 太早把自然语言写死进仓库
- 太早围绕单一 bot 绑定产品

---

## 11. 对 Claude Code / 其他 agent 的讨论提纲

后续可以把下面问题交给 Claude Code 一起深入：

1. 如何把当前 `ncm_*` 逻辑抽象成 service adapter 层？
2. `target/source/action/strategy/state` 这五维模型，最适合落成什么 CLI/API 结构？
3. 哪些能力最值得优先固化成正式命令？
4. QQ 音乐 / 酷我接入时，架构要预留哪些扩展点？
5. 哪些能力应该长期放在开源层，哪些才适合商业化？
6. README / docs / examples 怎么写，最容易吸引 agent 开发者接入？

---

## 12. 当前建议结论

### 结论 1
继续沿用当前 fork，作为主仓库持续迭代。

### 结论 2
短期先把“网易云能力层”做稳，不急着马上重构到完全泛化。

### 结论 3
中期开始抽 service adapter，把项目从“网易云专用增强”推进到“中国大陆 Sonos 音乐服务控制层”。

### 结论 4
自然语言交给任何上层 agent，repo 专注做标准化能力层。

---

## 13. 下一步建议（最务实）

### 近期最优先
1. 把今晚已验证能力整理成正式命令/接口
2. 清理临时探针代码
3. 统一 queue/action/source/target 的接口风格
4. 定第一批 issues / milestones

### 接着做
5. 调研 QQ 音乐 / 酷我在 Sonos 上的可行接法
6. 设计 service adapter 结构
7. 给 Claude Code 一份结构化 briefing，一起细化 roadmap

---

## 一句话总括

> `sonoscli-plus` 的未来方向，是成为一个可被任何 agent 接入的 Sonos × 中国大陆音乐服务控制层；自然语言属于上层，稳定能力属于这个仓库。