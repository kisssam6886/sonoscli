package cli

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

func TestSayCmd_AudioURIJSONIncludesExecutionEnvelopeAndRestoresVolume(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.40", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newSayCmd(flags)

	var (
		playCalls     int
		sawCurrentURI string
		sawMeta       string
		setVolumes    []string
	)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "RenderingControl:1#GetVolume"):
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:RenderingControl:1", "GetVolume", `<CurrentVolume>33</CurrentVolume>`),
			), nil
		case strings.Contains(action, "RenderingControl:1#SetVolume"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			bodyStr := string(body)
			start := strings.Index(bodyStr, "<DesiredVolume>")
			end := strings.Index(bodyStr, "</DesiredVolume>")
			if start >= 0 && end > start {
				start += len("<DesiredVolume>")
				setVolumes = append(setVolumes, bodyStr[start:end])
			}
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:RenderingControl:1", "SetVolume", ``),
			), nil
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			bodyStr := string(body)
			if start := strings.Index(bodyStr, "<CurrentURI>"); start >= 0 {
				start += len("<CurrentURI>")
				if end := strings.Index(bodyStr, "</CurrentURI>"); end > start {
					sawCurrentURI = bodyStr[start:end]
				}
			}
			if start := strings.Index(bodyStr, "<CurrentURIMetaData>"); start >= 0 {
				start += len("<CurrentURIMetaData>")
				if end := strings.Index(bodyStr, "</CurrentURIMetaData>"); end > start {
					sawMeta = bodyStr[start:end]
				}
			}
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "SetAVTransportURI", ``),
			), nil
		case strings.Contains(action, "AVTransport:1#Play"):
			playCalls++
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "Play", ``),
			), nil
		default:
			t.Fatalf("unexpected action: %q", action)
			return nil, nil
		}
	})

	oldNew := newSonosClient
	t.Cleanup(func() { newSonosClient = oldNew })
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{
			IP:   ip,
			Port: 1400,
			HTTP: &http.Client{Timeout: timeout, Transport: rt},
		}
	}

	out, err := execute(t, cmd, "--audio-uri", "http://example.com/tts.aiff", "--title", "测试广播", "--temp-volume", "20", "--hold-seconds", "1", "你好，客厅")
	if err != nil {
		t.Fatalf("say audio-uri: %v", err)
	}
	if playCalls != 1 {
		t.Fatalf("expected Play once, got %d", playCalls)
	}
	if sawCurrentURI != "http://example.com/tts.aiff" {
		t.Fatalf("unexpected CurrentURI: %q", sawCurrentURI)
	}
	if sawMeta != "" {
		t.Fatalf("expected no radio metadata for direct-file playback, got %q", sawMeta)
	}
	if len(setVolumes) != 2 || setVolumes[0] != "20" || setVolumes[1] != "33" {
		t.Fatalf("expected volume set to 20 then restore 33, got %#v", setVolumes)
	}
	if !strings.Contains(out, `"action": "say"`) || !strings.Contains(out, `"capability": "say"`) || !strings.Contains(out, `"operation": "announce"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"inputMode": "audio-uri"`) || !strings.Contains(out, `"usedAudioURI": "http://example.com/tts.aiff"`) {
		t.Fatalf("unexpected request envelope: %s", out)
	}
}

