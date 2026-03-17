package cli

import (
	"context"
	"errors"
	"fmt"
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
				return errors.New("query is required")
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
			if len(items) == 0 {
				return errors.New("no playable tracks")
			}
			if index > len(items) {
				return fmt.Errorf("--index %d out of range (got %d playable tracks)", index, len(items))
			}
			selected := items[index-1]
			qc, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			queuedItems := dedupeNCMTracks(items)
			queued, err := rebuildNCMQueueNative(ctx, qc, queuedItems, index-1)
			if err != nil {
				return err
			}
			playURI := buildNCMTrackURI(selected.ID)
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{
					"service":      svc.Name,
					"speakerIP":    speaker.IP,
					"category":     category,
					"query":        query,
					"selected":     selected,
					"uri":          playURI,
					"playableHits": len(items),
					"queuedTracks": len(queued),
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
				return errors.New("query is required")
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
			if len(items) == 0 {
				return errors.New("no playable tracks")
			}
			selectedIndex := rand.Intn(len(items))
			selected := items[selectedIndex]
			queuedItems := dedupeNCMTracks(items)
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			queued, err := rebuildNCMQueueNative(ctx, c, queuedItems, selectedIndex)
			if err != nil {
				return err
			}
			playURI := buildNCMTrackURI(selected.ID)
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{
					"service":      svc.Name,
					"speakerIP":    speaker.IP,
					"query":        query,
					"selected":     selected,
					"uri":          playURI,
					"playableHits": len(items),
					"queuedTracks": len(queued),
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
	ID            string `json:"id"`
	ItemType      string `json:"itemType"`
	Title         string `json:"title"`
	Summary       string `json:"summary,omitempty"`
	Artist        string `json:"artist,omitempty"`
	Album         string `json:"album,omitempty"`
	AlbumArtURI   string `json:"albumArtURI,omitempty"`
	DurationSec   int    `json:"durationSec,omitempty"`
	CanPlay       bool   `json:"canPlay,omitempty"`
	CanSkip       bool   `json:"canSkip,omitempty"`
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

func rebuildNCMQueueNative(ctx context.Context, c *sonos.Client, items []smapiLikeItem, selectedIndex int) ([]int, error) {
	if len(items) == 0 {
		return nil, errors.New("no queueable tracks")
	}
	if selectedIndex < 0 || selectedIndex >= len(items) {
		return nil, fmt.Errorf("selected index %d out of range for %d tracks", selectedIndex, len(items))
	}
	if err := c.ClearQueue(ctx); err != nil {
		return nil, err
	}

	queued := make([]int, 0, len(items))
	for _, it := range items {
		uri := buildNCMTrackURI(it.ID)
		meta := buildNCMQueueTrackMeta(it)
		pos, err := c.AddURIToQueue(ctx, uri, meta, 0, false)
		if err != nil {
			return queued, fmt.Errorf("enqueue %q failed: %w", it.Title, err)
		}
		queued = append(queued, pos)
	}

	playPos := queued[selectedIndex]
	if playPos <= 0 {
		playPos = selectedIndex + 1
	}
	if err := c.PlayQueuePosition(ctx, playPos); err != nil {
		return queued, err
	}
	_ = c.Play(ctx)
	if ti, err := c.GetTransportInfo(ctx); err == nil && strings.EqualFold(strings.TrimSpace(ti.State), "TRANSITIONING") {
		time.Sleep(1200 * time.Millisecond)
		_ = c.Play(ctx)
	}
	return queued, nil
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
