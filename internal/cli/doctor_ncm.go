package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

const doctorNCMCandidatePreviewLimit = 5

type doctorNCMReport struct {
	OK          bool                     `json:"ok"`
	Target      doctorNCMTargetInfo      `json:"target"`
	Service     doctorNCMServiceInfo     `json:"service"`
	Playability doctorNCMPlayabilityInfo `json:"playability"`
	Search      *doctorNCMSearchInfo     `json:"search,omitempty"`
	Warnings    []string                 `json:"warnings,omitempty"`
}

type doctorNCMTargetInfo struct {
	Room          string `json:"room,omitempty"`
	SpeakerIP     string `json:"speakerIP,omitempty"`
	CoordinatorIP string `json:"coordinatorIP,omitempty"`
}

type doctorNCMServiceInfo struct {
	Name           string   `json:"name,omitempty"`
	ID             string   `json:"id,omitempty"`
	Auth           string   `json:"auth,omitempty"`
	Found          bool     `json:"found"`
	SpeakerIP      string   `json:"speakerIP,omitempty"`
	HouseholdID    string   `json:"householdID,omitempty"`
	TokenPresent   bool     `json:"tokenPresent"`
	AuthRequired   bool     `json:"authRequired"`
	SearchReady    bool     `json:"searchReady"`
	Categories     []string `json:"categories,omitempty"`
	AvailableCount int      `json:"availableCount"`
	AvailableNames []string `json:"availableNames,omitempty"`
	Error          string   `json:"error,omitempty"`
}

type doctorNCMPlayabilityInfo struct {
	Path           string `json:"path,omitempty"`
	Loaded         bool   `json:"loaded"`
	TotalEntries   int    `json:"totalEntries"`
	ServiceEntries int    `json:"serviceEntries"`
	BlockedEntries int    `json:"blockedEntries"`
	StableEntries  int    `json:"stableEntries"`
	WeakEntries    int    `json:"weakEntries"`
	Error          string `json:"error,omitempty"`
}

type doctorNCMSearchInfo struct {
	Performed       bool                 `json:"performed"`
	Query           string               `json:"query,omitempty"`
	NormalizedQuery string               `json:"normalizedQuery,omitempty"`
	Category        string               `json:"category,omitempty"`
	Limit           int                  `json:"limit"`
	RawTrackCount   int                  `json:"rawTrackCount"`
	CandidateCount  int                  `json:"candidateCount"`
	BlockedFiltered int                  `json:"blockedFiltered"`
	Error           string               `json:"error,omitempty"`
	Recommended     *doctorNCMCandidate  `json:"recommended,omitempty"`
	Candidates      []doctorNCMCandidate `json:"candidates,omitempty"`
}

type doctorNCMCandidate struct {
	Rank              int    `json:"rank"`
	ID                string `json:"id,omitempty"`
	Title             string `json:"title,omitempty"`
	Artist            string `json:"artist,omitempty"`
	Album             string `json:"album,omitempty"`
	CanPlay           bool   `json:"canPlay"`
	DurationSec       int    `json:"durationSec,omitempty"`
	URI               string `json:"uri,omitempty"`
	MatchLevel        string `json:"matchLevel,omitempty"`
	ArtistScore       int    `json:"artistScore"`
	TitleScore        int    `json:"titleScore"`
	AdaptationPenalty int    `json:"adaptationPenalty"`
	LivePenalty       int    `json:"livePenalty"`
}

func newDoctorNCMCmd(flags *rootFlags) *cobra.Command {
	var (
		query    string
		category string
		limit    int
	)
	cmd := &cobra.Command{
		Use:          "ncm",
		Short:        "Inspect NetEase SMAPI readiness, playability state, and candidate quality",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := buildDoctorNCMReport(cmd.Context(), flags, query, category, limit)
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, doctorNCMJSONOutput(flags, report))
			}
			printDoctorNCMPlain(cmd, report)
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Optional exact-match query to inspect candidate quality, e.g. \"郑伊健 甘心替代你\"")
	cmd.Flags().StringVar(&category, "category", "tracks", "SMAPI search category for diagnostics")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max search results to inspect when --query is provided")
	return cmd
}