func TestExecuteCmd_SayAnnounceWithAudioURI(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	var (
		playCalls     int
		sawCurrentURI string
	)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			bodyStr := string(body)
			if start := strings.Index(bodyStr, "<CurrentURI>"); start >= 0 {
				start += len("<CurrentURI>")
				if end := strings.Index(bodyStr, "</CurrentURI>"); end > start {
					sawCurrentURI = bodyStr[start:end]
				}
			}
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "SetAVTransportURI", ``),
			), nil
		case strings.Contains(action, "AVTransport:1#Play"):
			playCalls++
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "Play", ``),
			), nil
		default:
			t.Fatalf("unexpected action: %q", action)
			return nil, nil
		}
	})

	oldNew := newSonosClient
	t.Cleanup(func() { newSonosClient = oldNew })
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{
			IP:   ip,
			Port: 1400,
			HTTP: &http.Client{Timeout: timeout, Transport: rt},
		}
	}

	out, err := execute(t, cmd, "--data", `{
		"capability":"say",
		"operation":"announce",
		"target":{"ip":"192.0.2.41"},
		"request":{
			"text":"测试 execute say",
			"audioURI":"http://example.com/say.mp3",
			"radio":false,
			"holdSeconds":1
		}
	}`)
	if err != nil {
		t.Fatalf("execute say: %v", err)
	}
	if playCalls != 1 {
		t.Fatalf("expected Play once, got %d", playCalls)
	}
	if sawCurrentURI != "http://example.com/say.mp3" {
		t.Fatalf("unexpected CurrentURI: %q", sawCurrentURI)
	}
	if !strings.Contains(out, `"action": "say"`) || !strings.Contains(out, `"capability": "say"`) || !strings.Contains(out, `"operation": "announce"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"inputMode": "audio-uri"`) || !strings.Contains(out, `"usedAudioURI": "http://example.com/say.mp3"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSayCmd_AudioURICanForceRadioMode(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.43", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newSayCmd(flags)

	var (
		playCalls     int
		sawCurrentURI string
		sawMeta       string
	)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			bodyStr := string(body)
			if start := strings.Index(bodyStr, "<CurrentURI>"); start >= 0 {
				start += len("<CurrentURI>")
				if end := strings.Index(bodyStr, "</CurrentURI>"); end > start {
					sawCurrentURI = bodyStr[start:end]
				}
			}
			if start := strings.Index(bodyStr, "<CurrentURIMetaData>"); start >= 0 {
				start += len("<CurrentURIMetaData>")
				if end := strings.Index(bodyStr, "</CurrentURIMetaData>"); end > start {
					sawMeta = bodyStr[start:end]
				}
			}
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "SetAVTransportURI", ``),
			), nil
		case strings.Contains(action, "AVTransport:1#Play"):
			playCalls++
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "Play", ``),
			), nil
		default:
			t.Fatalf("unexpected action: %q", action)
			return nil, nil
		}
	})

	oldNew := newSonosClient
	t.Cleanup(func() { newSonosClient = oldNew })
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{
			IP:   ip,
			Port: 1400,
			HTTP: &http.Client{Timeout: timeout, Transport: rt},
		}
	}

	out, err := execute(t, cmd, "--audio-uri", "http://example.com/tts.aiff", "--radio", "--title", "测试广播", "--hold-seconds", "1", "你好，客厅")
	if err != nil {
		t.Fatalf("say audio-uri radio mode: %v", err)
	}
	if playCalls != 1 {
		t.Fatalf("expected Play once, got %d", playCalls)
	}
	if sawCurrentURI != "x-rincon-mp3radio://example.com/tts.aiff" {
		t.Fatalf("unexpected CurrentURI: %q", sawCurrentURI)
	}
	if !strings.Contains(sawMeta, "测试广播") {
		t.Fatalf("expected radio metadata to include title, got %q", sawMeta)
	}
	if !strings.Contains(out, `"radio": true`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSayDefaults_SelectVoiceAndRateByLanguageAndStyle(t *testing.T) {
	t.Parallel()

	if got := defaultVoiceFor("yue"); got != "Sinji" {
		t.Fatalf("defaultVoiceFor(yue) = %q", got)
	}
	if got := defaultVoiceFor("zh"); got != "Tingting" {
		t.Fatalf("defaultVoiceFor(zh) = %q", got)
	}
	if got := defaultRateFor("warm"); got != 168 {
		t.Fatalf("defaultRateFor(warm) = %d", got)
	}
	if got := defaultRateFor("cheerful"); got != 188 {
		t.Fatalf("defaultRateFor(cheerful) = %d", got)
	}
	if got := defaultRateFor("normal"); got != 178 {
		t.Fatalf("defaultRateFor(normal) = %d", got)
	}
}

func TestLocalReachableIPForRequiresTarget(t *testing.T) {
	t.Parallel()

	_, err := localReachableIPFor("")
	if err == nil || !strings.Contains(err.Error(), "missing target speaker IP") {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestSayCmd_RespectsContextCancelDuringHold(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.42", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newSayCmd(flags)

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "SetAVTransportURI", ``),
			), nil
		case strings.Contains(action, "AVTransport:1#Play"):
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "Play", ``),
			), nil
		default:
			t.Fatalf("unexpected action: %q", action)
			return nil, nil
		}
	})

	oldNew := newSonosClient
	t.Cleanup(func() { newSonosClient = oldNew })
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{
			IP:   ip,
			Port: 1400,
			HTTP: &http.Client{Timeout: timeout, Transport: rt},
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{"--audio-uri", "http://example.com/hold.mp3", "--radio=false", "--hold-seconds", "5", "短播报"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("say with cancelled hold: %v", err)
	}
}
