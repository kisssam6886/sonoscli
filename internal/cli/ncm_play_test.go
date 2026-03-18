package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakeNCMSearcher struct {
	result sonos.SMAPISearchResult
	err    error
}

func (f *fakeNCMSearcher) Search(ctx context.Context, category, term string, index, count int) (sonos.SMAPISearchResult, error) {
	return f.result, f.err
}

func TestNCMPlayJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Format: formatJSON, Name: "客厅"}
	cmd := newNCMPlayCmd(flags)

	origSearcher := newSMAPISearcher
	origCoordinator := newNCMCoordinatorClient
	origRebuild := executeNCMQueueRebuild
	t.Cleanup(func() {
		newSMAPISearcher = origSearcher
		newNCMCoordinatorClient = origCoordinator
		executeNCMQueueRebuild = origRebuild
	})

	newSMAPISearcher = func(ctx context.Context, flags *rootFlags, serviceName string) (smapiSearcher, sonos.MusicServiceDescriptor, *sonos.Client, error) {
		return &fakeNCMSearcher{result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{
					ID:       "SONG:1",
					ItemType: "track",
					Title:    "友情岁月",
					TrackMetadata: sonos.SMAPITrackMetadata{
						Artist:  "郑伊健",
						Album:   "古惑仔II之猛龙过江",
						CanPlay: true,
					},
				},
				{
					ID:       "SONG:2",
					ItemType: "track",
					Title:    "心照",
					TrackMetadata: sonos.SMAPITrackMetadata{
						Artist:  "郑伊健",
						Album:   "Discover",
						CanPlay: true,
					},
				},
			},
		}}, sonos.MusicServiceDescriptor{ID: "165", Name: "网易云音乐"}, &sonos.Client{IP: "10.0.0.31"}, nil
	}

	newNCMCoordinatorClient = func(ctx context.Context, flags *rootFlags) (queuePlaybackClient, error) {
		return &fakeQueuePlaybackClient{}, nil
	}
	executeNCMQueueRebuild = func(ctx context.Context, c queuePlaybackClient, items []smapiLikeItem, selectedIndex int) (queuePlaybackResult, error) {
		return queuePlaybackResult{
			Source:         "netease.ncm",
			EnqueuedCount:  len(items),
			PlayedPosition: selectedIndex + 1,
			FinalState:     "PLAYING",
		}, nil
	}

	out, err := execute(t, cmd, "--limit", "5", "--index", "1", "郑伊健")
	if err != nil {
		t.Fatalf("ncm play: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"ncm.play\"") || !strings.Contains(out, "\"capability\": \"music.netease\"") {
		t.Fatalf("missing execution envelope: %q", out)
	}
	if !strings.Contains(out, "\"selectedTitle\": \"友情岁月\"") && !strings.Contains(out, "\"title\": \"友情岁月\"") {
		t.Fatalf("unexpected selected output: %q", out)
	}
}

func TestNCMLuckyJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Format: formatJSON, Name: "客厅"}
	cmd := newNCMLuckyCmd(flags)

	origSearcher := newSMAPISearcher
	origCoordinator := newNCMCoordinatorClient
	origRebuild := executeNCMQueueRebuild
	t.Cleanup(func() {
		newSMAPISearcher = origSearcher
		newNCMCoordinatorClient = origCoordinator
		executeNCMQueueRebuild = origRebuild
	})

	newSMAPISearcher = func(ctx context.Context, flags *rootFlags, serviceName string) (smapiSearcher, sonos.MusicServiceDescriptor, *sonos.Client, error) {
		return &fakeNCMSearcher{result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{
					ID:       "SONG:1",
					ItemType: "track",
					Title:    "谢谢你的爱",
					TrackMetadata: sonos.SMAPITrackMetadata{
						Artist:  "刘德华",
						Album:   "劲歌精选",
						CanPlay: true,
					},
				},
			},
		}}, sonos.MusicServiceDescriptor{ID: "165", Name: "网易云音乐"}, &sonos.Client{IP: "10.0.0.31"}, nil
	}

	newNCMCoordinatorClient = func(ctx context.Context, flags *rootFlags) (queuePlaybackClient, error) {
		return &fakeQueuePlaybackClient{}, nil
	}
	executeNCMQueueRebuild = func(ctx context.Context, c queuePlaybackClient, items []smapiLikeItem, selectedIndex int) (queuePlaybackResult, error) {
		return queuePlaybackResult{
			Source:         "netease.ncm",
			EnqueuedCount:  len(items),
			PlayedPosition: 1,
			FinalState:     "PLAYING",
		}, nil
	}

	out, err := execute(t, cmd, "--limit", "5", "刘德华")
	if err != nil {
		t.Fatalf("ncm lucky: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"ncm.lucky\"") || !strings.Contains(out, "\"capability\": \"music.netease\"") {
		t.Fatalf("missing execution envelope: %q", out)
	}
}