func buildDoctorNCMReport(ctx context.Context, flags *rootFlags, query, category string, limit int) (doctorNCMReport, error) {
	if err := validateTarget(flags); err != nil {
		return doctorNCMReport{}, err
	}
	query = strings.TrimSpace(query)
	category = strings.TrimSpace(category)
	if category == "" {
		category = "tracks"
	}
	if limit <= 0 {
		limit = 10
	}

	report := doctorNCMReport{
		Target: doctorNCMTargetInfo{
			Room: strings.TrimSpace(flags.Name),
		},
		Service: doctorNCMServiceInfo{
			Name: neteaseServiceName,
		},
	}

	if coordIP, err := resolveTargetCoordinatorIP(ctx, flags); err == nil {
		report.Target.CoordinatorIP = strings.TrimSpace(coordIP)
	}

	speaker, err := anySpeakerClient(ctx, flags)
	if err != nil {
		return doctorNCMReport{}, err
	}
	report.Target.SpeakerIP = strings.TrimSpace(speaker.IP)
	report.Service.SpeakerIP = strings.TrimSpace(speaker.IP)

	services, err := speaker.ListAvailableServices(ctx)
	if err != nil {
		return doctorNCMReport{}, err
	}
	report.Service.AvailableCount = len(services)

	svc, svcErr := findServiceByName(services, neteaseServiceName)
	if svcErr != nil {
		report.Service.Error = svcErr.Error()
		report.Service.AvailableNames = availableServiceNames(services)
		report.Warnings = append(report.Warnings, svcErr.Error())
		report.OK = false
		return report, nil
	}
	report.Service.Found = true
	report.Service.ID = strings.TrimSpace(svc.ID)
	report.Service.Auth = strings.TrimSpace(string(svc.Auth))
	report.Service.AuthRequired = doctorNCMAuthRequired(svc.Auth)

	playability, playabilityErr := buildDoctorNCMPlayabilityInfo(svc)
	report.Playability = playability
	if playabilityErr != nil {
		report.Warnings = append(report.Warnings, playabilityErr.Error())
	}

	store, err := newSMAPITokenStore()
	if err != nil {
		return doctorNCMReport{}, err
	}
	sm, err := sonos.NewSMAPIClient(ctx, speaker, svc, store)
	if err != nil {
		report.Service.Error = err.Error()
		report.Warnings = append(report.Warnings, err.Error())
		report.OK = false
		return report, nil
	}
	report.Service.HouseholdID = strings.TrimSpace(sm.HouseholdID)
	report.Service.TokenPresent = store.Has(svc.ID, sm.HouseholdID)
	if report.Service.AuthRequired && !report.Service.TokenPresent {
		report.Warnings = append(report.Warnings, "网易云音乐需要本地 SMAPI token，但当前 token store 未命中 household")
	}

	categories, categoriesErr := sm.SearchCategories(ctx)
	if categoriesErr != nil {
		report.Service.Error = categoriesErr.Error()
		report.Warnings = append(report.Warnings, categoriesErr.Error())
	} else {
		sort.Strings(categories)
		report.Service.Categories = categories
		report.Service.SearchReady = containsFold(categories, category)
		if !report.Service.SearchReady {
			report.Warnings = append(report.Warnings, fmt.Sprintf("网易云音乐当前未声明搜索分类 %q", category))
		}
	}

	if query != "" && categoriesErr == nil && report.Service.SearchReady {
		search := buildDoctorNCMSearchInfo(ctx, sm, svc, query, category, limit, report.Playability)
		report.Search = &search
		if strings.TrimSpace(search.Error) != "" {
			report.Warnings = append(report.Warnings, search.Error)
		}
		if search.BlockedFiltered > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("本地 playability store 已过滤 %d 条网易云候选", search.BlockedFiltered))
		}
		if search.Recommended == nil {
			report.Warnings = append(report.Warnings, "当前搜索结果里没有可用的网易云候选")
		} else {
			if search.Recommended.ArtistScore <= 0 || search.Recommended.TitleScore <= 0 {
				report.Warnings = append(report.Warnings, "当前推荐候选不是严格的歌手+歌名命中")
			}
			if search.Recommended.AdaptationPenalty > 0 {
				report.Warnings = append(report.Warnings, "当前推荐候选疑似翻唱/Remix/DJ 版本")
			}
			if search.Recommended.LivePenalty > 0 {
				report.Warnings = append(report.Warnings, "当前推荐候选是 Live 版本")
			}
		}
	}

	report.OK = len(report.Warnings) == 0
	return report, nil
}

