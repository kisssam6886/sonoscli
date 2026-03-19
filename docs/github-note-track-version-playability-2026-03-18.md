# GitHub Note Draft — 同名歌曲“能搜到但不一定能播”已定位到版本层

## 摘要
这次不是单纯修了一个“播放不了”的 bug，而是把一个很容易误判的问题拆清楚了：

- 不是队列加到 10 首就一定坏；
- 不是网易云服务整体不可用；
- 而是同名歌曲的 **不同专辑版本** 在 `Sonos × 网易云` 链路里的可播性不同；
- 同时，客厅还会偶发一次 `TRANSITIONING -> STOPPED/0:00:00` 的首播卡态。

## 为什么重要
如果仓库未来要给任何 agent / bot / automation 调用，就不能再把：

- “搜索命中”
- “metadata 正常”
- `canPlay=true`

当成最终成功条件。

真正的成功条件应该是：

1. 进入 `PLAYING`
2. `RelTime` 明确前进

## 已确认的版本差异

### 郑伊健《极速》
- `Magic`：失败，内置恢复 3 次后仍 `ERR_TRANSITION_STUCK`
- `Discover`：成功
- `The Best Show 3`：成功
- `Friends For Life (新曲+精选)`：成功

### 刘德华《谢谢你的爱》
- `劲歌精选`：更稳，优先级最高
- `经典重现`：可播，但更容易先卡住，再补一次 `play` 恢复
- `谢谢你的爱`：可播，但更容易先卡住，再补一次 `play` 恢复
- `永远爱你`：可播，但更容易先卡住，再补一次 `play` 恢复

## 对 repo 的直接影响
后续 agent 标准流程应该补上：

1. 版本选择策略
2. 黑名单 / 白名单
3. `ERR_TRANSITION_STUCK` 后的单次 `play` fallback
4. 只有真实进入 `PLAYING` 才算播放成功

## 建议后续工程项
1. 为高频歌建立版本优先级缓存
2. 把“稳定可播 / 弱稳定可播 / 不可播”写成统一机器可读结果
3. 把这一层做成 service-agnostic reliability policy，后续可复用到 QQ 音乐 / 酷我 / 其他 Sonos 服务
