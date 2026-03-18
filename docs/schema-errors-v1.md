# Schema Errors v1

时间：2026-03-19  
适用仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 目标

这份文档冻结当前仓库已经实现的 machine-readable error 输出。

重点是：

1. 错误 envelope 长什么样
2. 当前真实存在的 `ERR_*` 错误码有哪些
3. 这些错误码大概意味着什么，通常会附哪些 detail

这份文档以 `internal/cli/error_output.go` 为准。

---

## 1. 错误响应结构

当前 JSON 错误响应使用独立 envelope：

```json
{
  "ok": false,
  "error": {
    "code": "ERR_TARGET_NOT_FOUND",
    "message": "speaker name not found: 客厅",
    "details": {
      "ip": "",
      "name": "客厅",
      "timeout": "5s",
      "resolution": "topology",
      "room": "客厅"
    }
  }
}
```

### 字段定义

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ok` | bool | 错误时固定为 `false` |
| `error.code` | string | 机器可判定错误码 |
| `error.message` | string | 面向人类的错误信息 |
| `error.details` | object | 可选，补充上下文 |

### 当前重要边界

错误 envelope 目前：

1. 没有顶层 `action`
2. 没有 `execution`
3. 只保证 `ok=false` 和 `error` 对象

所以如果你在做统一解析：

- 成功响应走 `action + execution`
- 错误响应走 `ok + error.code + error.details`

这是当前仓库的真实状态。

---

## 2. 错误码产生方式

当前错误码有两种来源。

### A. 显式 coded error

命令逻辑主动调用：

- `newInvalidArgumentError`
- `newTargetNotFoundError`
- `newServiceNotFoundError`
- `newNoResultsError`

这类错误会直接输出预设 `ERR_*` code，并尽量补齐 `details`。

### B. 文本推断

如果某个错误不是 coded error，当前实现会根据错误消息内容做推断，例如：

- 包含 `no speakers found` -> `ERR_TARGET_NOT_FOUND`
- 包含 `service not found` -> `ERR_SERVICE_NOT_FOUND`
- 包含 `no results` -> `ERR_NO_RESULTS`

如果推断不到明确类别，则回落到：

- `ERR_COMMAND_FAILED`

这意味着：

1. 显式 coded error 的稳定性更高
2. 推断类错误仍可用，但最好逐步减少字符串猜测依赖

---

## 3. 当前正式错误码

以下错误码已经在代码中定义并投入使用。

| Code | 含义 | 常见场景 | 常见 details |
| --- | --- | --- | --- |
| `ERR_COMMAND_FAILED` | 通用命令失败 | 其他错误无法归类时回落 | `cause` 等零散字段 |
| `ERR_INVALID_ARGUMENT` | 参数非法 | 输入格式不对、枚举值错误、缺少某必填 flag | `action`, `flag`, `value`, `cause` |
| `ERR_PARTIAL_FAILURE` | 部分成功、部分失败 | `group solo/party/dissolve` 批量操作时部分成员失败 | `attempted`, `succeeded`, `failed`, `skipped`, `results`, `failures` |
| `ERR_TARGET_REQUIRED` | 缺少目标 | 需要 `--ip` / `--name` 才能执行 | `ip`, `name`, `timeout`, `action` |
| `ERR_TARGET_NOT_FOUND` | 找不到目标设备/房间 | 房间名不存在、网络里没有设备、场景应用时成员不在网络里 | `ip`, `name`, `timeout`, `room`, `speaker`, `speakerIP`, `resolution`, `uuid` |
| `ERR_TARGET_AMBIGUOUS` | 目标匹配不唯一 | 房间名模糊匹配到多个设备 | `ip`, `name`, `timeout` |
| `ERR_QUERY_REQUIRED` | 缺少查询词 | 搜索命令没给 query | `source` |
| `ERR_NO_RESULTS` | 没有结果 / 没有可播结果 | 搜索为空、browse 为空、没有 playable track | `action`, `source`, `service`, `query`, `category`, `id` |
| `ERR_INDEX_OUT_OF_RANGE` | 选择位置超范围 | `--index` / favorite 索引 / 可播结果位置超界 | `index`, `total`, `action`, `source`, `query`, `category` |
| `ERR_SERVICE_REQUIRED` | 缺少服务名 | `--service` 必填但未提供 | 无或极少 detail |
| `ERR_SERVICE_NOT_FOUND` | 找不到音乐服务 | 指定的 SMAPI 服务不存在 | `service`, `available` |
| `ERR_SERVICE_AMBIGUOUS` | 服务匹配不唯一 | 服务名模糊匹配到多个候选 | `service`, `matches` |
| `ERR_NOT_FOUND` | 泛化“未找到” | `scene not found`、`favorite not found`、`schedule id not found` | `action`, `scene`, `title`, `id`, `source` |
| `ERR_UNSUPPORTED_REF` | 引用不可直接播放/入队 | 结果不是可播放 Spotify ref 等 | `source`, `query`, `ref` |
| `ERR_STATE_INCONSISTENT` | 运行时状态不一致 | dedupe 后选中项消失、拓扑缺 UUID、目标不在任何 group | `action`, `source`, `selectedID`, `speaker`, `speakerIP`, `groupID` |

---

## 4. `details` 字段约定

`error.details` 当前没有单一 schema，但已经形成一些稳定习惯。

### A. Target 相关

常见键：

- `ip`
- `name`
- `timeout`
- `room`
- `speaker`
- `speakerIP`
- `resolution`
- `uuid`
- `kind`

### B. Query / Source 相关

常见键：

- `source`
- `service`
- `query`
- `category`
- `id`
- `ref`

### C. 索引 / 选择相关

常见键：

- `index`
- `total`

### D. 批量执行相关

常见键：

- `attempted`
- `succeeded`
- `failed`
- `skipped`
- `results`
- `failures`

### E. 底层错误原因

如果错误是 wrapped error，当前实现可能补：

- `cause`

说明：

`details` 是上下文字段，不保证每次都齐全；agent 应按“有则利用，无则降级”的方式解析。

---

## 5. 典型示例

### 1. 目标缺失

```json
{
  "ok": false,
  "error": {
    "code": "ERR_TARGET_REQUIRED",
    "message": "provide --ip or --name (or run `sonos discover`)",
    "details": {
      "ip": "",
      "name": "",
      "timeout": "5s"
    }
  }
}
```

### 2. 没有搜索结果

```json
{
  "ok": false,
  "error": {
    "code": "ERR_NO_RESULTS",
    "message": "no playable tracks",
    "details": {
      "source": "netease.ncm",
      "query": "郑伊健 + 不存在歌曲",
      "category": "tracks"
    }
  }
}
```

### 3. Group 部分失败

```json
{
  "ok": false,
  "error": {
    "code": "ERR_PARTIAL_FAILURE",
    "message": "group.party partially failed (1 failed, 4 succeeded, 2 skipped)",
    "details": {
      "action": "group.party",
      "attempted": 5,
      "succeeded": 4,
      "failed": 1,
      "skipped": 2,
      "results": [
        {
          "action": "join",
          "target": "主卧",
          "ip": "192.168.1.33",
          "error": "timeout"
        }
      ],
      "failures": [
        {
          "target": "主卧",
          "ip": "192.168.1.33",
          "error": "timeout"
        }
      ]
    }
  }
}
```

---

## 6. 对 agent 的判断建议

### 先看 `error.code`

推荐优先级：

1. `ERR_INVALID_ARGUMENT`
2. `ERR_TARGET_*`
3. `ERR_SERVICE_*`
4. `ERR_NO_RESULTS`
5. `ERR_INDEX_OUT_OF_RANGE`
6. `ERR_PARTIAL_FAILURE`
7. `ERR_STATE_INCONSISTENT`
8. `ERR_COMMAND_FAILED`

### 再看 `details`

例如：

- `ERR_TARGET_NOT_FOUND` 配合 `room` / `speakerIP`
- `ERR_NO_RESULTS` 配合 `query` / `category`
- `ERR_PARTIAL_FAILURE` 配合 `results` / `failures`

### 最后才回退 `message`

`message` 适合展示给人看，但不应成为唯一机器判断依据。

---

## 7. 可重试性建议

当前实现还没有显式输出 `retryable` 字段，因此 v1 只能给出建议性判断：

### 通常不该直接重试

- `ERR_INVALID_ARGUMENT`
- `ERR_TARGET_REQUIRED`
- `ERR_QUERY_REQUIRED`
- `ERR_SERVICE_REQUIRED`
- `ERR_SERVICE_NOT_FOUND`
- `ERR_SERVICE_AMBIGUOUS`
- `ERR_UNSUPPORTED_REF`

### 可能需要改参数后重试

- `ERR_TARGET_NOT_FOUND`
- `ERR_TARGET_AMBIGUOUS`
- `ERR_NO_RESULTS`
- `ERR_INDEX_OUT_OF_RANGE`
- `ERR_NOT_FOUND`

### 可能值得自动恢复或重试

- `ERR_PARTIAL_FAILURE`
- `ERR_STATE_INCONSISTENT`
- `ERR_COMMAND_FAILED`

这部分目前只是接入建议，不是响应协议字段。

---

## 8. 当前限制

1. 错误响应还没并入 `execution` envelope
2. 部分错误码仍依赖 message 推断，而不是全量显式构造
3. `details` 还没有更细的 capability-specific schema

即便如此，当前这套 `ERR_*` 规范已经足够支撑：

- agent 分支判断
- 自动回退
- UI 提示分类
- 日志与指标聚合

后续如果继续产品化，建议下一步优先做：

1. 让更多错误走显式 coded error
2. 给错误补 `action`
3. 评估是否把错误也接进 `execution` envelope
