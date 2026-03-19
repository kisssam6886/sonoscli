# MVP Closure Checklist

## 仓库收口（当前优先）

- [x] 清理 `cmd/tmp_*` 临时探针代码
- [ ] 固化已验证能力为正式接口
- [x] 统一 queue action 模型（append / insert-front / insert-next / replace）
- [ ] 统一 source 模型（artist / chart / playlist / album / search）
- [x] 起草 JSON request / response / error schema
- [x] 写 Agent Quickstart
- [x] 评估并设计 ServiceAdapter 抽象

## 已完成基础

- [x] 网易云 Sonos SMAPI 已打通
- [x] 队列 metadata 空白 / 骨架屏修复
- [x] 亮了但不播修复
- [x] 歌手热门歌插播验证
- [x] 热歌榜前 10 插播验证
- [x] 欧美热歌榜前 10 插播验证
- [x] 方向文档已落地

## 第二阶段（后续再做）

- [ ] `sonos say`
- [ ] `sonos schedule`
- [ ] snapshot / restore / undo
- [ ] QQ 音乐 / 酷我可行性验证
