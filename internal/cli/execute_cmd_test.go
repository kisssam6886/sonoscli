package cli

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/appconfig"
	"github.com/steipete/sonoscli/internal/sonos"
)

func TestExecuteCmd_DoctorInlineJSON(t *testing.T) {
	oldCollect := collectDoctorReport
	t.Cleanup(func() { collectDoctorReport = oldCollect })
	collectDoctorReport = func() (doctorReport, error) {
		return doctorReport{
			OK:      true,
			Version: "test",
			Capabilities: doctorCapabilities{
				SMAPISongOpenQueuePath: true,
			},
		}, nil
	}

	flags := &rootFlags{Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--data", `{"capability":"doctor","operation":"report"}`)
	if err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
	if !strings.Contains(out, `"action": "doctor"`) || !strings.Contains(out, `"capability": "doctor"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestExecuteCmd_ModeActionAliasUsesTargetIP(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	var seenIP string
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
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
		seenIP = ip
		return &sonos.Client{
			IP:   ip,
			Port: 1400,
			HTTP: &http.Client{Timeout: timeout, Transport: rt},
		}
	}

	out, err := execute(t, cmd, "--data", `{"action":"mode.repeat-one","target":{"ip":"192.0.2.88"}}`)
	if err != nil {
		t.Fatalf("execute mode action alias: %v", err)
	}
	if seenIP != "192.0.2.88" {
		t.Fatalf("seenIP = %q", seenIP)
	}
	if !strings.Contains(out, `"action": "mode.repeat-one"`) || !strings.Contains(out, `"playMode": "REPEAT_ONE"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestExecuteCmd_DiscoverFromFile(t *testing.T) {
	oldDiscover := discoverFunc
	t.Cleanup(func() { discoverFunc = oldDiscover })
	discoverFunc = func(ctx context.Context, opts sonos.DiscoverOptions) ([]sonos.Device, error) {
		return []sonos.Device{
			{Name: "Kitchen", IP: "192.168.1.20", UDN: "uuid:RINCON_KITCHEN"},
		}, nil
	}

	file, err := os.CreateTemp(t.TempDir(), "execute-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(`{"capability":"discover","operation":"scan","request":{"all":true}}`); err != nil {
		t.Fatalf("WriteString: %v", err)
	}

	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--file", file.Name())
	if err != nil {
		t.Fatalf("execute discover file: %v", err)
	}
	if !strings.Contains(out, `"action": "discover"`) || !strings.Contains(out, `"count": 1`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestExecuteArgs_ExecuteJSONErrorWritesEnvelope(t *testing.T) {
	oldLoad := loadAppConfig
	oldCollect := collectDoctorReport
	t.Cleanup(func() {
		loadAppConfig = oldLoad
		collectDoctorReport = oldCollect
	})
	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{Format: "plain"}.Normalize(), nil
	}
	collectDoctorReport = func() (doctorReport, error) {
		return doctorReport{}, nil
	}

	root, flags, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}

	var stdout captureWriter
	var stderr captureWriter
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	err = ExecuteArgs(root, flags, []string{"execute", "--format", "json", "--data", `{"capability":"queue","operation":"play"}`})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(stdout.String(), `"ok": false`) || !strings.Contains(stdout.String(), `"code": "ERR_INVALID_ARGUMENT"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"field": "pos"`) {
		t.Fatalf("expected missing pos detail, got: %s", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}
