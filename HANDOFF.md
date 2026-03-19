# HANDOFF.md

## 阶段
Phase 1 — 仓库 MVP 收口 / 文档与接口草案

## 当前状态
- 仓库方向已明确：`sonoscli-plus` 作为 **Sonos × 中国大陆音乐服务能力层 / control layer**
- 已完成一轮 NCM 播放链路修复与真机验证
- 已开始把能力、接口、路线图沉淀成正式文档

## 本阶段已完成
### 代码 /功能
- 修复网易云 Sonos 队列 metadata 空白 / 骨架屏
- 修复“亮了但不播”问题（`TRANSITIONING` 后补 `Play()`）
- 验证通过：
  - 歌手热门歌插播
  - 热歌榜前 10 插播
  - 欧美热歌榜前 10 插播

### 文档
- `docs/agent-interface-v1.md`
- `docs/project-direction.md`
- `docs/mvp-closure-checklist.md`
- `docs/json-schema-v1.md`
- `docs/agent-quickstart.md`
- `docs/service-adapter-v1.md`
- `docs/queue-request-v1.md`

### 清理
- 已删除 `cmd/tmp_*` 临时探针目录

## 关键结论
- repo 负责稳定能力层，不负责自然语言理解
- 房间 / 来源 / 动作 / 策略都应参数化，不应写死在命令名里
- 当前最重要的不是继续堆功能，而是接口标准化，方便任何 agent 接入
- 网易云是第一条打通的 service adapter，但长期不应写死在 `ncm_*`

## 当前建议优先级
1. 固化已验证能力为正式接口
2. 统一 source 模型（artist / chart / playlist / album / search）
3. 逐步形成统一 queue action 入口
4. 开始把 `ncm_*` 逻辑往 ServiceAdapter 抽象迁移
5. 然后再进入第二阶段：`sonos say` / `sonos schedule`

## 不要重复做的事
- 不要再回头排查 favorites 那条旧错线
- 不要现在就做 App / Web UI / Sonos 硬件代理
- 不要把 repo 做成自然语言仓库
- 不要在未验证市场前做不可逆投入

## 下一步动作
- 设计并落一版正式 queue/source/action CLI 入口草案
- 评估如何从当前 `ncm_*` 过渡到 adapter 结构而不大重写
- 继续保持 README / docs / examples 对 agent 开发者友好

## 相关 commit
- `1aa7670` fix: preserve NCM queue metadata on Sonos
- `3ddee12` fix: make NCM queue items display and start reliably
- `0860293` docs: define agent-friendly Sonos x Netease direction
- `162c5fa` docs: add project direction and commercialization notes
- `a459fb5` docs: add MVP closure checklist
- `14ab2f3` docs: add agent schema and quickstart
- `7a3c368` docs: add queue request and service adapter drafts

## 更新时间
2026-03-17 23:42 GMT+8
