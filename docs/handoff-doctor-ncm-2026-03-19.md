# Handoff：`doctor ncm` 已落地（2026-03-19）

## 这次加了什么

新增命令：

```bash
./sonos doctor ncm --name "客厅"
./sonos doctor ncm --name "客厅" --query "郑伊健 甘心替代你"
./sonos doctor ncm --name "客厅" --query "郑伊健 甘心替代你" --format json
```

以及对应执行层别名：

```json
{"action":"doctor.ncm","target":{"name":"客厅"},"request":{"query":"郑伊健 甘心替代你"}}
```

## 命令目的

这个命令不是直接播歌，而是给其他 Agent 在真正动手前先做网易云链路体检。

它的目标是先回答这几个问题：

1. 这个房间当前能不能访问 Sonos 里的网易云服务
2. 网易云服务本身有没有挂在当前 Sonos 环境里
3. 当前 household/token 状态大概通不通
4. 本地 playability 黑名单有没有已经拦掉候选
5. 如果给一个精确查询，当前候选里有没有像样的原唱非 Live 版本

## 会输出什么

### 1. target

- 当前房间
- 当前 speaker IP
- 当前 coordinator IP

### 2. service

- 是否找到 `网易云音乐`
- service id
- auth 类型
- household id
- 当前 token store 是否命中
- 当前 categories
- 当前是否 ready 做网易云搜索

### 3. playability

- 本地 playability store 路径
- 总条目数
- 网易云相关条目数
- blocked / weak / stable 数量

### 4. search

如果你传了 `--query`，还会输出：

- 原始 query
- 归一化后的 query
- 原始命中 track 数
- 被 blocked 规则过滤掉多少条
- 当前最终候选数
- 推荐候选
- 前几条候选的诊断信息

候选会额外标出：

- `matchLevel`
- `artistScore`
- `titleScore`
- `adaptationPenalty`
- `livePenalty`

所以其他 Agent 能很快知道：

- 是不是严格命中歌手 + 歌名
- 是不是疑似翻唱 / Remix / DJ
- 是不是 Live

## 和现有规则的关系

这个命令已经复用当前 repo 的两套核心规则：

1. 搜索排序规则
   - 原唱优先
   - 歌名命中优先
   - 翻唱 / Remix / DJ 降权
   - Live 进一步降权

2. 本地 playability store
   - 已被记录成 blocked 的版本会统计出来
   - 如有 `--query`，会显示被过滤掉的 blocked 数量

## 当前代码位置

- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor_ncm.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor_ncm_test.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/doctor.go`
- `/Users/sam/.openclaw/workspace-dev/sonoscli-plus/internal/cli/execute.go`

## 已通过的验证

### 本地验证

已通过：

```bash
go test ./internal/cli ./internal/sonos
go build -o ./sonos ./cmd/sonos
```

### 单测覆盖了什么

1. 正常返回精确推荐候选
2. blocked 规则把唯一候选拦掉时，命令会返回 `ok=false`
3. `execute` 别名 `doctor.ncm` 可以正常工作

## 当前真机限制

今天在本机现场继续验证时，遇到的是**网络 / Sonos 可达性问题**，不是 `doctor ncm` 逻辑崩溃：

1. 一次用 `--name "客厅"` 时，SSDP discover 返回 `no speakers found`
2. 一次用已知 IP `10.10.10.31` 时，访问 `http://10.10.10.31:1400/MusicServices/Control` 返回 `connect: connection refused`

这说明此刻的限制在现场网络层或设备可达性，不在命令结构本身。

所以当前结论要分开看：

- `doctor ncm` 命令本身：已实现、已编译、已单测通过
- 现场 Sonos 可达性：这次验证窗口里不稳定，需要等设备网络恢复后再做完整真机回归

## 建议下一位 Agent 怎么用

### 标准前置命令

先跑：

```bash
cd /Users/sam/.openclaw/workspace-dev/sonoscli-plus
go build -o ./sonos ./cmd/sonos
./sonos doctor
./sonos doctor room --name "客厅" --format json
./sonos doctor ncm --name "客厅" --query "郑伊健 甘心替代你" --format json
```

### 判断原则

如果 `doctor ncm` 返回：

- `service.found = true`
- `service.searchReady = true`
- `search.recommended` 存在
- `matchLevel` 是 `exact_nonlive` 或至少 `exact`

那才适合继续做网易云真机播放测试。

如果看到：

- `blockedFiltered > 0`
- 推荐候选是 `Live`
- 推荐候选不是原唱精确命中

先不要直接开播，应该先换查询词或重新核对候选。

## 给下一位 Agent 的一句话

以后别再只用“搜得到”来判断网易云链路是否可用，先跑 `doctor ncm`。它是“网易云服务 + 本地 blocked 规则 + 候选质量”的统一门禁。
