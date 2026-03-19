package cli

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/appconfig"
	"github.com/steipete/sonoscli/internal/sonos"
	"github.com/steipete/sonoscli/internal/spotify"
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

func TestExecuteCmd_SMAPISearchAliasOpen(t *testing.T) {
	fs := newFakeSonosSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
	})

	store := &memTokenStore{}
	_ = store.Save("2311", "Sonos_TEST", sonos.SMAPITokenPair{AuthToken: "t", PrivateKey: "k"})
	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return store, nil }
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--data", `{
		"action":"smapi.search",
		"target":{"ip":"`+u.Hostname()+`"},
		"request":{"service":"Spotify","category":"tracks","query":"gareth","open":true,"index":1}
	}`)
	if err != nil {
		t.Fatalf("execute smapi search: %v", err)
	}
	if !strings.Contains(out, `"action": "smapi.search"`) || !strings.Contains(out, `"capability": "music.smapi"`) || !strings.Contains(out, `"selectedTitle": "Gareth Emery"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fs.playCalls.Load() == 0 {
		t.Fatalf("expected Play to be called")
	}
}

func TestExecuteCmd_PlaySpotifyAliasInfersEnqueue(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Format: formatJSON}
	cmd := newExecuteCmd(flags)

	searcher := &fakePlaySpotifySearcher{
		result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{ID: "spotify:track:def", ItemType: "track", Title: "男人哭吧不是罪"},
			},
		},
	}
	enq := &fakePlaySpotifyEnqueuer{coordinatorIP: "10.0.0.10", pos: 2}

	origSearcher := newSMAPISearcher
	origEnqueuer := newSpotifyEnqueuer
	t.Cleanup(func() {
		newSMAPISearcher = origSearcher
		newSpotifyEnqueuer = origEnqueuer
	})

	newSMAPISearcher = func(ctx context.Context, flags *rootFlags, serviceName string) (smapiSearcher, sonos.MusicServiceDescriptor, *sonos.Client, error) {
		return searcher, sonos.MusicServiceDescriptor{ID: "9", Name: "Spotify", Auth: "UserId"}, &sonos.Client{IP: "10.0.0.8"}, nil
	}
	newSpotifyEnqueuer = func(ctx context.Context, flags *rootFlags) (spotifyEnqueuer, error) {
		return enq, nil
	}

	out, err := execute(t, cmd, "--data", `{
		"action":"play.spotify",
		"target":{"room":"Kitchen"},
		"request":{"query":"刘德华","enqueueOnly":true}
	}`)
	if err != nil {
		t.Fatalf("execute play.spotify: %v", err)
	}
	if !strings.Contains(out, `"action": "play.spotify"`) || !strings.Contains(out, `"operation": "enqueue"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if enq.calls != 1 || enq.lastOpts.PlayNow {
		t.Fatalf("unexpected enqueue call state: calls=%d playNow=%v", enq.calls, enq.lastOpts.PlayNow)
	}
}

func TestExecuteCmd_SearchSpotifyAliasOpen(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Format: formatJSON}
	cmd := newExecuteCmd(flags)

	origSearcher := newSpotifySearcher
	origEnqueuer := newSonosEnqueuer
	t.Cleanup(func() {
		newSpotifySearcher = origSearcher
		newSonosEnqueuer = origEnqueuer
	})

	newSpotifySearcher = func(flags *rootFlags, clientID, clientSecret string) (spotifySearcher, error) {
		return &fakeSpotifySearcher{results: []spotify.Result{
			{Type: spotify.TypeTrack, ID: "t1", URI: "spotify:track:t1", Title: "Song 1"},
		}}, nil
	}
	fakeSonos := &fakeSonosEnqueuer{}
	newSonosEnqueuer = func(ctx context.Context, flags *rootFlags) (sonosEnqueuer, error) {
		return fakeSonos, nil
	}

	out, err := execute(t, cmd, "--data", `{
		"action":"search.spotify",
		"target":{"room":"Kitchen"},
		"request":{"query":"hello","type":"track","selectionAction":"open","index":1}
	}`)
	if err != nil {
		t.Fatalf("execute search.spotify: %v", err)
	}
	if !strings.Contains(out, `"action": "search.spotify"`) || !strings.Contains(out, `"operation": "search_open"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if fakeSonos.calls != 1 || !fakeSonos.lastOpts.PlayNow {
		t.Fatalf("unexpected sonos call state: calls=%d playNow=%v", fakeSonos.calls, fakeSonos.lastOpts.PlayNow)
	}
}

func TestExecuteCmd_AuthSMAPIBegin(t *testing.T) {
	fs := newFakeSonosSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
	})

	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return &memTokenStore{}, nil }
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{Timeout: time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--data", `{
		"capability":"auth.smapi",
		"operation":"begin",
		"target":{"ip":"`+u.Hostname()+`"},
		"request":{"service":{"name":"Spotify"}}
	}`)
	if err != nil {
		t.Fatalf("execute auth.smapi.begin: %v", err)
	}
	if !strings.Contains(out, `"action": "auth.smapi.begin"`) || !strings.Contains(out, `"operation": "begin"`) || !strings.Contains(out, `"linkCode"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