func buildDoctorNCMSearchInfo(ctx context.Context, sm *sonos.SMAPIClient, svc sonos.MusicServiceDescriptor, query, category string, limit int, playability doctorNCMPlayabilityInfo) doctorNCMSearchInfo {
	info := doctorNCMSearchInfo{
		Performed:       true,
		Query:           strings.TrimSpace(query),
		NormalizedQuery: normalizeNCMQuery(strings.TrimSpace(query)),
		Category:        strings.TrimSpace(category),
		Limit:           limit,
	}
	res, err := sm.Search(ctx, category, info.NormalizedQuery, 0, limit)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	rawItems := onlyNCMTracks(append(toLikeItemsFromMetadata(res.MediaMetadata), toLikeItemsFromCollections(res.MediaCollection)...))
	info.RawTrackCount = len(rawItems)

	filteredItems := rawItems
	if playability.Loaded {
		if _, reg, err := loadPlayabilityFile(); err == nil {
			filteredItems, info.BlockedFiltered = filterBlockedLikeItems(reg, svc.ID, svc.Name, rawItems)
		}
	}
	ranked := rankLikeItemsForQuery(info.NormalizedQuery, filteredItems)
	info.CandidateCount = len(ranked)
	info.Candidates = previewDoctorNCMCandidates(info.NormalizedQuery, ranked, doctorNCMCandidatePreviewLimit)
	if len(info.Candidates) > 0 {
		recommended := info.Candidates[0]
		info.Recommended = &recommended
	}
	return info
}

func previewDoctorNCMCandidates(query string, items []smapiLikeItem, limit int) []doctorNCMCandidate {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]doctorNCMCandidate, 0, limit)
	for idx, item := range items[:limit] {
		out = append(out, doctorNCMCandidate{
			Rank:              idx + 1,
			ID:                strings.TrimSpace(item.ID),
			Title:             strings.TrimSpace(item.Title),
			Artist:            strings.TrimSpace(item.Artist),
			Album:             strings.TrimSpace(item.Album),
			CanPlay:           item.CanPlay,
			DurationSec:       item.DurationSec,
			URI:               buildNCMTrackURI(item.ID),
			MatchLevel:        doctorNCMCandidateMatchLevel(query, item),
			ArtistScore:       artistMatchScoreForQuery(query, item.Artist),
			TitleScore:        titleMatchScoreForQuery(query, item.Title),
			AdaptationPenalty: adaptationPenaltyForParts(item.Title, item.Summary, item.Artist, item.Album),
			LivePenalty:       livePenaltyForLikeItem(item),
		})
	}
	return out
}

func doctorNCMCandidateMatchLevel(query string, item smapiLikeItem) string {
	artistScore := artistMatchScoreForQuery(query, item.Artist)
	titleScore := titleMatchScoreForQuery(query, item.Title)
	adaptation := adaptationPenaltyForParts(item.Title, item.Summary, item.Artist, item.Album)
	live := livePenaltyForLikeItem(item)
	switch {
	case artistScore == 2 && titleScore == 2 && adaptation == 0 && live == 0:
		return "exact_nonlive"
	case artistScore == 2 && titleScore == 2:
		return "exact"
	case artistScore > 0 && titleScore > 0:
		return "artist_title"
	case titleScore > 0:
		return "title_only"
	case artistScore > 0:
		return "artist_only"
	default:
		return "weak"
	}
}

func buildDoctorNCMPlayabilityInfo(svc sonos.MusicServiceDescriptor) (doctorNCMPlayabilityInfo, error) {
	store, reg, err := loadPlayabilityFile()
	if err != nil {
		return doctorNCMPlayabilityInfo{
			Path:   playabilityStorePath(store),
			Loaded: false,
			Error:  err.Error(),
		}, err
	}
	info := doctorNCMPlayabilityInfo{
		Path:         playabilityStorePath(store),
		Loaded:       true,
		TotalEntries: len(reg.Entries),
	}
	for _, entry := range reg.Entries {
		if !doctorNCMEntryMatchesService(entry, svc.ID, svc.Name) {
			continue
		}
		info.ServiceEntries++
		switch entry.Status {
		case playabilityStatusBlocked:
			info.BlockedEntries++
		case playabilityStatusStable:
			info.StableEntries++
		case playabilityStatusWeak:
			info.WeakEntries++
		}
	}
	return info, nil
}

func doctorNCMEntryMatchesService(entry playabilityEntry, serviceID, serviceName string) bool {
	if strings.TrimSpace(serviceID) != "" && strings.TrimSpace(entry.ServiceID) != "" {
		return strings.TrimSpace(entry.ServiceID) == strings.TrimSpace(serviceID)
	}
	return strings.EqualFold(strings.TrimSpace(entry.ServiceName), strings.TrimSpace(serviceName))
}

