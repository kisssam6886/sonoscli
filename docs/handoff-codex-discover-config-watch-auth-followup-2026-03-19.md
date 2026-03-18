# Handoff — Codex 继续补齐 Discover / Config / Watch / Auth 入口

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次继续把一批低风险、agent 常用的辅助入口接进统一 `execution` envelope：

### `discover`
- `discover`

### `config`
- `config path`
- `config get`
- `config set`
- `config unset`

### `watch`
- `watch`（JSON line event）

### `auth.smapi`
- `auth smapi begin`
- `auth smapi complete`

---

## 为什么改

前几轮已经把：

- 播放控制
- 模式切换
- 来源切换
- Spotify / Netease / SMAPI 音乐入口
- 状态 / 音量 / 静音

统一到了 execution envelope。

但如果：

- discover
- config
- watch
- auth

这些辅助入口还停在旧风格输出，那么上层 agent 在做：

- 自发现
- 初始化配置
- 实时观察
- 服务授权

时，仍然要走另一套解析逻辑。

这次补完后，这些“非播放但 agent 很常需要”的入口也更接近统一协议了。

---

## 结果细节

### `discover`

统一到：

- `capability = "discover"`
- `operation = "scan"`

注意：

**JSON 根节点已从原来的数组，统一为对象**，现在会返回：

- `action`
- `execution`
- `items`
- `count`

这样上层 agent 才能稳定拿到：

- discover 请求摘要
- discover 结果计数
- 设备列表

这是这轮唯一一个带轻微兼容性取舍的入口。

### `config`

统一到：

- `capability = "config"`
- `operation = "path" / "get" / "set" / "unset"`

### `watch`

JSON 模式下，仍然是 **JSON line event stream**，但每条事件现在都会附带：

- `action = "watch.event"`
- `execution.capability = "watch"`
- `execution.operation = "event"`

同时保留原有事件字段：

- `time`
- `service`
- `sid`
- `seq`
- `vars`

### `auth smapi`

统一到：

- `capability = "auth.smapi"`
- `operation = "begin" / "complete"`

这样上层 agent 可以更明确地区分：

- 开始授权
- 完成授权

而不需要只靠命令名判断。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*Discover|Test.*Config|Test.*Watch|TestAuthSMAPI|TestCompleteSMAPIAuth'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `discover` JSON 输出带 `execution`
- `config path/get/set` JSON 输出带 `execution`
- `watch` JSON line event 带 `execution`
- `auth smapi begin/complete` JSON 输出带 `execution`

---

## 当前限制

1. `discover` JSON 根节点已不再是裸数组
   - 对 agent 标准化更友好
   - 但如果旧脚本直接假设 `[]` 根节点，需要同步适配

2. 这轮没有动 `doctor`
   - 因为当前工作树里它还是未跟踪文件
   - 为避免误碰其他 agent / 用户未提交内容，这次先跳过

3. `group` 这一大块还没正式接进 envelope
   - 当前文件本身就在脏工作树里
   - 下一轮应在现有改动基础上继续推进，而不是重写

---

## 下一步建议

建议下一轮继续：

1. 把 `group` 全量接入 `execution`
2. 视当前工作树状态，再决定是否收 `doctor`
3. 正式写 `request/response/error schema`
4. 再把 runbook 经验继续落到恢复策略和自动修复

---

## 给后续 agent 的一句话

> 非播放辅助入口现在也开始进入统一 execution 协议；下一步优先把 group 收完，再看 doctor，保持“小步标准化 + 小步验证 + 小步留痕”的节奏继续推进。
