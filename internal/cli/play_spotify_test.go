package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakePlaySpotifySearcher struct {
	result sonos.SMAPISearchResult
	err    error
	calls  int
}

func (f *fakePlaySpotifySearcher) Search(ctx context.Context, category, term string, index, count int) (sonos.SMAPISearchResult, error) {
	f.calls++
	return f.result, f.err
}

type fakePlaySpotifyEnqueuer struct {
	coordinatorIP string
	pos           int
	err           error
	calls         int
	lastInput     string
	lastOpts      sonos.EnqueueOptions
}

func (f *fakePlaySpotifyEnqueuer) EnqueueSpotify(ctx context.Context, input string, opts sonos.EnqueueOptions) (int, error) {
	f.calls++
	f.lastInput = input
	f.lastOpts = opts
	if f.pos <= 0 {
		f.pos = 1
	}
	return f.pos, f.err
}

func (f *fakePlaySpotifyEnqueuer) CoordinatorIP() string { return f.coordinatorIP }

func TestRealSpotifyEnqueuer_EnqueueSpotify(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/xml/device_description.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<root>
  <device>
    <deviceType>urn:schemas-upnp-org:device:ZonePlayer:1</deviceType>
    <manufacturer>Sonos, Inc.</manufacturer>
    <roomName>Office</roomName>
    <UDN>uuid:RINCON_OFFICE1400</UDN>
  </device>
</root>`))
	})
	mux.HandleFunc("/MediaRenderer/AVTransport/Control", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "AVTransport:1#AddURIToQueue"):
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:AddURIToQueueResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
      <FirstTrackNumberEnqueued>1</FirstTrackNumberEnqueued>
    </u:AddURIToQueueResponse>
  </s:Body>
</s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#Seek"):
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SeekResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SeekResponse></s:Body></s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#Play"):
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:PlayResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PlayResponse></s:Body></s:Envelope>`))
		default:
			t.Fatalf("unexpected SOAPACTION: %q", action)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())

	c := &sonos.Client{
		IP:   u.Hostname(),
		Port: port,
		HTTP: srv.Client(),
	}

	enq := realSpotifyEnqueuer{c: c}
	if got := enq.CoordinatorIP(); got != c.IP {
		t.Fatalf("CoordinatorIP: %q", got)
	}

	pos, err := enq.EnqueueSpotify(context.Background(), "spotify:track:abc", sonos.EnqueueOptions{
		Title:   "X",
		PlayNow: true,
	})
	if err != nil {
		t.Fatalf("EnqueueSpotify: %v", err)
	}
	if pos != 1 {
		t.Fatalf("expected pos=1, got %d", pos)
	}

	// Ensure wrapper passes through context cancellation cleanly.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.HTTP.Timeout = 10 * time.Second
	_, _ = enq.EnqueueSpotify(ctx, "spotify:track:abc", sonos.EnqueueOptions{Title: "X"})
}

func TestPlaySpotifyJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Format: formatJSON}
	cmd := newPlaySpotifyCmd(flags)

	searcher := &fakePlaySpotifySearcher{
		result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{
					ID:       "spotify:track:abc",
					ItemType: "track",
					Title:    "友情岁月",
				},
			},
		},
	}
	enq := &fakePlaySpotifyEnqueuer{coordinatorIP: "10.0.0.9", pos: 4}

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

	out, err := execute(t, cmd, "郑伊健")
	if err != nil {
		t.Fatalf("play spotify: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"play.spotify\"") || !strings.Contains(out, "\"capability\": \"music.spotify\"") || !strings.Contains(out, "\"operation\": \"play\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"selectedTitle\": \"友情岁月\"") || !strings.Contains(out, "\"enqueuedPos\": 4") {
		t.Fatalf("unexpected output: %q", out)
	}
	if enq.calls != 1 {
		t.Fatalf("expected enqueuer call, got %d", enq.calls)
	}
	if !enq.lastOpts.PlayNow {
		t.Fatalf("expected PlayNow=true")
	}
}

func TestPlaySpotifyEnqueueJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Format: formatJSON}
	cmd := newPlaySpotifyCmd(flags)

	searcher := &fakePlaySpotifySearcher{
		result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{
					ID:       "spotify:track:def",
					ItemType: "track",
					Title:    "男人哭吧不是罪",
				},
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

	out, err := execute(t, cmd, "--enqueue", "刘德华")
	if err != nil {
		t.Fatalf("play spotify --enqueue: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"play.spotify\"") || !strings.Contains(out, "\"capability\": \"music.spotify\"") || !strings.Contains(out, "\"operation\": \"enqueue\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"enqueueOnly\": true") {
		t.Fatalf("unexpected output: %q", out)
	}
	if enq.calls != 1 {
		t.Fatalf("expected enqueuer call, got %d", enq.calls)
	}
	if enq.lastOpts.PlayNow {
		t.Fatalf("expected PlayNow=false")
	}
}
