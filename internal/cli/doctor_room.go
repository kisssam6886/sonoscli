package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

type doctorRoomClient interface {
	GetMediaInfo(ctx context.Context) (sonos.MediaInfo, error)
	GetPositionInfo(ctx context.Context) (sonos.PositionInfo, error)
	GetTransportInfo(ctx context.Context) (sonos.TransportInfo, error)
	GetVolume(ctx context.Context) (int, error)
	GetMute(ctx context.Context) (bool, error)
}

type doctorRoomReport struct {
	OK          bool                 `json:"ok"`
	Target      sonos.Member         `json:"target"`
	Coordinator sonos.Member         `json:"coordinator"`
	Group       doctorRoomGroupInfo  `json:"group"`
	Source      doctorRoomSourceInfo `json:"source"`
	Status      doctorRoomStatusInfo `json:"status"`
	Warnings    []string             `json:"warnings,omitempty"`
}

type doctorRoomGroupInfo struct {
	ID                    string         `json:"id,omitempty"`
	MemberCount           int            `json:"memberCount"`
	VisibleMemberCount    int            `json:"visibleMemberCount"`
	InvisibleMemberCount  int            `json:"invisibleMemberCount"`
	CoordinatorRedirected bool           `json:"coordinatorRedirected"`
	Members               []sonos.Member `json:"members,omitempty"`
}

type doctorRoomSourceInfo struct {
	CurrentURI        string `json:"currentURI,omitempty"`
	TrackURI          string `json:"trackURI,omitempty"`
	Kind              string `json:"kind"`
	RestorePath       string `json:"restorePath"`
	Active            bool   `json:"active"`
	Playing           bool   `json:"playing"`
	CanTestSayRestore bool   `json:"canTestSayRestore"`
}

