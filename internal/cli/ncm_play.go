package cli

import (
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
			playURI := buildNCMTrackURI(selected.ID)
			c, err := newSourceClient(ctx, flags)
			if err != nil {
				return err
			}
			if err := c.SetAVTransportURI(ctx, playURI, ""); err != nil {
				return err
			}
			if err := c.Play(ctx); err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{
					"service":      svc.Name,
					"speakerIP":    speaker.IP,
					"category":     category,
					"query":        query,
					"selected":     selected,
					"uri":          playURI,
					"playableHits": len(items),
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
			selected := items[rand.Intn(len(items))]
			playURI := buildNCMTrackURI(selected.ID)
			c, err := newSourceClient(ctx, flags)
			if err != nil {
				return err
			}
			if err := c.SetAVTransportURI(ctx, playURI, ""); err != nil {
				return err
			}
			if err := c.Play(ctx); err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{
					"service":      svc.Name,
					"speakerIP":    speaker.IP,
					"query":        query,
					"selected":     selected,
					"uri":          playURI,
					"playableHits": len(items),
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
	ID       string `json:"id"`
	ItemType string `json:"itemType"`
	Title    string `json:"title"`
	Summary  string `json:"summary,omitempty"`
}

func toLikeItemsFromMetadata(items []sonos.SMAPIItem) []smapiLikeItem {
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		out = append(out, smapiLikeItem{ID: it.ID, ItemType: it.ItemType, Title: it.Title, Summary: it.Summary})
	}
	return out
}

func toLikeItemsFromCollections(items []sonos.SMAPIItem) []smapiLikeItem {
	out := make([]smapiLikeItem, 0, len(items))
	for _, it := range items {
		out = append(out, smapiLikeItem{ID: it.ID, ItemType: it.ItemType, Title: it.Title, Summary: it.Summary})
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
