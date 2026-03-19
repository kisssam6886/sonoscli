# Handoff — 修复 Sonos `say` 本地 TTS 有返回但无声

时间：2026-03-19  
仓库：`/Users/sam/.openclaw/workspace-dev/sonoscli-plus`

## 问题现象

用户在客厅执行：

```bash
./sonos --name 客厅 say --lang yue "..."
```

CLI 返回：

- `ok: true`
- `action: "say"`
- `capability: "say"`

但人耳没有听到播报。

---

## 实测发现

这次不是单一问题，而是两层问题叠在一起：

1. 旧的 auto-TTS 播放路径直接生成并播放 `AIFF`
   - 客厅会切到本地 URI
   - 但很快变成 `STOPPED`

2. 代码里默认的 macOS voice 名字过旧
   - 代码原来用：
     - `Sin-ji`
     - `Ting-Ting`
   - 当前机器上可用 voice 名字实际是：
     - `Sinji`
     - `Tingting`

更关键的是，这不只是“名字显示不同”，而是实际生成结果差很多。

实测例子：

1. `Sin-ji`
   - 对长粤语句子只生成约 `0.011s`

2. `Sinji`
   - 对同一句能生成约 `9.88s`

所以之前“命令成功但几乎无声”，并不一定是 Sonos 没播，而可能是本地 TTS 本身就生成了近乎听不到的超短音频。

---

## 对照实验

为了把根因钉死，做过这些对照：

1. `say` 原逻辑生成本地 `tts.aiff`
   - Sonos source 确实切过去
   - 但状态很快变 `STOPPED`

2. 同样房间，播放公网 `mp3`
   - 可正常进入播放链路

3. 手工把本地 TTS 转成 `m4a`，再经本地 HTTP 提供给 Sonos
   - 客厅状态进入 `PLAYING`
   - `duration` / `RelTime` 正常前进

4. 用最小 Go 静态服务器提供同一个 `m4a`
   - 也能 `PLAYING`

结论：

1. 不是客厅路由到本机失败
2. 不是 Sonos 不会拉本地 HTTP
3. 主要问题在：
   - auto-TTS 原来直接用 `AIFF`
   - 默认 voice 名字过旧，导致生成内容异常短

---

## 代码修复

本轮实际改动：

1. `internal/cli/say.go`
   - auto-TTS 不再直接播放 `AIFF`
   - 改为：
     - `say` 先生成 `tts.aiff`
     - 再用 `afconvert` 转成 `tts.m4a`
     - 最终对 Sonos 提供 `audio/mp4`

2. `internal/cli/say.go`
   - `say --radio` 默认值从 `true` 改成 `false`
   - 因为 TTS 属于短音频文件，不是持续电台流

3. `internal/cli/say.go`
   - 默认 voice 更新为当前系统可用名：
     - `yue` -> `Sinji`
     - `zh` -> `Tingting`

4. `internal/cli/say_test.go`
   - 更新默认 voice 断言
   - 补 `audio-uri` 直播 / 强制 radio 模式测试

---

## 真机验证

修复后在客厅执行长粤语播报：

```bash
./sonos --format json --name 客厅 say --lang yue --temp-volume 30 --hold-seconds 8 "客厅自动播报长测试。客厅自动播报长测试。客厅自动播报长测试。客厅自动播报长测试。"
```

得到关键结果：

1. `say` 输出：
   - `usedAudioURI = http://10.10.10.129:51820/tts.m4a`
   - `voice = "Sinji"`
   - `radio = false`

2. 播放中 `status`：
   - `state = "PLAYING"`
   - `title = "tts.m4a"`
   - `duration = "0:00:09"`
   - `time = "0:00:09"`

这说明：

1. auto-TTS 现在实际走的是 `m4a`
2. 客厅 Sonos 能正常拉取并播放
3. 这次修复已经从代码层和真机层都验证通过

---

## 后续建议

下一步建议继续做两件事：

1. 给 `say` 增加一份更正式的 runbook
   - 哪些 voice 名字在当前 macOS 有效
   - 什么时候该开 `--radio`
   - 什么时候该用 `--audio-uri`

2. 后面如果要做商业化
   - 尽量不要硬编码旧 voice 别名
   - 最好加一个本机 voice discovery / fallback
   - 让 agent 能在第一次运行时自动探测：
     - `Sinji`
     - `Tingting`
     - 其他可用中文 voice

一句话总结：

> `say` 之前“成功但无声”，不是一个 bug，而是“AIFF 路径不稳 + 旧 voice 名字生成异常短音频”两个问题叠加；现在已改成 `m4a + 正确 macOS voice 名`，并已在客厅真机验证到 `PLAYING`。
