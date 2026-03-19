package cli

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var executeNCMQueueRebuild = rebuildNCMQueueNative
var newNCMCoordinatorClient = func(ctx context.Context, flags *rootFlags) (queuePlaybackClient, error) {
	return coordinatorClient(ctx, flags)
}

func newNCMPlayCmd(flags *rootFlags) *cobra.Command {
	var (
		category string
		limit    int
		index    int
	)
	cmd := &cobra.Command{
		Use:          "play <query>",
		Short:        "搜索网易云并播放结果",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			query := normalizeNCMQuery(strings.TrimSpace(strings.Join(args, " ")))
			if query == "" {
				return newQueryRequiredError(map[string]any{
					"source": "netease.ncm",
				})
			}
			if strings.TrimSpace(category) == "" {
				category = "tracks"
			}
			if limit <= 0 {
				limit = 10
			}
			if index <= 0 {
				index = 1
			}

			ctx := cmd.Context()
			sm, svc, speaker, err := newSMAPISearcher(ctx, flags, neteaseServiceName)
			if err != nil {
				return err
			}
			res, err := sm.Search(ctx, category, query, 0, limit)
			if err != nil {
				return err
			}
			items := onlyNCMTracks(append(toLikeItemsFromMetadata(res.MediaMetadata), toLikeItemsFromCollections(res.MediaCollection)...))
			blockedFiltered := 0
			_, playability, playabilityErr := loadPlayabilityFile()
			if playabilityErr != nil {
				slog.Warn("playability store unavailable", "error", playabilityErr)
			} else {
				items, blockedFiltered = filterBlockedLikeItems(playability, svc.ID, svc.Name, items)
			}
			items = rankLikeItemsForQuery(query, items)
			if len(items) == 0 {
				message := "no playable tracks"
				if blockedFiltered > 0 {
					message = "no playable tracks after blocked items filtered"
				}
				return newNoResultsError(message, map[string]any{
					"source":          "netease.ncm",
					"query":           query,
					"category":        category,
					"blockedFiltered": blockedFiltered,
				})
			}
			if index > len(items) {
				return newPlayableIndexOutOfRangeError(index, len(items), map[string]any{
					"source":   "netease.ncm",
					"query":    query,
					"category": category,
				})
			}
			selected := items[index-1]
			qc, err := newNCMCoordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			queuedItems := dedupeNCMTracks(items)
			playIndex := findLikeItemIndexByID(queuedItems, selected.ID)
			if playIndex < 0 {
				return newStateInconsistentError(fmt.Sprintf("selected track %q missing after dedupe", selected.ID), map[string]any{
					"source":     "netease.ncm",
					"query":      query,
					"selectedID": selected.ID,
				})
			}
			playback, err := executeNCMQueueRebuild(ctx, qc, queuedItems, playIndex)
			if err != nil {
				rememberBlockedLikeItem(svc.ID, svc.Name, selected, err)
				return err
			}
			playURI := buildNCMTrackURI(selected.ID)
			if isJSON(flags) {
				target := executionTargetFromFlags(flags)
				target["speakerIP"] = speaker.IP
				return writeExecutionOK(cmd, flags, "ncm.play", newExecutionOutput("music.netease", "play", target, map[string]any{
					"query":    query,
					"category": category,
					"limit":    limit,
					"index":    index,
				}, map[string]any{
					"selectedID":      selected.ID,
					"selectedTitle":   selected.Title,
					"playableHits":    len(items),
					"queuedTracks":    playback.EnqueuedCount,
					"playedPosition":  playback.PlayedPosition,
					"blockedFiltered": blockedFiltered,
				}), map[string]any{
					"service":         svc.Name,
					"speakerIP":       speaker.IP,
					"category":        category,
					"query":           query,
					"selected":        selected,
					"uri":             playURI,
					"playableHits":    len(items),
					"queuedTracks":    playback.EnqueuedCount,
					"blockedFiltered": blockedFiltered,
					"playback":        playback,
				})
			}
			writePlainLine(cmd, flags, fmt.Sprintf("已播放：%s", selected.Title))
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "tracks", "搜索分类，默认 tracks")
	cmd.Flags().IntVar(&limit, "limit", 10, "最大搜索结果数")
	cmd.Flags().IntVar(&index, "index", 1, "选择第几个可播放结果（1-based）")
	return cmd
}

func newNCMLuckyCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:          "lucky <query>",
		Short:        "搜索网易云并随机播放一首匹配歌曲",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			query := normalizeNCMQuery(strings.TrimSpace(strings.Join(args, " ")))
			if query == "" {
				return newQueryRequiredError(map[string]any{
					"source": "netease.ncm",
				})
			}
			if limit <= 0 {
				limit = 20
			}
			ctx := cmd.Context()
			sm, svc, speaker, err := newSMAPISearcher(ctx, flags, neteaseServiceName)
			if err != nil {
				return err
			}
			res, err := sm.Search(ctx, "tracks", query, 0, limit)
			if err != nil {
				return err
			}
			items := onlyNCMTracks(toLikeItemsFromMetadata(res.MediaMetadata))
			blockedFiltered := 0
			_, playability, playabilityErr := loadPlayabilityFile()
			if playabilityErr != nil {
				slog.Warn("playability store unavailable", "error", playabilityErr)
			} else {
				items, blockedFiltered = filterBlockedLikeItems(playability, svc.ID, svc.Name, items)
			}
			items = onlyPreferTopRankedNonLiveLikeItems(query, rankLikeItemsForQuery(query, items))
			if len(items) == 0 {
				message := "no playable tracks"
				if blockedFiltered > 0 {
					message = "no playable tracks after blocked items filtered"
				}
				return newNoResultsError(message, map[string]any{
					"source":          "netease.ncm",
					"query":           query,
					"category":        "tracks",
					"blockedFiltered": blockedFiltered,
				})
			}
			selectedIndex := rand.Intn(len(items))
			selected := items[selectedIndex]
			queuedItems := dedupeNCMTracks(items)
			c, err := newNCMCoordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			playIndex := findLikeItemIndexByID(queuedItems, selected.ID)
			if playIndex < 0 {
				return newStateInconsistentError(fmt.Sprintf("selected track %q missing after dedupe", selected.ID), map[string]any{
					"source":     "netease.ncm",
					"query":      query,
					"selectedID": selected.ID,
				})
			}
			playback, err := executeNCMQueueRebuild(ctx, c, queuedItems, playIndex)
			if err != nil {
				rememberBlockedLikeItem(svc.ID, svc.Name, selected, err)
				return err
			}
			playURI := buildNCMTrackURI(selected.ID)
			if isJSON(flags) {
				target := executionTargetFromFlags(flags)
				target["speakerIP"] = speaker.IP
				return writeExecutionOK(cmd, flags, "ncm.lucky", newExecutionOutput("music.netease", "lucky", target, map[string]any{
					"query": query,
					"limit": limit,
				}, map[string]any{
					"selectedID":      selected.ID,
					"selectedTitle":   selected.Title,
					"selectedIndex":   selectedIndex,
					"playableHits":    len(items),
					"queuedTracks":    playback.EnqueuedCount,
					"playedPosition":  playback.PlayedPosition,
					"blockedFiltered": blockedFiltered,
				}), map[string]any{
					"service":         svc.Name,
					"speakerIP":       speaker.IP,
					"query":           query,
					"selected":        selected,
					"uri":             playURI,
					"playableHits":    len(items),
					"queuedTracks":    playback.EnqueuedCount,
					"blockedFiltered": blockedFiltered,
					"playback":        playback,
				})
			}
			writePlainLine(cmd, flags, fmt.Sprintf("已随机播放：%s", selected.Title))
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "最大搜索结果数")
	return cmd
}

type smapiLikeItem struct {
	ID          string `json:"id"`
	ItemType    string `json:"itemType"`
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtURI string `json:"albumArtURI,omitempty"`
	DurationSec int    `json:"durationSec,omitempty"`
	CanPlay     bool   `json:"canPlay,omitempty"`
	CanSkip     bool   `json:"canSkip,omitempty"`
}

func toLikeItemsFromMetadata(items []sonos.SMAPIItem) []smapiLikeItem {
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		out = append(out, smapiLikeItem{
			ID:          it.ID,
			ItemType:    it.ItemType,
			Title:       it.Title,
			Summary:     it.Summary,
			Artist:      it.TrackMetadata.Artist,
			Album:       it.TrackMetadata.Album,
			AlbumArtURI: it.TrackMetadata.AlbumArtURI,
			DurationSec: it.TrackMetadata.DurationSec,
			CanPlay:     it.TrackMetadata.CanPlay,
			CanSkip:     it.TrackMetadata.CanSkip,
		})
	}
	return out
}

