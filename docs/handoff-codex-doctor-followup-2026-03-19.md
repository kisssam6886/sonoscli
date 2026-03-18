# Handoff — Codex 继续补齐 Doctor

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 改了什么

这次把 `doctor` 也接进统一 `execution` envelope。

### `doctor`
- `doctor`

统一到：

- `capability = "doctor"`
- `operation = "report"`

---

## 为什么改

前面几轮已经把：

- 播放控制
- 组网控制
- 状态 / 音量 / 静音
- 音乐来源入口
- discover / config / watch / auth

都陆续接进 execution envelope。

如果 `doctor` 还停留在旧 JSON 风格，那么上层 agent 在做：

- 二进制自检
- 路径校验
- repo binary 校验
- 能力自检

时，仍然要走另一套特殊解析逻辑。

这次补完后，核心命令入口基本都已经进入统一协议了。

---

## 兼容处理

`doctor` 有一个特别点：

它原来的 JSON 里，顶层 `ok` 表示的是“doctor 报告是否健康”，
不是“命令调用有没有执行成功”。

这次为了兼容原有语义，没有把这个字段强行改成别的含义，而是：

1. 保留原来的：
   - `ok`
   - `version`
   - `build`
   - `binary`
   - `capabilities`
   - `warnings`

2. 额外补上：
   - `action = "doctor"`
   - `execution.capability = "doctor"`
   - `execution.operation = "report"`

这样旧语义不被打坏，新 agent 也能标准化解析。

---

## 怎么验证

已完成本地验证：

1. 定向测试：
   - `go test ./internal/cli -run 'Test.*Doctor'`

2. 全量 CLI 包测试：
   - `go test ./internal/cli`

3. 构建：
   - `go build ./cmd/sonos`

本次新增/调整的测试重点：

- `doctor` JSON 输出带 `action`
- `doctor` JSON 输出带 `execution`
- `doctor` 仍保留原有 `ok/warnings/...` 语义

---

## 当前限制

1. 现在 execution envelope 覆盖已经很广
   - 再继续补零散入口的收益开始下降
   - 更该做的是正式 schema 和恢复能力

2. 仍有少量 meta 级内容未统一
   - 例如纯帮助/版本类输出
   - 但它们已经不是执行层主价值所在

---

## 下一步建议

建议后续把重点转到：

1. 正式写 `request/response/error schema`
2. 把 runbook 经验继续落成可执行恢复策略
3. 再推进 `snapshot / restore / undo / auto-heal`

---

## 给后续 agent 的一句话

> `doctor` 已经进入 execution envelope；从这一刻开始，更值得投入的不是继续补边角命令，而是把 schema、恢复能力和自动修复做扎实。
