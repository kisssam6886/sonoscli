package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakeFavoritesClient struct {
	page sonos.FavoritesPage

	playCalls int
	lastItem  sonos.DIDLItem
}

func (f *fakeFavoritesClient) ListFavorites(ctx context.Context, start, count int) (sonos.FavoritesPage, error) {
	if count == 1 {
		if start >= 0 && start < len(f.page.Items) {
			return sonos.FavoritesPage{
				Items:          []sonos.FavoriteItem{f.page.Items[start]},
				NumberReturned: 1,
				TotalMatches:   len(f.page.Items),
			}, nil
		}
		return sonos.FavoritesPage{Items: nil, NumberReturned: 0, TotalMatches: len(f.page.Items)}, nil
	}
	return f.page, nil
}

func (f *fakeFavoritesClient) PlayFavorite(ctx context.Context, favorite sonos.DIDLItem) error {
	f.playCalls++
	f.lastItem = favorite
	return nil
}

func TestFavoritesListJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newFavoritesListCmd(flags)

	fake := &fakeFavoritesClient{
		page: sonos.FavoritesPage{
			Items: []sonos.FavoriteItem{
				{Position: 1, Item: sonos.DIDLItem{Title: "Fav 1", URI: "x://1"}},
			},
			NumberReturned: 1,
			TotalMatches:   1,
		},
	}

	orig := newFavoritesClient
	t.Cleanup(func() { newFavoritesClient = orig })
	newFavoritesClient = func(ctx context.Context, flags *rootFlags) (favoritesClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"favorites.list\"") || !strings.Contains(out, "\"capability\": \"favorites\"") || !strings.Contains(out, "\"operation\": \"list\"") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "\"title\": \"Fav 1\"") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestFavoritesOpenByIndex(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second}
	cmd := newFavoritesOpenCmd(flags)

	fake := &fakeFavoritesClient{
		page: sonos.FavoritesPage{
			Items: []sonos.FavoriteItem{
				{Position: 1, Item: sonos.DIDLItem{Title: "Fav 1", URI: "x://1"}},
				{Position: 2, Item: sonos.DIDLItem{Title: "Fav 2", URI: "x://2"}},
			},
			NumberReturned: 2,
			TotalMatches:   2,
		},
	}

	orig := newFavoritesClient
	t.Cleanup(func() { newFavoritesClient = orig })
	newFavoritesClient = func(ctx context.Context, flags *rootFlags) (favoritesClient, error) {
		return fake, nil
	}

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SetArgs([]string{"--index", "2"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.playCalls != 1 {
		t.Fatalf("expected playCalls=1, got %d", fake.playCalls)
	}
	if fake.lastItem.Title != "Fav 2" {
		t.Fatalf("expected Fav 2, got %q", fake.lastItem.Title)
	}
}

func TestFavoritesOpenByIndexJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newFavoritesOpenCmd(flags)

	fake := &fakeFavoritesClient{
		page: sonos.FavoritesPage{
			Items: []sonos.FavoriteItem{
				{Position: 1, Item: sonos.DIDLItem{Title: "Fav 1", URI: "x://1"}},
				{Position: 2, Item: sonos.DIDLItem{Title: "Fav 2", URI: "x://2"}},
			},
			NumberReturned: 2,
			TotalMatches:   2,
		},
	}

	orig := newFavoritesClient
	t.Cleanup(func() { newFavoritesClient = orig })
	newFavoritesClient = func(ctx context.Context, flags *rootFlags) (favoritesClient, error) {
		return fake, nil
	}

	out, err := execute(t, cmd, "--index", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"favorites.open\"") || !strings.Contains(out, "\"capability\": \"favorites\"") || !strings.Contains(out, "\"operation\": \"open\"") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "\"title\": \"Fav 2\"") {
		t.Fatalf("unexpected output: %s", out)
	}
	if fake.playCalls != 1 {
		t.Fatalf("expected playCalls=1, got %d", fake.playCalls)
	}
}

func TestFavoritesOpenRequiresIndexOrTitle(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second}
	cmd := newFavoritesOpenCmd(flags)

	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if code := errorCode(err); code != errCodeInvalidArgument {
		t.Fatalf("code = %q, want %q", code, errCodeInvalidArgument)
	}
}

func TestFavoritesOpenIndexOutOfRange(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second}
	cmd := newFavoritesOpenCmd(flags)

	fake := &fakeFavoritesClient{
		page: sonos.FavoritesPage{
			Items: []sonos.FavoriteItem{
				{Position: 1, Item: sonos.DIDLItem{Title: "Fav 1", URI: "x://1"}},
			},
			NumberReturned: 1,
			TotalMatches:   1,
		},
	}

	orig := newFavoritesClient
	t.Cleanup(func() { newFavoritesClient = orig })
	newFavoritesClient = func(ctx context.Context, flags *rootFlags) (favoritesClient, error) {
		return fake, nil
	}

	cmd.SetArgs([]string{"--index", "2"})
	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if code := errorCode(err); code != errCodeIndexOutOfRange {
		t.Fatalf("code = %q, want %q", code, errCodeIndexOutOfRange)
	}
}

func TestFavoritesOpenTitleNotFound(t *testing.T) {
	flags := &rootFlags{Name: "Kitchen", Timeout: 2 * time.Second}
	cmd := newFavoritesOpenCmd(flags)

	fake := &fakeFavoritesClient{
		page: sonos.FavoritesPage{
			Items: []sonos.FavoriteItem{
				{Position: 1, Item: sonos.DIDLItem{Title: "Fav 1", URI: "x://1"}},
			},
			NumberReturned: 1,
			TotalMatches:   1,
		},
	}

	orig := newFavoritesClient
	t.Cleanup(func() { newFavoritesClient = orig })
	newFavoritesClient = func(ctx context.Context, flags *rootFlags) (favoritesClient, error) {
		return fake, nil
	}

	cmd.SetArgs([]string{"Missing"})
	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if code := errorCode(err); code != errCodeNotFound {
		t.Fatalf("code = %q, want %q", code, errCodeNotFound)
	}
}
