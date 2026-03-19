package cli

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

const (
	playabilityStatusBlocked         = "blocked"
	playabilityStatusWeak            = "weak"
	playabilityStatusStable          = "stable"
	playabilityReasonTransitionStuck = "transition_stuck"
	defaultPlayabilityStoreVersion   = 1
)

type playabilityStore interface {
	Path() string
	Load() (playabilityFile, error)
	Save(playabilityFile) error
}

type playabilityFileStore struct {
	path string
}

type playabilityFile struct {
	Version int                `json:"version"`
	Entries []playabilityEntry `json:"entries,omitempty"`
}

type playabilityEntry struct {
	ServiceID   string `json:"serviceID,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	ItemID      string `json:"itemID"`
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

var newPlayabilityStore = func() (playabilityStore, error) {
	return newDefaultPlayabilityStore()
}

func newFileBackedPlayabilityStore(path string) (*playabilityFileStore, error) {
	return &playabilityFileStore{path: path}, nil
}

func newDefaultPlayabilityStore() (*playabilityFileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &playabilityFileStore{path: filepath.Join(dir, "sonoscli", "playability.json")}, nil
}

func (s *playabilityFileStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *playabilityFileStore) Load() (playabilityFile, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return playabilityFile{Version: defaultPlayabilityStoreVersion}, nil
		}
		return playabilityFile{}, err
	}
	var f playabilityFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return playabilityFile{}, err
	}
	return normalizePlayabilityFile(f), nil
}

func (s *playabilityFileStore) Save(f playabilityFile) error {
	f = normalizePlayabilityFile(f)
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func normalizePlayabilityFile(f playabilityFile) playabilityFile {
	if f.Version == 0 {
		f.Version = defaultPlayabilityStoreVersion
	}
	out := playabilityFile{
		Version: f.Version,
		Entries: make([]playabilityEntry, 0, len(f.Entries)),
	}
	seen := make(map[string]int, len(f.Entries))
	for _, entry := range f.Entries {
		entry, ok := normalizePlayabilityEntry(entry)
		if !ok {
			continue
		}
		key := playabilityEntryKey(entry.ServiceID, entry.ServiceName, entry.ItemID)
		if idx, exists := seen[key]; exists {
			out.Entries[idx] = entry
			continue
		}
		seen[key] = len(out.Entries)
		out.Entries = append(out.Entries, entry)
	}
	return out
}

func normalizePlayabilityEntry(entry playabilityEntry) (playabilityEntry, bool) {
	entry.ServiceID = strings.TrimSpace(entry.ServiceID)
	entry.ServiceName = strings.TrimSpace(entry.ServiceName)
	entry.ItemID = normalizePlayabilityItemID(entry.ItemID)
	entry.Title = strings.TrimSpace(entry.Title)
	entry.Artist = strings.TrimSpace(entry.Artist)
	entry.Album = strings.TrimSpace(entry.Album)
	entry.Status = normalizePlayabilityStatus(entry.Status)
	entry.Reason = strings.TrimSpace(entry.Reason)
	entry.UpdatedAt = strings.TrimSpace(entry.UpdatedAt)
	if entry.ItemID == "" || entry.Status == "" {
		return playabilityEntry{}, false
	}
	if entry.UpdatedAt == "" {
		entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return entry, true
}

func normalizePlayabilityStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "blocked", "block", "blacklist":
		return playabilityStatusBlocked
	case "weak", "weak_playable", "weak-playable":
		return playabilityStatusWeak
	case "stable", "playable", "allow", "whitelist":
		return playabilityStatusStable
	default:
		return ""
	}
}

func normalizePlayabilityItemID(itemID string) string {
	return strings.ToUpper(strings.TrimSpace(itemID))
}

func playabilityEntryKey(serviceID, serviceName, itemID string) string {
	serviceID = strings.TrimSpace(serviceID)
	serviceName = strings.ToLower(strings.TrimSpace(serviceName))
	itemID = normalizePlayabilityItemID(itemID)
	if serviceID != "" {
		return serviceID + "|" + itemID
	}
	return serviceName + "|" + itemID
}

func (f playabilityFile) find(serviceID, serviceName, itemID string) (playabilityEntry, bool) {
	itemID = normalizePlayabilityItemID(itemID)
	serviceID = strings.TrimSpace(serviceID)
	serviceName = strings.TrimSpace(serviceName)
	if itemID == "" {
		return playabilityEntry{}, false
	}
	for _, entry := range f.Entries {
		if entry.ItemID != itemID {
			continue
		}
		if serviceID != "" && entry.ServiceID != "" && entry.ServiceID == serviceID {
			return entry, true
		}
		if serviceID == "" && entry.ServiceID == "" && strings.EqualFold(entry.ServiceName, serviceName) {
			return entry, true
		}
		if serviceID != "" && entry.ServiceID == "" && strings.EqualFold(entry.ServiceName, serviceName) {
			return entry, true
		}
	}
	return playabilityEntry{}, false
}

func (f *playabilityFile) upsert(entry playabilityEntry) {
	if f == nil {
		return
	}
	entry, ok := normalizePlayabilityEntry(entry)
	if !ok {
		return
	}
	key := playabilityEntryKey(entry.ServiceID, entry.ServiceName, entry.ItemID)
	for idx, existing := range f.Entries {
		if playabilityEntryKey(existing.ServiceID, existing.ServiceName, existing.ItemID) == key {
			f.Entries[idx] = entry
			return
		}
	}
	f.Entries = append(f.Entries, entry)
}

func loadPlayabilityFile() (playabilityStore, playabilityFile, error) {
	store, err := newPlayabilityStore()
	if err != nil {
		return nil, playabilityFile{}, err
	}
	f, err := store.Load()
	if err != nil {
		return nil, playabilityFile{}, err
	}
	return store, f, nil
}

func filterBlockedSearchResult(reg playabilityFile, svc sonos.MusicServiceDescriptor, res sonos.SMAPISearchResult) (sonos.SMAPISearchResult, int) {
	filtered := res
	blocked := 0
	filtered.MediaMetadata, blocked = filterBlockedSMAPIItems(reg, svc, res.MediaMetadata)
	filteredCollections, blockedCollections := filterBlockedSMAPIItems(reg, svc, res.MediaCollection)
	filtered.MediaCollection = filteredCollections
	blocked += blockedCollections
	filtered = preferNonLiveSearchResult(filtered)
	filtered.Count = len(filtered.MediaMetadata) + len(filtered.MediaCollection)
	return filtered, blocked
}

func filterBlockedSMAPIItems(reg playabilityFile, svc sonos.MusicServiceDescriptor, items []sonos.SMAPIItem) ([]sonos.SMAPIItem, int) {
	if len(items) == 0 {
		return nil, 0
	}
	out := make([]sonos.SMAPIItem, 0, len(items))
	blocked := 0
	for _, item := range items {
		entry, ok := reg.find(svc.ID, svc.Name, item.ID)
		if ok && entry.Status == playabilityStatusBlocked {
			blocked++
			continue
		}
		out = append(out, item)
	}
	return out, blocked
}

func filterBlockedLikeItems(reg playabilityFile, serviceID, serviceName string, items []smapiLikeItem) ([]smapiLikeItem, int) {
	if len(items) == 0 {
		return nil, 0
	}
	out := make([]smapiLikeItem, 0, len(items))
	blocked := 0
	for _, item := range items {
		entry, ok := reg.find(serviceID, serviceName, item.ID)
		if ok && entry.Status == playabilityStatusBlocked {
			blocked++
			continue
		}
		out = append(out, item)
	}
	return out, blocked
}

func preferNonLiveSearchResult(res sonos.SMAPISearchResult) sonos.SMAPISearchResult {
	res.MediaMetadata = preferNonLiveSMAPIItems(res.MediaMetadata)
	res.MediaCollection = preferNonLiveSMAPIItems(res.MediaCollection)
	return res
}

func preferNonLiveSMAPIItems(items []sonos.SMAPIItem) []sonos.SMAPIItem {
	if len(items) < 2 {
		return items
	}
	out := append([]sonos.SMAPIItem(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		return livePenaltyForSMAPIItem(out[i]) < livePenaltyForSMAPIItem(out[j])
	})
	return out
}

func preferNonLiveLikeItems(items []smapiLikeItem) []smapiLikeItem {
	if len(items) < 2 {
		return items
	}
	out := append([]smapiLikeItem(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		return livePenaltyForLikeItem(out[i]) < livePenaltyForLikeItem(out[j])
	})
	return out
}

func onlyPreferNonLiveLikeItems(items []smapiLikeItem) []smapiLikeItem {
	if len(items) == 0 {
		return nil
	}
	nonLive := make([]smapiLikeItem, 0, len(items))
	for _, item := range items {
		if livePenaltyForLikeItem(item) == 0 {
			nonLive = append(nonLive, item)
		}
	}
	if len(nonLive) == 0 {
		return items
	}
	return nonLive
}

func livePenaltyForSMAPIItem(item sonos.SMAPIItem) int {
	if isLikelyLiveVersion(item.Title, item.Summary, item.TrackMetadata.Album) {
		return 1
	}
	return 0
}

func livePenaltyForLikeItem(item smapiLikeItem) int {
	if isLikelyLiveVersion(item.Title, item.Summary, item.Album) {
		return 1
	}
	return 0
}

func isLikelyLiveVersion(parts ...string) bool {
	if len(parts) == 0 {
		return false
	}
	text := strings.ToLower(strings.Join(parts, " "))
	if text == "" {
		return false
	}
	indicators := []string{
		"live",
		"concert",
		"演唱会",
		"音乐会",
		"巡回",
		"现场",
	}
	for _, indicator := range indicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}
	return false
}

func rememberBlockedSMAPIItem(svc sonos.MusicServiceDescriptor, item sonos.SMAPIItem, err error) {
	rememberBlockedPlayabilityEntry(svc.ID, svc.Name, playabilityEntry{
		ItemID: item.ID,
		Title:  item.Title,
		Artist: item.TrackMetadata.Artist,
		Album:  item.TrackMetadata.Album,
	}, err)
}

func rememberBlockedLikeItem(serviceID, serviceName string, item smapiLikeItem, err error) {
	rememberBlockedPlayabilityEntry(serviceID, serviceName, playabilityEntry{
		ItemID: item.ID,
		Title:  item.Title,
		Artist: item.Artist,
		Album:  item.Album,
	}, err)
}

func rememberBlockedPlayabilityEntry(serviceID, serviceName string, entry playabilityEntry, err error) {
	if errorCode(err) != errCodeTransitionStuck {
		return
	}
	entry.ServiceID = serviceID
	entry.ServiceName = serviceName
	entry.Status = playabilityStatusBlocked
	entry.Reason = playabilityReasonTransitionStuck
	entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	store, file, loadErr := loadPlayabilityFile()
	if loadErr != nil {
		slog.Warn("playability store unavailable",
			"path", playabilityStorePath(store),
			"error", loadErr,
		)
		return
	}
	file.upsert(entry)
	if saveErr := store.Save(file); saveErr != nil {
		slog.Warn("playability store save failed",
			"path", playabilityStorePath(store),
			"error", saveErr,
			"item_id", normalizePlayabilityItemID(entry.ItemID),
			"service_id", strings.TrimSpace(serviceID),
		)
	}
}

func playabilityStorePath(store playabilityStore) string {
	if store == nil {
		return ""
	}
	return store.Path()
}