func doctorNCMJSONOutput(flags *rootFlags, report doctorNCMReport) map[string]any {
	out := map[string]any{
		"action":      "doctor.ncm",
		"execution":   buildExecutionEnvelope(newExecutionOutput("doctor", "ncm", executionTargetFromFlags(flags), nil, doctorNCMExecutionResult(report))),
		"ok":          report.OK,
		"target":      report.Target,
		"service":     report.Service,
		"playability": report.Playability,
	}
	if report.Search != nil {
		out["search"] = report.Search
	}
	if len(report.Warnings) > 0 {
		out["warnings"] = report.Warnings
	}
	return out
}

func doctorNCMExecutionResult(report doctorNCMReport) map[string]any {
	result := map[string]any{
		"room":           strings.TrimSpace(report.Target.Room),
		"speakerIP":      strings.TrimSpace(report.Target.SpeakerIP),
		"coordinatorIP":  strings.TrimSpace(report.Target.CoordinatorIP),
		"serviceFound":   report.Service.Found,
		"tokenPresent":   report.Service.TokenPresent,
		"searchReady":    report.Service.SearchReady,
		"blockedEntries": report.Playability.BlockedEntries,
		"warningCount":   len(report.Warnings),
	}
	if report.Search != nil {
		result["query"] = strings.TrimSpace(report.Search.NormalizedQuery)
		result["candidateCount"] = report.Search.CandidateCount
		result["blockedFiltered"] = report.Search.BlockedFiltered
		if report.Search.Recommended != nil {
			result["recommendedTitle"] = strings.TrimSpace(report.Search.Recommended.Title)
			result["recommendedArtist"] = strings.TrimSpace(report.Search.Recommended.Artist)
			result["recommendedMatchLevel"] = strings.TrimSpace(report.Search.Recommended.MatchLevel)
		}
	}
	return compactMap(result)
}

func printDoctorNCMPlain(cmd *cobra.Command, report doctorNCMReport) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Room:\t\t%s\n", valueOrDash(report.Target.Room))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Speaker IP:\t%s\n", valueOrDash(report.Target.SpeakerIP))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Coordinator IP:\t%s\n", valueOrDash(report.Target.CoordinatorIP))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Service:\t\t%s (%s)\n", valueOrDash(report.Service.Name), valueOrDash(report.Service.ID))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Auth:\t\t%s required=%t tokenPresent=%t\n", valueOrDash(report.Service.Auth), report.Service.AuthRequired, report.Service.TokenPresent)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Household:\t%s\n", valueOrDash(report.Service.HouseholdID))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Search Ready:\t%t\n", report.Service.SearchReady)
	if len(report.Service.Categories) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Categories:\t%s\n", strings.Join(report.Service.Categories, ", "))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Playability:\tloaded=%t path=%s blocked=%d weak=%d stable=%d serviceEntries=%d total=%d\n",
		report.Playability.Loaded,
		valueOrDash(report.Playability.Path),
		report.Playability.BlockedEntries,
		report.Playability.WeakEntries,
		report.Playability.StableEntries,
		report.Playability.ServiceEntries,
		report.Playability.TotalEntries,
	)
	if report.Search != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Query:\t\t%s\n", valueOrDash(report.Search.NormalizedQuery))
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Search:\t\tperformed=%t rawTracks=%d candidates=%d blockedFiltered=%d\n",
			report.Search.Performed,
			report.Search.RawTrackCount,
			report.Search.CandidateCount,
			report.Search.BlockedFiltered,
		)
		if report.Search.Recommended != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Recommended:\t%s - %s [%s]\n",
				valueOrDash(report.Search.Recommended.Artist),
				valueOrDash(report.Search.Recommended.Title),
				valueOrDash(report.Search.Recommended.MatchLevel),
			)
		}
		for _, candidate := range report.Search.Candidates {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Candidate %d:\t%s - %s live=%d adapt=%d\n",
				candidate.Rank,
				valueOrDash(candidate.Artist),
				valueOrDash(candidate.Title),
				candidate.LivePenalty,
				candidate.AdaptationPenalty,
			)
		}
	}
	if len(report.Warnings) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings:\tnone")
		return
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings:")
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", warning)
	}
}

func availableServiceNames(services []sonos.MusicServiceDescriptor) []string {
	out := make([]string, 0, len(services))
	for _, service := range services {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func containsFold(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), needle) {
			return true
		}
	}
	return false
}

func doctorNCMAuthRequired(auth sonos.MusicServiceAuthType) bool {
	switch auth {
	case sonos.MusicServiceAuthAppLink, sonos.MusicServiceAuthDeviceLink:
		return true
	default:
		return false
	}
}
