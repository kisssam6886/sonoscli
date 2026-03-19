# Handoff — Codex 接手后 Playback/Execution 输出补完

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次在不碰当前播放实验脏工作树的前提下，把播放控制主链进一步收进统一 `execution` envelope：

1. `transport`
   - `play`
   - `pause`
   - `stop`
   - `next`
   - `prev`

2. `transport.mode`
   - `mode get`
   - `mode shuffle`
   - `mode shuffle-norepeat`
   - `mode repeat`
   - `mode repeat-one`
   - `mode normal`

3. `music.spotify`
   - `open`
   - `enqueue`

4. `favorites`
   - `favorites list`
   - `favorites open`

换句话说，JSON 模式下，这些命令现在都能给上层 agent 返回稳定的：

- `action`
- `execution.version`
- `execution.capability`
- `execution.operation`
- `execution.status`
- `execution.target`
- `execution.request`
- `execution.result`

---

## 为什么改

前一轮已经把：

- `say`
- `scene`
- `schedule`
- `queue`
- `ncm`
- `smapi search/browse`

接进了统一执行输出。

但如果 `mode` 和最常用的 transport/open/favorites 仍然是旧 JSON，上层 agent 还是要混着写多套解析逻辑，无法真正把 repo 当成稳定 execution layer。

这次补完后，日常最常走的“播控入口”已经更接近统一模型：

- 目标选择
- 执行动作
- 请求摘要
- 结果摘要

这样后续无论是 Codex、Claude、OpenClaw 还是脚本/自动化，都更容易标准化对接。

---

## 这次确认的播放模式

仓库里已经支持以下 6 个模式：

- `mode get`
- `mode shuffle`
- `mode shuffle-norepeat`
- `mode repeat`
- `mode repeat-one`
- `mode normal`

另外，`schedule` 的 `mode` payload 也已经支持：

- `shuffle`
- `shuffle-norepeat`
- `repeat`
- `repeat-one`
- `normal`

---

## 怎么验证

已完成的本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*Mode|Test.*Transport|Test.*Open|Test.*Favorites'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `mode` JSON 输出带 `execution`
- `transport` JSON 输出带 `execution`
- `open/enqueue` JSON 输出带 `execution`
- `favorites list/open` JSON 输出带 `execution`

---

## 当前限制

1. 这次改的是“输出协议统一”，不是“播放链路稳定性修复”
   - 即：对 agent 更好接了
   - 但不等于所有音乐服务/所有版本歌曲都一定可播

2. `open/enqueue` 目前仍是 Spotify 路径
   - 不是通用多服务入口

3. 还有几条能力线未接入 envelope：
   - `play spotify`
   - `play-uri`
   - `line-in`
   - `tv`
   - `music`

4. 还没有正式 request schema 文档
   - 当前是命令级 envelope 已统一一大半
   - 但还未进入统一 `execute` 入口

---

## 下一步建议

建议继续按下面顺序收口：

1. 把 `play spotify / play-uri / line-in / tv / music` 也接进 `execution`
2. 正式写 `request/response/error schema`
3. 把播放稳定性问题从“经验 runbook”提升到“可执行恢复策略”
4. 再推进 `snapshot / restore / undo / auto-heal`

---

## 给后续 agent 的一句话

> 当前仓库已经不是单纯的“会播歌 CLI”，而是在逐步收口成 Sonos execution layer；后续改动优先补统一协议、恢复能力和可标准化接入，不要再回到零散命令堆叠。
