# Handoff — Codex 正式化 Schema v1

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次没有继续堆新命令，而是把前面已经接入的 execution envelope 正式落成三份 schema 文档：

1. `docs/schema-request-v1.md`
2. `docs/schema-response-v1.md`
3. `docs/schema-errors-v1.md`

同时补充了总说明文档：

4. `docs/execution-envelope-v1-2026-03-18.md`

---

## 为什么改

前几轮已经把大量命令接进统一 `execution` envelope，但如果没有正式 schema，后续问题会越来越明显：

1. 不同 agent 仍然会各自猜字段
2. 容易把旧草案当成“已经实现”
3. 错误处理会继续依赖字符串猜测
4. 很难把仓库往可商业化 execution layer 推

这次的目标就是把“已经做出来的现实协议”冻结下来，方便后续 agent、UI、automation 和外部调用方直接对齐。

---

## 这次正式化了什么

### 1. Request schema

没有虚构一个还不存在的统一 `execute` 入口，而是明确：

- 当前 v1 request 语义实际分布在
  - `execution.target`
  - `execution.request`

并把当前真实字段按能力域整理出来，例如：

- `query`
- `category`
- `index`
- `pos`
- `mode`
- `volume`
- `mute`
- `service`
- `to`
- `name`
- `key`
- `value`

这样后续 agent 不会再误以为仓库已经实现了完整 `target/source/action/strategy/state` 嵌套输入层。

### 2. Response schema

把当前真实存在的三类响应分开写清楚：

1. 标准成功响应
   - `ok: true`
   - `action`
   - `execution`
2. JSON line 事件
   - `watch`
   - `schedule serve`
3. `doctor` 特例
   - 顶层 `ok` 表示报告健康度，不是命令执行成功

这点很关键，因为如果不写清楚，后续 agent 很容易把 `doctor.ok` 当成普通 success flag。

### 3. Error schema

这次冻结的是当前仓库已经实现的真实错误码：

- `ERR_COMMAND_FAILED`
- `ERR_INVALID_ARGUMENT`
- `ERR_PARTIAL_FAILURE`
- `ERR_TARGET_REQUIRED`
- `ERR_TARGET_NOT_FOUND`
- `ERR_TARGET_AMBIGUOUS`
- `ERR_QUERY_REQUIRED`
- `ERR_NO_RESULTS`
- `ERR_INDEX_OUT_OF_RANGE`
- `ERR_SERVICE_REQUIRED`
- `ERR_SERVICE_NOT_FOUND`
- `ERR_SERVICE_AMBIGUOUS`
- `ERR_NOT_FOUND`
- `ERR_UNSUPPORTED_REF`
- `ERR_STATE_INCONSISTENT`

不是写未来愿景码表，而是直接对齐 `internal/cli/error_output.go` 当前实现。

---

## 怎么验证

这轮是文档正式化，没有改 Go 代码逻辑。

已做最小必要校验：

1. 对照现有实现与草案来源：
   - `internal/cli/execution_output.go`
   - `internal/cli/error_output.go`
   - `docs/json-schema-v1.md`
   - `docs/execution-envelope-v1-2026-03-18.md`
   - `docs/agent-interface-v1.md`
   - `docs/service-adapter-v1.md`
2. 校对 capability / operation / 错误码命名与仓库当前实现一致
3. 只新增 schema 文档并补总说明链接，不碰其他 agent / 用户的脏工作树文件

建议提交前再跑一遍：

- `git diff --stat`

确认范围只落在这轮新增/更新的文档上。

---

## 当前限制

1. 当前还没有统一 JSON 输入层
   - schema v1 正式化的是“当前已输出的请求摘要”
   - 不是“已经支持统一 execute API”

2. 错误响应还没有进入 `execution` envelope
   - 目前仍然是独立 `ok=false + error` 结构

3. `details` 仍然是 capability-specific 上下文
   - 已可机读
   - 但还没有更细一级的强约束子 schema

---

## 下一步建议

建议后续直接沿着这个顺序继续：

1. 选 1 个最小统一入口做 PoC
   - 例如先从 queue/action 型能力做一个 `execute` 草案
2. 把错误逐步从 message 推断迁到显式 coded error
3. 开始做恢复能力：
   - `snapshot`
   - `restore`
   - `undo`
   - `auto-heal`
4. 再考虑多服务 adapter
   - 网易云先打稳
   - 再评估 QQ 音乐 / 酷我接入

---

## 给后续 agent 的一句话

> execution envelope 现在不只是“有了”，而是已经有 request/response/errors 三份正式 schema 可以对齐；下一步该做统一入口和恢复能力，而不是回去继续堆零散命令。