func toLikeItemsFromCollections(items []sonos.SMAPIItem) []smapiLikeItem {
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		out = append(out, smapiLikeItem{
			ID:          it.ID,
			ItemType:    it.ItemType,
			Title:       it.Title,
			Summary:     it.Summary,
			Artist:      it.TrackMetadata.Artist,
			Album:       it.TrackMetadata.Album,
			AlbumArtURI: it.TrackMetadata.AlbumArtURI,
			DurationSec: it.TrackMetadata.DurationSec,
			CanPlay:     it.TrackMetadata.CanPlay,
			CanSkip:     it.TrackMetadata.CanSkip,
		})
	}
	return out
}

func onlyNCMTracks(items []smapiLikeItem) []smapiLikeItem {
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		id := strings.TrimSpace(it.ID)
		if strings.HasPrefix(strings.ToUpper(id), "SONG:") {
			out = append(out, it)
		}
	}
	return out
}

func buildNCMTrackURI(id string) string {
	id = strings.TrimSpace(id)
	return "x-sonos-http:" + url.QueryEscape(id) + ".mp3?sid=165&flags=8232&sn=5"
}

func dedupeNCMTracks(items []smapiLikeItem) []smapiLikeItem {
	seen := make(map[string]struct{}, len(items))
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			continue
		}
		key := strings.ToUpper(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}
	return out
}

func findLikeItemIndexByID(items []smapiLikeItem, id string) int {
	key := strings.ToUpper(strings.TrimSpace(id))
	if key == "" {
		return -1
	}
	for idx, it := range items {
		if strings.ToUpper(strings.TrimSpace(it.ID)) == key {
			return idx
		}
	}
	return -1
}

func rebuildNCMQueueNative(ctx context.Context, c queuePlaybackClient, items []smapiLikeItem, selectedIndex int) (queuePlaybackResult, error) {
	queueItems := make([]queuePlaybackItem, 0, len(items))
	for _, it := range items {
		queueItems = append(queueItems, queuePlaybackItem{
			ID:       it.ID,
			Title:    it.Title,
			URI:      buildNCMTrackURI(it.ID),
			Metadata: buildNCMQueueTrackMeta(it),
		})
	}
	return executeQueuePlayback(ctx, c, queueItems, queuePlaybackOptions{
		Source:        "netease.ncm",
		ClearQueue:    true,
		PlayNow:       true,
		SelectedIndex: selectedIndex,
	})
}

func buildNCMQueueTrackMeta(it smapiLikeItem) string {
	rawTitle := strings.TrimSpace(it.Title)
	if rawTitle == "" {
		rawTitle = strings.TrimSpace(it.ID)
	}
	artist := strings.TrimSpace(it.Artist)
	album := strings.TrimSpace(it.Album)
	uri := buildNCMTrackURI(it.ID)
	encodedURI := strings.ToLower(url.QueryEscape(uri))
	displayTitle := compactQueueField(rawTitle)
	displayArtist := compactQueueField(artist)
	displayAlbum := compactQueueField(album)
	itemID := "SQ:0/" + encodedURI + ":A" + displayTitle + "," + displayArtist + "," + displayAlbum + ",0000000197,0"
	return `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="` + xmlEscapeText(itemID) + `" parentID="SQ:0" restricted="true"><dc:title>` + xmlEscapeText(displayTitle) + `</dc:title><upnp:class>object.item.audioItem.musicTrack</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">RINCON_AssociatedZPUDN</desc></item></DIDL-Lite>`
}

func xmlEscapeText(s string) string {
	repl := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return repl.Replace(s)
}

func compactQueueField(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	s = strings.Map(func(r rune) rune {
		switch r {
		case ',', '/', '\n', '\r', '\t', ':', ';', '+', '&', '(', ')', '[', ']', '{', '}', '#', '"', '\'', '\\', '|':
			return ' '
		default:
			return r
		}
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "-"
	}
	if len([]rune(s)) > 32 {
		r := []rune(s)
		s = string(r[:32])
	}
	return s
}

func normalizeNCMQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}
	aliases := map[string]string{
		"sammi":    "郑秀文",
		"eason":    "陈奕迅",
		"jay":      "周杰伦",
		"gem":      "邓紫棋",
		"g.e.m.":   "邓紫棋",
		"gem邓紫棋":   "邓紫棋",
		"joey":     "容祖儿",
		"hins":     "张敬轩",
		"jj":       "林俊杰",
		"andy lau": "刘德华",
		"leehom":   "王力宏",
		"kay":      "谢安琪",
		"hacken":   "许志安",
	}
	lower := strings.ToLower(query)
	if v, ok := aliases[lower]; ok {
		return v
	}
	return query
}