type doctorRoomStatusInfo struct {
	State    string `json:"state,omitempty"`
	Track    string `json:"track,omitempty"`
	Time     string `json:"time,omitempty"`
	Duration string `json:"duration,omitempty"`
	Volume   int    `json:"volume"`
	Mute     bool   `json:"mute"`
	Title    string `json:"title,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
}

var newDoctorRoomClient = func(ctx context.Context, flags *rootFlags) (doctorRoomClient, error) {
	return coordinatorClient(ctx, flags)
}

func newDoctorRoomCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:          "room",
		Short:        "Inspect room topology, source kind, and say-restore readiness",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := buildDoctorRoomReport(cmd.Context(), flags)
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, doctorRoomJSONOutput(flags, report))
			}
			printDoctorRoomPlain(cmd, report)
			return nil
		},
	}
}

func buildDoctorRoomReport(ctx context.Context, flags *rootFlags) (doctorRoomReport, error) {
	if err := validateTarget(flags); err != nil {
		return doctorRoomReport{}, err
	}

	tg, err := newTopologyGetter(ctx, flags.Timeout)
	if err != nil {
		return doctorRoomReport{}, err
	}
	top, err := tg.GetTopology(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}

	target, err := resolveMember(top, flags.Name, flags.IP)
	if err != nil {
		return doctorRoomReport{}, err
	}
	group, ok := top.GroupForIP(target.IP)
	if !ok {
		return doctorRoomReport{}, newStateInconsistentError("target group missing in topology", map[string]any{
			"action":    "doctor.room",
			"room":      strings.TrimSpace(target.Name),
			"speakerIP": strings.TrimSpace(target.IP),
		})
	}

	c, err := newDoctorRoomClient(ctx, flags)
	if err != nil {
		return doctorRoomReport{}, err
	}

	media, err := c.GetMediaInfo(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}
	position, err := c.GetPositionInfo(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}
	transport, err := c.GetTransportInfo(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}
	volume, err := c.GetVolume(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}
	mute, err := c.GetMute(ctx)
	if err != nil {
		return doctorRoomReport{}, err
	}

	status := doctorRoomStatusInfo{
		State:    normalizeTransportState(transport.State),
		Track:    strings.TrimSpace(position.Track),
		Time:     normalizeRelTime(position.RelTime),
		Duration: strings.TrimSpace(position.TrackDuration),
		Volume:   volume,
		Mute:     mute,
	}
	if np, ok := sonos.ParseNowPlaying(position.TrackMeta); ok {
		status.Title = strings.TrimSpace(np.Title)
		status.Artist = strings.TrimSpace(np.Artist)
		status.Album = strings.TrimSpace(np.Album)
	}

	source := classifyDoctorRoomSource(media.CurrentURI, position.TrackURI, transport.State)
	report := doctorRoomReport{
		Target:      target,
		Coordinator: group.Coordinator,
		Group: doctorRoomGroupInfo{
			ID:                    strings.TrimSpace(group.ID),
			MemberCount:           len(group.Members),
			VisibleMemberCount:    countVisibleMembers(group.Members),
			InvisibleMemberCount:  countInvisibleMembers(group.Members),
			CoordinatorRedirected: strings.TrimSpace(group.Coordinator.IP) != "" && strings.TrimSpace(target.IP) != strings.TrimSpace(group.Coordinator.IP),
			Members:               group.Members,
		},
		Source: source,
		Status: status,
	}
	report.Warnings = doctorRoomWarnings(report)
	report.OK = len(report.Warnings) == 0
	return report, nil
}

func doctorRoomJSONOutput(flags *rootFlags, report doctorRoomReport) map[string]any {
	out := map[string]any{
		"action":      "doctor.room",
		"execution":   buildExecutionEnvelope(newExecutionOutput("doctor", "room", executionTargetFromFlags(flags), nil, doctorRoomExecutionResult(report))),
		"ok":          report.OK,
		"target":      report.Target,
		"coordinator": report.Coordinator,
		"group":       report.Group,
		"source":      report.Source,
		"status":      report.Status,
	}
	if len(report.Warnings) > 0 {
		out["warnings"] = report.Warnings
	}
	return out
}

func doctorRoomExecutionResult(report doctorRoomReport) map[string]any {
	return compactMap(map[string]any{
		"room":              strings.TrimSpace(report.Target.Name),
		"speakerIP":         strings.TrimSpace(report.Target.IP),
		"coordinatorIP":     strings.TrimSpace(report.Coordinator.IP),
		"groupMemberCount":  report.Group.MemberCount,
		"sourceKind":        strings.TrimSpace(report.Source.Kind),
		"restorePath":       strings.TrimSpace(report.Source.RestorePath),
		"state":             strings.TrimSpace(report.Status.State),
		"canTestSayRestore": report.Source.CanTestSayRestore,
		"warningCount":      len(report.Warnings),
	})
}

func printDoctorRoomPlain(cmd *cobra.Command, report doctorRoomReport) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target:\t\t%s (%s)\n", valueOrDash(report.Target.Name), valueOrDash(report.Target.IP))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Coordinator:\t%s (%s)\n", valueOrDash(report.Coordinator.Name), valueOrDash(report.Coordinator.IP))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Group:\t\t%s\n", valueOrDash(report.Group.ID))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Group Members:\t%d visible=%d invisible=%d redirected=%t\n",
		report.Group.MemberCount,
		report.Group.VisibleMemberCount,
		report.Group.InvisibleMemberCount,
		report.Group.CoordinatorRedirected,
	)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Source Kind:\t%s\n", valueOrDash(report.Source.Kind))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Restore Path:\t%s\n", valueOrDash(report.Source.RestorePath))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Current URI:\t%s\n", valueOrDash(report.Source.CurrentURI))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Track URI:\t%s\n", valueOrDash(report.Source.TrackURI))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "State:\t\t%s\n", valueOrDash(report.Status.State))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Track:\t\t%s\n", valueOrDash(report.Status.Track))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Time:\t\t%s / %s\n", valueOrDash(report.Status.Time), valueOrDash(report.Status.Duration))
	if strings.TrimSpace(report.Status.Title) != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title:\t\t%s\n", report.Status.Title)
	}
	if strings.TrimSpace(report.Status.Artist) != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Artist:\t\t%s\n", report.Status.Artist)
	}
	if strings.TrimSpace(report.Status.Album) != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Album:\t\t%s\n", report.Status.Album)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Volume:\t\t%d\n", report.Status.Volume)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mute:\t\t%v\n", report.Status.Mute)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Can Test say Restore:\t%t\n", report.Source.CanTestSayRestore)
	if len(report.Warnings) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings:\tnone")
		return
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings:")
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", warning)
	}
}

func classifyDoctorRoomSource(currentURI string, trackURI string, transportState string) doctorRoomSourceInfo {
	currentURI = strings.TrimSpace(currentURI)
	trackURI = strings.TrimSpace(trackURI)
	state := normalizeTransportState(transportState)
	kind := doctorRoomSourceKind(currentURI)
	restorePath := "direct"
	switch {
	case currentURI == "":
		restorePath = "none"
	case isQueueSourceURI(currentURI):
		restorePath = "queue"
	}
	active := currentURI != ""
	playing := state == "PLAYING"
	return doctorRoomSourceInfo{
		CurrentURI:        currentURI,
		TrackURI:          trackURI,
		Kind:              kind,
		RestorePath:       restorePath,
		Active:            active,
		Playing:           playing,
		CanTestSayRestore: active && playing && restorePath != "none",
	}
}

func doctorRoomSourceKind(uri string) string {
	uri = strings.ToLower(strings.TrimSpace(uri))
	switch {
	case uri == "":
		return "empty"
	case strings.HasPrefix(uri, "x-sonos-htastream:"):
		return "tv"
	case strings.HasPrefix(uri, "x-rincon-queue:"):
		return "queue"
	case strings.HasPrefix(uri, "x-rincon-stream:"):
		return "line-in"
	case strings.HasPrefix(uri, "x-rincon-mp3radio:"),
		strings.HasPrefix(uri, "x-sonosapi-stream:"),
		strings.HasPrefix(uri, "x-sonosapi-radio:"):
		return "radio"
	case strings.HasPrefix(uri, "x-sonos-http:"),
		strings.HasPrefix(uri, "x-file-cifs:"):
		return "music-track"
	case strings.HasPrefix(uri, "http://"),
		strings.HasPrefix(uri, "https://"):
		return "direct-http"
	case strings.HasPrefix(uri, "x-rincon:"):
		return "group-link"
	default:
		return "unknown"
	}
}

func doctorRoomWarnings(report doctorRoomReport) []string {
	warnings := []string{}
	if !report.Target.IsVisible {
		warnings = append(warnings, "target room is not a visible room in topology; avoid testing bonded satellites/subs directly")
	}
	if strings.TrimSpace(report.Source.CurrentURI) == "" {
		warnings = append(warnings, "current source is empty; start TV or music before testing say restore")
		return warnings
	}
	if !report.Source.Playing {
		warnings = append(warnings, "transport is not PLAYING; start TV or music before testing say restore")
	}
	if report.Source.Kind == "unknown" {
		warnings = append(warnings, "current source kind is unknown; say restore will fall back to direct-source recovery")
	}
	if report.Source.RestorePath == "queue" && strings.TrimSpace(report.Status.Track) == "" {
		warnings = append(warnings, "queue source has no current track number; queue restore verification may be incomplete")
	}
	if report.Source.Kind == "music-track" &&
		report.Status.State == "STOPPED" &&
		normalizeRelTime(report.Status.Time) == "0:00:00" {
		warnings = append(warnings, "music baseline is stopped at 0:00:00; verify the selected track itself can play before testing say restore")
	}
	return warnings
}

func countVisibleMembers(members []sonos.Member) int {
	count := 0
	for _, member := range members {
		if member.IsVisible {
			count++
		}
	}
	return count
}

func countInvisibleMembers(members []sonos.Member) int {
	count := 0
	for _, member := range members {
		if !member.IsVisible {
			count++
		}
	}
	return count
}
