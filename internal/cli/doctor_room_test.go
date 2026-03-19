package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakeDoctorRoomTopologyGetter struct {
	top sonos.Topology
	err error
}

func (f *fakeDoctorRoomTopologyGetter) GetTopology(ctx context.Context) (sonos.Topology, error) {
	return f.top, f.err
}

type fakeDoctorRoomClient struct {
	media     sonos.MediaInfo
	position  sonos.PositionInfo
	transport sonos.TransportInfo
	volume    int
	mute      bool
	err       error
}

func (f *fakeDoctorRoomClient) GetMediaInfo(ctx context.Context) (sonos.MediaInfo, error) {
	return f.media, f.err
}

func (f *fakeDoctorRoomClient) GetPositionInfo(ctx context.Context) (sonos.PositionInfo, error) {
	return f.position, f.err
}

func (f *fakeDoctorRoomClient) GetTransportInfo(ctx context.Context) (sonos.TransportInfo, error) {
	return f.transport, f.err
}

func (f *fakeDoctorRoomClient) GetVolume(ctx context.Context) (int, error) {
	return f.volume, f.err
}

func (f *fakeDoctorRoomClient) GetMute(ctx context.Context) (bool, error) {
	return f.mute, f.err
}

func TestDoctorRoomCmd_JSONSummarizesTVRestoreReadiness(t *testing.T) {
	origTG := newTopologyGetter
	origClient := newDoctorRoomClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newDoctorRoomClient = origClient
	})

	target := sonos.Member{Name: "客厅", IP: "10.0.0.31", UUID: "RINCON_LIVING", IsVisible: true, IsCoordinator: true}
	top := sonos.Topology{
		ByName: map[string]sonos.Member{"客厅": target},
		ByIP:   map[string]sonos.Member{"10.0.0.31": target},
		Groups: []sonos.Group{
			{
				ID:          "RINCON_LIVING:1",
				Coordinator: target,
				Members:     []sonos.Member{target},
			},
		},
	}

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeDoctorRoomTopologyGetter{top: top}, nil
	}
	newDoctorRoomClient = func(ctx context.Context, flags *rootFlags) (doctorRoomClient, error) {
		return &fakeDoctorRoomClient{
			media: sonos.MediaInfo{
				CurrentURI: "x-sonos-htastream:RINCON_LIVING:spdif",
			},
			position: sonos.PositionInfo{
				Track:         "1",
				TrackURI:      "x-sonos-htastream:RINCON_LIVING:spdif",
				TrackDuration: "NOT_IMPLEMENTED",
				RelTime:       "NOT_IMPLEMENTED",
			},
			transport: sonos.TransportInfo{
				State: "PLAYING",
			},
			volume: 16,
			mute:   false,
		}, nil
	}

	flags := &rootFlags{Name: "客厅", Timeout: 2 * time.Second, Format: formatJSON}
	out, err := execute(t, newDoctorRoomCmd(flags))
	if err != nil {
		t.Fatalf("doctor room: %v", err)
	}
	if !strings.Contains(out, `"action": "doctor.room"`) || !strings.Contains(out, `"operation": "room"`) {
		t.Fatalf("unexpected envelope output: %s", out)
	}
	if !strings.Contains(out, `"kind": "tv"`) || !strings.Contains(out, `"restorePath": "direct"`) {
		t.Fatalf("unexpected source output: %s", out)
	}
	if !strings.Contains(out, `"canTestSayRestore": true`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("unexpected readiness output: %s", out)
	}
}

func TestDoctorRoomCmd_JSONWarnsWhenQueueBaselineIsStopped(t *testing.T) {
	origTG := newTopologyGetter
	origClient := newDoctorRoomClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newDoctorRoomClient = origClient
	})

	target := sonos.Member{Name: "客厅", IP: "10.0.0.31", UUID: "RINCON_LIVING", IsVisible: true, IsCoordinator: true}
	top := sonos.Topology{
		ByName: map[string]sonos.Member{"客厅": target},
		ByIP:   map[string]sonos.Member{"10.0.0.31": target},
		Groups: []sonos.Group{
			{
				ID:          "RINCON_LIVING:1",
				Coordinator: target,
				Members:     []sonos.Member{target},
			},
		},
	}

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeDoctorRoomTopologyGetter{top: top}, nil
	}
	newDoctorRoomClient = func(ctx context.Context, flags *rootFlags) (doctorRoomClient, error) {
		return &fakeDoctorRoomClient{
			media: sonos.MediaInfo{
				CurrentURI: "x-rincon-queue:RINCON_LIVING#0",
			},
			position: sonos.PositionInfo{
				Track:         "1",
				TrackURI:      "x-sonos-http:SONG%3ATEST.mp3",
				TrackDuration: "0:03:17",
				RelTime:       "0:00:00",
			},
			transport: sonos.TransportInfo{
				State: "STOPPED",
			},
			volume: 16,
			mute:   false,
		}, nil
	}

	flags := &rootFlags{Name: "客厅", Timeout: 2 * time.Second, Format: formatJSON}
	out, err := execute(t, newDoctorRoomCmd(flags))
	if err != nil {
		t.Fatalf("doctor room: %v", err)
	}
	if !strings.Contains(out, `"restorePath": "queue"`) || !strings.Contains(out, `"canTestSayRestore": false`) {
		t.Fatalf("unexpected queue readiness output: %s", out)
	}
	if !strings.Contains(out, `transport is not PLAYING`) || !strings.Contains(out, `"ok": false`) {
		t.Fatalf("expected warning output, got: %s", out)
	}
}

func TestExecuteCmd_DoctorRoomActionAlias(t *testing.T) {
	origTG := newTopologyGetter
	origClient := newDoctorRoomClient
	t.Cleanup(func() {
		newTopologyGetter = origTG
		newDoctorRoomClient = origClient
	})

	target := sonos.Member{Name: "客厅", IP: "10.0.0.31", UUID: "RINCON_LIVING", IsVisible: true, IsCoordinator: true}
	top := sonos.Topology{
		ByName: map[string]sonos.Member{"客厅": target},
		ByIP:   map[string]sonos.Member{"10.0.0.31": target},
		Groups: []sonos.Group{
			{
				ID:          "RINCON_LIVING:1",
				Coordinator: target,
				Members:     []sonos.Member{target},
			},
		},
	}

	newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
		return &fakeDoctorRoomTopologyGetter{top: top}, nil
	}
	newDoctorRoomClient = func(ctx context.Context, flags *rootFlags) (doctorRoomClient, error) {
		return &fakeDoctorRoomClient{
			media: sonos.MediaInfo{CurrentURI: "x-sonos-htastream:RINCON_LIVING:spdif"},
			position: sonos.PositionInfo{
				Track:         "1",
				TrackURI:      "x-sonos-htastream:RINCON_LIVING:spdif",
				TrackDuration: "NOT_IMPLEMENTED",
				RelTime:       "NOT_IMPLEMENTED",
			},
			transport: sonos.TransportInfo{State: "PLAYING"},
			volume:    16,
			mute:      false,
		}, nil
	}

	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--data", `{"action":"doctor.room","target":{"name":"客厅"}}`)
	if err != nil {
		t.Fatalf("execute doctor.room: %v", err)
	}
	if !strings.Contains(out, `"action": "doctor.room"`) || !strings.Contains(out, `"sourceKind": "tv"`) {
		t.Fatalf("unexpected execute output: %s", out)
	}
}
