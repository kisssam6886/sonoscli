package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakeSourceClient struct {
	setCalls  int
	uri       string
	meta      string
	playCalls int
}

func (f *fakeSourceClient) SetAVTransportURI(ctx context.Context, uri, meta string) error {
	f.setCalls++
	f.uri = uri
	f.meta = meta
	return nil
}

func (f *fakeSourceClient) Play(ctx context.Context) error {
	f.playCalls++
	return nil
}

func TestPlayURICmdRadio(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second}
	cmd := newPlayURICmd(flags)

	fake := &fakeSourceClient{}
	origClient := newSourceClient
	t.Cleanup(func() { newSourceClient = origClient })
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{"--radio", "--title", "My Station", "http://example.com/stream"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.setCalls != 1 || fake.playCalls != 1 {
		t.Fatalf("expected set+play once, got set=%d play=%d", fake.setCalls, fake.playCalls)
	}
	if fake.uri != "x-rincon-mp3radio://example.com/stream" {
		t.Fatalf("unexpected uri: %q", fake.uri)
	}
	if !strings.Contains(fake.meta, "My Station") {
		t.Fatalf("expected meta to include title, got: %q", fake.meta)
	}
}

func TestPlayURICmdJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newPlayURICmd(flags)

	fake := &fakeSourceClient{}
	origClient := newSourceClient
	t.Cleanup(func() { newSourceClient = origClient })
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd, "--radio", "--title", "My Station", "http://example.com/stream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"play-uri\"") || !strings.Contains(out, "\"capability\": \"transport.source\"") || !strings.Contains(out, "\"operation\": \"play-uri\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"radio\": true") || !strings.Contains(out, "\"uri\": \"x-rincon-mp3radio://example.com/stream\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestLineInCmdFrom(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second}
	cmd := newLineInCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Kitchen": {Name: "Kitchen", IP: "192.168.1.11", UUID: "RINCON_K1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.11": {Name: "Kitchen", IP: "192.168.1.11", UUID: "RINCON_K1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{"--from", "Kitchen"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.uri != "x-rincon-stream:RINCON_K1400" {
		t.Fatalf("unexpected uri: %q", fake.uri)
	}
}

func TestLineInCmdJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newLineInCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Kitchen": {Name: "Kitchen", IP: "192.168.1.11", UUID: "RINCON_K1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.11": {Name: "Kitchen", IP: "192.168.1.11", UUID: "RINCON_K1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd, "--from", "Kitchen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"linein\"") || !strings.Contains(out, "\"capability\": \"transport.source\"") || !strings.Contains(out, "\"operation\": \"linein\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"uri\": \"x-rincon-stream:RINCON_K1400\"") || !strings.Contains(out, "\"name\": \"Kitchen\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTVCmd(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second}
	cmd := newTVCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Living Room": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.10": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.uri != "x-sonos-htastream:RINCON_LR1400:spdif" {
		t.Fatalf("unexpected uri: %q", fake.uri)
	}
}

func TestTVCmdJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newTVCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Living Room": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.10": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"tv\"") || !strings.Contains(out, "\"capability\": \"transport.source\"") || !strings.Contains(out, "\"operation\": \"tv\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"uri\": \"x-sonos-htastream:RINCON_LR1400:spdif\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestMusicCmd(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second}
	cmd := newMusicCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Living Room": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.10": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.uri != "x-rincon-queue:RINCON_LR1400#0" {
		t.Fatalf("unexpected uri: %q", fake.uri)
	}
}

func TestMusicCmdJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Living Room", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newMusicCmd(flags)

	top := sonos.Topology{
		ByName: map[string]sonos.Member{
			"Living Room": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
		ByIP: map[string]sonos.Member{
			"192.168.1.10": {Name: "Living Room", IP: "192.168.1.10", UUID: "RINCON_LR1400"},
		},
	}

	origTG := newTopologyGetter
	origClient := newSourceClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newSourceClient = origClient
	})

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeTopologyGetter{top: top}, nil
	}
	fake := &fakeSourceClient{}
	newSourceClient = func(ctx context.Context, flags *rootFlags) (sourceClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"music\"") || !strings.Contains(out, "\"capability\": \"transport.source\"") || !strings.Contains(out, "\"operation\": \"music\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"uri\": \"x-rincon-queue:RINCON_LR1400#0\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}
