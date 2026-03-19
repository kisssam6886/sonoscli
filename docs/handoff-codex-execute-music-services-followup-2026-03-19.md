# Handoff — Codex 扩展 Execute 到音乐服务层

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这一轮把 `sonos execute` 从“只覆盖控制层 + 网易云 + 说话”继续扩到了音乐服务层：

1. 新接入 capability
   - `auth.smapi`
   - `music.smapi`
   - `music.spotify`

2. 新增 / 扩展 action alias
   - `smapi.search`
   - `smapi.browse`
   - `auth.smapi.begin`
   - `auth.smapi.complete`
   - `open`
   - `enqueue`
   - `play.spotify`
   - `search.spotify`

3. 新增 execute builder / helper
   - `buildExecuteAuthSMAPICommand`
   - `buildExecuteSMAPICommand`
   - `buildExecuteSpotifyCommand`
   - `executeDurationField`
   - `executeServiceNameField`

4. 顺手修了一个旧 bug
   - `internal/cli/smapi.go` 里 `smapi.search` / `smapi.browse` 在 `--open` 或 `--enqueue` JSON 输出时，会把额外字段写到 `action`
   - 这会覆盖顶层本应稳定的 `"action": "smapi.search"` / `"action": "smapi.browse"`
   - 现已把冲突字段改成 `playbackAction`

---

## 为什么改

`execute` 如果只会控制 transport，而不能接音乐服务：

1. agent 还是要自己记住很多分散命令
2. 上层很难统一处理授权、搜索、打开、入队这类音乐动作
3. 后面要做 workflow / automation / TTS 联动时，编排入口会断层

所以这一轮的目标不是重写音乐逻辑，而是继续沿着“最小 dispatch 层”走：

1. 输入统一放进 `execute`
2. 底层仍复用现有稳定命令
3. 先把高价值音乐入口补齐

---

## 怎么实现

实现原则保持不变：

1. 不重写音乐命令主体
2. `execute` 只做 request 解析、校验和参数映射
3. 最终仍然调用已有 CLI command

这轮额外做了两个兼容输入约定，方便 agent：

1. `request.service`
   - 支持 `"Spotify"`
   - 也支持 `{ "name": "Spotify" }`

2. `request.wait`
   - `auth.smapi.complete` 支持 duration string
   - 例如 `"30s"`、`"500ms"`

还有两个动态 alias 规则：

1. `play.spotify`
   - 如果 `request.enqueueOnly=true`，自动落到 `music.spotify/enqueue`
   - 否则落到 `music.spotify/play`

2. `search.spotify`
   - `request.selectionAction=open` -> `search_open`
   - `request.selectionAction=enqueue` -> `search_enqueue`
   - 其他情况 -> `search`

---

## 怎么验证

这一轮对应补了 / 收紧的测试：

1. `internal/cli/execute_cmd_test.go`
   - `TestExecuteCmd_SMAPISearchAliasOpen`
   - `TestExecuteCmd_PlaySpotifyAliasInfersEnqueue`
   - `TestExecuteCmd_SearchSpotifyAliasOpen`
   - `TestExecuteCmd_AuthSMAPIBegin`

2. `internal/cli/smapi_cmd_test.go`
   - 收紧 `TestSMAPISearchCmd_OpenPlaysOnSonos`
   - 新增 `TestSMAPIBrowseCmd_OpenKeepsTopLevelAction`

重点不是只看命令成功，而是确认 schema 没回退：

1. 顶层 `action` 仍然是稳定字符串
2. `execution.capability` / `operation` 与 alias 映射一致
3. `--open` 路径会真正触发 Sonos `Play`

建议本轮完成后至少跑：

- `go test ./internal/cli`
- `go build ./cmd/sonos`

---

## 当前限制

这轮虽然把音乐服务入口接上了，但还没到“统一产品 API”阶段：

1. `execute` 仍然是 command dispatch，不是 workflow engine
2. 还没有 snapshot / restore / undo / auto-heal
3. 目前是 capability coverage 优先，输入 schema 仍是渐进冻结
4. 业务层面的可播放性、版权可播性、设备拓扑稳定性，依然是底层服务和 Sonos 状态问题，不会因为 `execute` 存在而自动消失

---

## 给后续 agent 的任务单

建议继续按这个顺序推进：

1. 先把这轮改动完整验证并提交
2. 给 `execute` 再补几类输入校验测试
   - `auth.smapi.complete` 的 `wait`
   - `music.smapi.browse --enqueue/--open`
   - `music.spotify.open/enqueue` 直接 `ref` 路径
3. 然后再考虑更大的统一动作：
   - snapshot
   - restore
   - undo
   - auto-heal

一句话总结：

> `sonos execute` 现在已经不只是控制层入口，而是开始具备“统一调音乐服务”的能力；下一步应该先把 schema 和验证做扎实，再往工作流层长。
