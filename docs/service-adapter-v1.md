# ServiceAdapter Draft v1

## 目标

当前仓库已经打通 `网易云音乐`，但长期方向不应把所有逻辑都写死在 `ncm_*` 上。

本草案定义一个 **ServiceAdapter** 抽象方向，让未来支持：
- 网易云音乐
- QQ 音乐
- 酷我音乐
- 其他 Sonos 中国区支持服务

同时复用同一套：
- Sonos target 选择
- queue action
- playback reliability
- metadata handling
- JSON output

---

## 1. 分层思路

### A. Sonos Core Layer（通用）
负责：
- discover / target room / coordinator
- queue inspect / clear / play position
- append / insert-front / insert-next / replace
- playback retry / transition heal
- queue snapshot / restore（未来）

### B. Music Service Adapter Layer（按服务）
负责：
- service auth / token
- browse / search / get metadata
- chart / artist / playlist / album / track mapping
- service-specific source normalization

### C. Agent-facing Request Layer（统一）
负责：
- 把 `target/source/action/strategy` 转成一次标准请求
- 调用对应 adapter
- 返回统一结果结构

---

## 2. 建议的最小接口

> 这里是设计草案，不要求一步到位实现。

```go
type ServiceAdapter interface {
    ServiceKey() string
    ServiceName() string

    Search(ctx context.Context, req SearchRequest) (SearchResult, error)
    Browse(ctx context.Context, req BrowseRequest) (BrowseResult, error)
    ResolveTracks(ctx context.Context, req ResolveRequest) ([]ResolvedTrack, error)
}
```

### SearchRequest
```go
type SearchRequest struct {
    Query string
    Type  string // artist / track / album / playlist
    Limit int
}
```

### BrowseRequest
```go
type BrowseRequest struct {
    SourceType string // chart / playlist / album / container
    ID         string
    Name       string
    Limit      int
}
```

### ResolveRequest
```go
type ResolveRequest struct {
    SourceType string // artist / chart / playlist / album / search / track
    Name       string
    ID         string
    Query      string
    Limit      int
    Strategy   ResolveStrategy
}
```

### ResolvedTrack
```go
type ResolvedTrack struct {
    Service      string
    ID           string
    Title        string
    Artist       string
    Album        string
    AlbumArtURI  string
    DurationSec  int
    CanPlay      bool
    CanSkip      bool
}
```

---

## 3. 为什么要有 ResolveTracks

当前从 agent / CLI 视角，最常见请求其实不是“我要调用 search 还是 browse”，而是：

- 给我刘德华 5 首可播歌
- 给我热歌榜前 10
- 给我欧美热歌榜前 10
- 给我某歌单前 20 首

所以 adapter 层应该暴露一个更接近上层语义的入口：

> **ResolveTracks**

它内部可以决定：
- 走 search
- 走 browse
- 走 getMetadata
- 走 chart container

但上层不需要知道这些细节。

---

## 4. 当前仓库怎么逐步演进

### 当前现实
- `ncm_*` 逻辑已经可用
- 现有代码大量集中在 `internal/cli/ncm_play.go`
- 还不适合大重写

### 建议的渐进路线

#### 第一步
先抽一个 **NeteaseAdapter** 草案，不立即动全部命令。

#### 第二步
把：
- 歌手热门歌
- 榜单前 N
- search 结果取 track

这些能力逐步从 CLI 逻辑里搬到 adapter 层。

#### 第三步
CLI 只负责：
- 解析参数
- 选择 target
- 选择 action
- 调 adapter resolve tracks
- 调 queue action 执行

---

## 5. 建议目录方向（草案）

```text
internal/
  sonos/                 # Sonos core control
  services/
    netease/
      adapter.go
      source_chart.go
      source_artist.go
      source_search.go
    qqmusic/
      adapter.go
    kuwo/
      adapter.go
  queue/
    actions.go           # append / insert-front / insert-next / replace
    request.go
    result.go
```

> 当前阶段不建议立刻大搬迁，只把这当成目标结构。

---

## 6. 第一版先做什么最务实

### 建议优先做
1. 定义 adapter interface
2. 先做 `netease` 的最小实现
3. 把 chart / artist / search resolve 统一到 `ResolveTracks`
4. 让 queue action 层消费 `[]ResolvedTrack`

这样以后接 QQ 音乐 / 酷我时：
- 不用重写 queue / Sonos 核心逻辑
- 只要新 service 实现 adapter

---

## 7. 结论

ServiceAdapter 的目标不是为了“抽象而抽象”，而是为了：
- 让项目不被 `ncm_*` 锁死
- 让多服务支持可持续
- 让 agent-facing 接口更统一
- 让代码结构从“命令脚本集合”走向“可接入能力层”
