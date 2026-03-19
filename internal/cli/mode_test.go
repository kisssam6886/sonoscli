package cli

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

func TestModeGetJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.30", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newModeCmd(flags)

	var calls []string
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		calls = append(calls, action)
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "AVTransport:1#GetTransportSettings"):
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "GetTransportSettings", `<PlayMode>SHUFFLE_NOREPEAT</PlayMode><RecQualityMode>NOT_IMPLEMENTED</RecQualityMode>`),
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

	out, err := execute(t, cmd, "get")
	if err != nil {
		t.Fatalf("mode get: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"mode.get\"") || !strings.Contains(out, "\"capability\": \"transport.mode\"") || !strings.Contains(out, "\"operation\": \"get\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"playMode\": \"SHUFFLE_NOREPEAT\"") {
		t.Fatalf("missing playMode: %q", out)
	}
	got := strings.Join(calls, "\n")
	if !strings.Contains(got, "AVTransport:1#GetTransportSettings") {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestModeSetJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.31", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newModeCmd(flags)

	var calls []string
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		calls = append(calls, action)
		switch {
		case strings.Contains(action, "ZoneGroupTopology:1#GetZoneGroupState"):
			return httpResponseWithStatus(500, ""), nil
		case strings.Contains(action, "AVTransport:1#SetPlayMode"):
			return httpResponseWithStatus(
				200,
				soapActionResponse("urn:schemas-upnp-org:service:AVTransport:1", "SetPlayMode", ``),
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

	out, err := execute(t, cmd, "repeat-one")
	if err != nil {
		t.Fatalf("mode repeat-one: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"mode.repeat-one\"") || !strings.Contains(out, "\"capability\": \"transport.mode\"") || !strings.Contains(out, "\"operation\": \"set\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"playMode\": \"REPEAT_ONE\"") {
		t.Fatalf("missing playMode: %q", out)
	}
	got := strings.Join(calls, "\n")
	if !strings.Contains(got, "AVTransport:1#SetPlayMode") {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestModeRejectsUnknownValue(t *testing.T) {
	flags := &rootFlags{IP: "192.0.2.32", Timeout: 2 * time.Second}
	cmd := newModeCmd(flags)

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{"weird"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}
