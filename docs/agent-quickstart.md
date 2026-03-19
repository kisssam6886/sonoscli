# Agent Quickstart

## 目标

让任何 agent / bot / automation 在 **不了解 Sonos / SMAPI 内部细节** 的情况下，也能快速调用 `sonoscli-plus` 完成实际任务。

---

## 1. 当前项目定位

`sonoscli-plus` 是一个 **Sonos × 中国大陆音乐服务的能力层 / control layer**。

它负责：
- 选房间
- 搜内容
- 编排队列
- 控制播放
- 返回稳定 JSON

它不负责：
- 自然语言理解
- 对话上下文推理
- Prompt 设计

这些交给上层 agent 完成。

---

## 2. Agent 接入思路

### 第一步：把自然语言翻译成结构化意图

例如用户说：
- “插播热歌榜前10”
- “播郑秀文”
- “加5首刘德华唔好清空队列”

上层 agent 应先翻译成类似：
- target = 房间
- source = artist/chart/search
- action = insert-front / append / replace
- strategy = limit / top / random / dedupe

### 第二步：调用 CLI

当前阶段优先使用现有命令，而不是等待统一抽象入口全部完成。

---

## 3. 现阶段最可用的能力

### A. 状态与目标
```bash
./sonos discover --format json
./sonos status --name "浴室" --format json
./sonos queue list --name "浴室" --format json
```

### B. 网易云搜索 / 浏览
```bash
./sonos ncm search --name "浴室" --format json 郑秀文
./sonos ncm browse --name "浴室" --id TOPBOARD --format json
```

### C. 网易云播放（当前已验证稳定）
```bash
./sonos ncm play --name "浴室" 郑秀文 --index 1 --format json
./sonos ncm lucky --name "浴室" 郑秀文 --format json
```

### D. 队列操作
```bash
./sonos queue list --name "浴室" --format json
./sonos queue play --name "浴室" 1
./sonos queue clear --name "浴室"
```

---

## 4. 当前已验证可工作的模式

### 模式 1：按歌手播并自动排队
- 输入：歌手名
- 行为：搜索匹配曲目，重建/设置队列并播放
- 适合：`播郑秀文`

### 模式 2：追加到现有队列
- 输入：歌手 / 榜单
- 行为：保留当前队列，在后面加歌
- 适合：`加5首刘德华`

### 模式 3：插播到前面
- 输入：歌手 / 榜单
- 行为：插到当前播放后方的前排
- 适合：`插播热歌榜前10`

---

## 5. Agent 最佳实践

### 1. 永远先拿 target room
不要写死房间。

### 2. 优先走 JSON
尽量用 `--format json`，不要依赖纯文本输出解析。

### 3. 区分 append / insert / replace
不要把所有“播”都翻译成清空重建。

### 4. 先确认内容来源类型
- 歌手
- 榜单
- 搜索
- 歌单
- 专辑

### 5. 遇到 Sonos 队列显示问题，不要重试老旧 favorites 路线
当前稳定主线是基于已修复的 NCM queue metadata 路径。

---

## 6. 推荐给 agent 的意图模型

建议上层 agent 内部至少维护：

```json
{
  "target": { "room": "浴室" },
  "source": { "type": "artist", "name": "刘德华" },
  "action": "append",
  "strategy": { "limit": 5 }
}
```

这样以后仓库升级到统一 schema 时，agent 侧不需要大改。

---

## 7. 当前限制

- 统一的 `queue request` 抽象命令还在设计阶段
- 多服务 adapter（QQ 音乐 / 酷我）尚未开始正式接入
- `sonos say` / `sonos schedule` 仍属于下一阶段目标

---

## 8. 下一步对 agent 开发者最重要的事

1. 先用当前已稳定命令接起来
2. 尽量按 `target/source/action/strategy` 自己做一层内部抽象
3. 关注后续统一 JSON schema / ServiceAdapter / queue request 抽象入口
