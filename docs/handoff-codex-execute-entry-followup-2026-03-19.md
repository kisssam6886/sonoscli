# Handoff — Codex 落最小 Execute 入口 PoC

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次往前推的不是新播放技巧，而是给仓库加了一个最小统一入口：

- `sonos execute`

代码入口：

- `internal/cli/execute.go`

同时把命令挂到 root：

- `internal/cli/root.go`

并补了测试：

- `internal/cli/execute_cmd_test.go`

再加一份使用说明：

- `docs/execute-entry-poc-v1-2026-03-19.md`

---

## 为什么改

前面几轮已经把响应协议统一得差不多了：

- `action`
- `execution`
- `ERR_*`

但如果上层 agent 还是要自己拼各种 CLI 参数，那么 repo 仍然更像“命令集合”，不是“统一执行层”。

所以这次先落一个最小 PoC：

1. 输入用一个小 JSON
2. 统一走 `sonos execute`
3. 底层先映射到已有命令
4. 不大改已有稳定能力

这样可以先证明：

> 这个仓库已经可以被任何 agent 当成一个统一入口去调用，而不是必须记几十条 CLI 命令细节。

---

## 这次支持了什么

当前 `execute` 已支持：

1. Core / Inspect
   - `doctor/report`
   - `discover/scan`
   - `transport.status/get`

2. Transport
   - `transport/play`
   - `transport/pause`
   - `transport/stop`
   - `transport/next`
   - `transport/prev`

3. Mode / Volume / Mute
   - `transport.mode/get|set`
   - `transport.volume/get|set`
   - `transport.mute/get|set|toggle`

4. Source
   - `transport.source/play-uri|linein|tv|music`

5. Queue / Favorites
   - `queue/list|clear|play|remove`
   - `favorites/list|open`

6. Group
   - `group/status|join|unjoin|solo|party|dissolve`
   - `group.volume/get|set`
   - `group.mute/get|set|toggle`

7. Music / Voice
   - `music.netease/play|lucky`
   - `say/announce`

另外还支持一批 action alias，方便上层直接复用现有响应里的 `action`。

例如：

- `status`
- `mode.repeat-one`
- `queue.play`
- `group.party`
- `ncm.play`
- `say`

---

## 怎么实现的

实现策略是：

1. `execute` 先读 JSON 请求
   - `--data`
   - `--file`
   - `--file -`

2. 把：
   - `action`
   - `capability`
   - `operation`
   - `target`
   - `request`

标准化

3. 再把请求映射到现有命令
   - 而不是重写一套新业务逻辑

4. 最后直接复用底层命令原有输出

这很重要，因为这样做有两个好处：

1. 现有命令不用大搬家
2. `execute` 直接继承现有的：
   - success response
   - execution envelope
   - error envelope

也就是说，这次新增的是统一入口，不是平行实现一套新系统。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'TestExecute'`

覆盖点包括：

- inline JSON dispatch
- file JSON dispatch
- action alias -> capability/operation 映射
- target.ip 透传到底层命令
- root 下 `--format json` 错误 envelope

建议本轮收尾再跑：

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

---

## 当前限制

1. 这是最小 PoC，不是最终 execute API
   - 还不是完整输入 schema
   - 也没有覆盖所有 capability

2. 现在主要是 command dispatch
   - 还不是 workflow engine
   - 没有 snapshot/restore/undo

3. 暂未覆盖：
   - `scene`
   - `schedule`
   - `config`
   - `auth.smapi`
   - `music.smapi`
   - `music.spotify`

4. 目前仍然是“映射到现有命令”
   - 这对小步推进是好事
   - 但长期仍要走向更真实的 capability handler / adapter

---

## 下一步建议

下一轮最值得继续的是：

1. 把 `music.smapi` / `music.spotify` 接进 `execute`
2. 再补 coverage test，避免 capability 长大后回归
3. 然后才开始做：
   - `snapshot`
   - `restore`
   - `undo`
   - `auto-heal`

---

## 给后续 agent 的一句话

> 仓库现在已经不只是“输出协议统一”，而是有了最小统一输入入口 `sonos execute`；下一步该扩 capability coverage 和恢复能力，而不是回去继续堆散命令。
