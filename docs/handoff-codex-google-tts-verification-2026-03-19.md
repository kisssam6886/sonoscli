# Google TTS 验证记录（2026-03-19）

## 目标

确认 Google Cloud Text-to-Speech 在当前项目中是否已经真正可用，并验证普通话、粤语两条实际合成链路。

## 结论

- Google TTS 现已从“未启用 / service disabled”进入“部分 API key 可用”的状态。
- 当前测试的 3 条 Google API key 中：
  - 2 条仍被 Google 标记为 `API_KEY_SERVICE_BLOCKED`
  - 1 条已可正常访问 `texttospeech.googleapis.com`
- 可用 key 已成功完成以下真实调用：
  - `ListVoices`
  - `text:synthesize` 普通话
  - `text:synthesize` 粤语

这说明：

- 当前 Google Cloud project 已经开通了 Text-to-Speech API
- 但不是所有 key 都有同样权限，后续仓库接入时必须明确使用“已解封的那条 key”或统一整理 key restriction

## 实测结果

### 1. Voices 列表

- 成功返回 `2066` 个 voice
- 中文音色家族确认可见：
  - `cmn-CN-Chirp3-HD-*`
  - `cmn-CN-Standard-*`
  - `cmn-CN-Wavenet-*`
  - `yue-HK-Chirp3-HD-*`
  - `yue-HK-Standard-*`

### 2. 普通话合成

- `cmn-CN-Standard-A` 成功返回音频
- `cmn-CN-Chirp3-HD-Aoede` 成功返回音频

### 3. 粤语合成

- `yue-HK-Standard-A` 成功返回音频
- `yue-HK-Chirp3-HD-Aoede` 成功返回音频

## 关键判断

### 1. Google TTS 已可用

这次不只是“控制台按钮点了启用”，而是 API 已经能真实产出普通话、粤语音频，所以 Google 路线可继续推进。

### 2. 问题从“服务未启用”转成“key 权限不一致”

之前报错是 project 级别 `SERVICE_DISABLED`。  
现在报错已经变成 key 级别 `API_KEY_SERVICE_BLOCKED`，表示服务已开，但 key restriction 仍需梳理。

### 3. 粤语可做正式候选

Google 当前确实有 `yue-HK` 音色，而且 `Chirp3-HD` 也能返回数据，所以它至少具备做 Sonos `say` 粤语播报候选 provider 的基础条件。

## 对仓库的直接影响

当前仓库里的 `say` provider 骨架只做到：

- `macos`
- `azure`

Google 现已具备进入下一步的条件：

1. 为 `say` 增加 `google` provider
2. 支持 `--google-key` 或环境变量读取
3. 默认提供普通话 / 粤语 voice 映射
4. 将生成音频走现有本地 HTTP 短时托管，再交 Sonos 播放

## 建议的默认策略

如果下一步接入 Google provider，建议先这样落：

- 默认 provider:
  - 仍保持 `macos`
- 新增可选 provider:
  - `google`
- 默认 Google voice:
  - 普通话：先用稳定基础 voice，必要时允许切到 `Chirp3-HD`
  - 粤语：优先提供 `yue-HK` 默认 voice，并允许显式指定 `Chirp3-HD-*`

原因：

- 先把“可播、稳定、可调试”做实
- 再决定是否默认押更贵或更高质量的 `Chirp3-HD`

## 风险与限制

- 本次验证没有把任何 API key 写入仓库
- 当前仅验证了 Google 官方 API 可回音频，尚未接入 `sonos say --provider google`
- 当前也未做三家真机听感 A/B：
  - Google
  - Azure
  - OpenAI

## 下一步

建议按这个顺序继续：

1. 把 `say --provider google` 接入仓库
2. 用已可用 key 做一次 Sonos 真机播报
3. 补自动化测试和执行入口透传
4. 再做普通话 / 粤语的主观听感对比
