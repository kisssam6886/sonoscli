package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/sonoscli/internal/sonos"
)

func TestRememberBlockedLikeItem_PersistsTransitionStuck(t *testing.T) {
	t.Parallel()

	store, err := newFileBackedPlayabilityStore(filepath.Join(t.TempDir(), "playability.json"))
	if err != nil {
		t.Fatalf("newFileBackedPlayabilityStore: %v", err)
	}

	origStore := newPlayabilityStore
	t.Cleanup(func() {
		newPlayabilityStore = origStore
	})
	newPlayabilityStore = func() (playabilityStore, error) {
		return store, nil
	}

	rememberBlockedLikeItem("165", "网易云音乐", smapiLikeItem{
		ID:     "song:bad",
		Title:  "甘心替代你",
		Artist: "郑伊健",
		Album:  "古惑仔最强精选集",
	}, newCodedError(errCodeTransitionStuck, "stuck", nil, nil))

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(got.Entries))
	}
	entry := got.Entries[0]
	if entry.ServiceID != "165" {
		t.Fatalf("serviceID = %q, want 165", entry.ServiceID)
	}
	if entry.ItemID != "SONG:BAD" {
		t.Fatalf("itemID = %q, want SONG:BAD", entry.ItemID)
	}
	if entry.Status != playabilityStatusBlocked {
		t.Fatalf("status = %q, want %q", entry.Status, playabilityStatusBlocked)
	}
	if entry.Reason != playabilityReasonTransitionStuck {
		t.Fatalf("reason = %q, want %q", entry.Reason, playabilityReasonTransitionStuck)
	}
}

func TestNCMPlaySkipsBlockedTracks(t *testing.T) {
	flags := &rootFlags{Format: formatJSON, Name: "客厅"}
	cmd := newNCMPlayCmd(flags)

	store, err := newFileBackedPlayabilityStore(filepath.Join(t.TempDir(), "playability.json"))
	if err != nil {
		t.Fatalf("newFileBackedPlayabilityStore: %v", err)
	}
	if err := store.Save(playabilityFile{
		Version: 1,
		Entries: []playabilityEntry{{
			ServiceID:   "165",
			ServiceName: "网易云音乐",
			ItemID:      "SONG:1",
			Title:       "坏版本",
			Status:      playabilityStatusBlocked,
			Reason:      playabilityReasonTransitionStuck,
		}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	origSearcher := newSMAPISearcher
	origCoordinator := newNCMCoordinatorClient
	origRebuild := executeNCMQueueRebuild
	origStore := newPlayabilityStore
	t.Cleanup(func() {
		newSMAPISearcher = origSearcher
		newNCMCoordinatorClient = origCoordinator
		executeNCMQueueRebuild = origRebuild
		newPlayabilityStore = origStore
	})

	newPlayabilityStore = func() (playabilityStore, error) {
		return store, nil
	}
	newSMAPISearcher = func(ctx context.Context, flags *rootFlags, serviceName string) (smapiSearcher, sonos.MusicServiceDescriptor, *sonos.Client, error) {
		return &fakeNCMSearcher{result: sonos.SMAPISearchResult{
			MediaMetadata: []sonos.SMAPIItem{
				{
					ID:       "SONG:1",
					ItemType: "track",
					Title:    "坏版本",
					TrackMetadata: sonos.SMAPITrackMetadata{
						Artist:  "郑伊健",
						Album:   "黑名单专辑",
						CanPlay: true,
					},
				},
				{
					ID:       "SONG:2",
					ItemType: "track",
					Title:    "好版本",
					TrackMetadata: sonos.SMAPITrackMetadata{
						Artist:  "郑伊健",
						Album:   "白名单专辑",
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
		if len(items) != 1 {
			t.Fatalf("items len = %d, want 1 after blocked filtering", len(items))
		}
		if items[0].ID != "SONG:2" {
			t.Fatalf("selected queue item = %q, want SONG:2", items[0].ID)
		}
		if selectedIndex != 0 {
			t.Fatalf("selectedIndex = %d, want 0", selectedIndex)
		}
		return queuePlaybackResult{
			Source:         "netease.ncm",
			EnqueuedCount:  len(items),
			PlayedPosition: 1,
			FinalState:     "PLAYING",
		}, nil
	}

	out, err := execute(t, cmd, "--limit", "5", "--index", "1", "郑伊健")
	if err != nil {
		t.Fatalf("ncm play: %v", err)
	}
	if !strings.Contains(out, "\"selectedTitle\": \"好版本\"") && !strings.Contains(out, "\"title\": \"好版本\"") {
		t.Fatalf("expected filtered playable title in output, got %q", out)
	}
	if !strings.Contains(out, "\"blockedFiltered\": 1") {
		t.Fatalf("expected blockedFiltered=1 in output, got %q", out)
	}
}

func TestPreferNonLiveLikeItems_PutsLiveAfterStudio(t *testing.T) {
	t.Parallel()

	items := []smapiLikeItem{
		{ID: "SONG:1", Title: "男人哭吧不是罪(Live)", Album: "幻影中国巡回演唱会Live"},
		{ID: "SONG:2", Title: "男人哭吧不是罪", Album: "经典精选"},
		{ID: "SONG:3", Title: "心照", Album: "Friends For Life (新曲+精选)"},
	}

	got := preferNonLiveLikeItems(items)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].ID != "SONG:2" || got[1].ID != "SONG:3" || got[2].ID != "SONG:1" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
