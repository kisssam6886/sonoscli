package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

type scheduleItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Room      string    `json:"room,omitempty"`
	Action    string    `json:"action"`
	Payload   string    `json:"payload,omitempty"`
	At        time.Time `json:"at"`
	CreatedAt time.Time `json:"createdAt"`
}

type scheduleFile struct {
	Version int            `json:"version"`
	Items   []scheduleItem `json:"items"`
}

func newScheduleCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage Sonos schedules (MVP local store + manual runner)",
		Long:  "MVP for sonos schedule. Stores schedule definitions locally and can manually run due jobs.",
	}

	cmd.AddCommand(newScheduleListCmd(flags))
	cmd.AddCommand(newScheduleAddCmd(flags))
	cmd.AddCommand(newScheduleRemoveCmd(flags))
	cmd.AddCommand(newScheduleRunCmd(flags))
	return cmd
}

func newScheduleListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scheduled jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := loadScheduleFile()
			if err != nil {
				return err
			}
			sort.Slice(f.Items, func(i, j int) bool { return f.Items[i].At.Before(f.Items[j].At) })
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"ok": true, "items": f.Items})
			}
			if len(f.Items) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No schedule items.")
				return nil
			}
			for _, it := range f.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s | %s | room=%s | action=%s | payload=%s\n", it.ID, it.At.Format(time.RFC3339), it.Room, it.Action, it.Payload)
			}
			return nil
		},
	}
}

func newScheduleAddCmd(flags *rootFlags) *cobra.Command {
	var (
		name    string
		action  string
		payload string
		at      string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a schedule item",
		RunE: func(cmd *cobra.Command, args []string) error {
			action = strings.TrimSpace(strings.ToLower(action))
			switch action {
			case "say", "play", "mode", "tv", "music":
			default:
				return fmt.Errorf("unsupported --action: %s", action)
			}
			when, err := time.Parse(time.RFC3339, strings.TrimSpace(at))
			if err != nil {
				return fmt.Errorf("invalid --at, use RFC3339: %w", err)
			}
			if when.Before(time.Now().Add(-30 * time.Second)) {
				return errors.New("--at is in the past")
			}

			f, err := loadScheduleFile()
			if err != nil {
				return err
			}

			item := scheduleItem{
				ID:        newScheduleID(),
				Name:      strings.TrimSpace(name),
				Room:      strings.TrimSpace(flags.Name),
				Action:    action,
				Payload:   strings.TrimSpace(payload),
				At:        when,
				CreatedAt: time.Now(),
			}
			f.Items = append(f.Items, item)
			if err := saveScheduleFile(f); err != nil {
				return err
			}

			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"ok": true, "item": item})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added schedule: %s at %s\n", item.ID, item.At.Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Human friendly schedule name")
	cmd.Flags().StringVar(&action, "action", "", "Action: say|play|mode|tv|music")
	cmd.Flags().StringVar(&payload, "payload", "", "Action payload (free text / serialized args)")
	cmd.Flags().StringVar(&at, "at", "", "Execution time (RFC3339), e.g. 2026-03-20T09:30:00+08:00")
	_ = cmd.MarkFlagRequired("action")
	_ = cmd.MarkFlagRequired("at")
	return cmd
}

func newScheduleRemoveCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a schedule item by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			f, err := loadScheduleFile()
			if err != nil {
				return err
			}
			next := make([]scheduleItem, 0, len(f.Items))
			removed := false
			for _, it := range f.Items {
				if it.ID == id {
					removed = true
					continue
				}
				next = append(next, it)
			}
			if !removed {
				return fmt.Errorf("schedule id not found: %s", id)
			}
			f.Items = next
			if err := saveScheduleFile(f); err != nil {
				return err
			}
			return writeOK(cmd, flags, "schedule.remove", map[string]any{"id": id})
		},
	}
}

func newScheduleRunCmd(flags *rootFlags) *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run due schedule items now",
		Long:  "Execute due schedule items from local schedule store. Use --id to run one specific item.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := loadScheduleFile()
			if err != nil {
				return err
			}
			now := time.Now()
			runItems := make([]scheduleItem, 0)
			keepItems := make([]scheduleItem, 0, len(f.Items))
			id = strings.TrimSpace(id)
			for _, it := range f.Items {
				if id != "" {
					if it.ID == id {
						runItems = append(runItems, it)
					} else {
						keepItems = append(keepItems, it)
					}
					continue
				}
				if !it.At.After(now) {
					runItems = append(runItems, it)
				} else {
					keepItems = append(keepItems, it)
				}
			}
			if len(runItems) == 0 {
				if isJSON(flags) {
					return writeJSON(cmd, map[string]any{"ok": true, "ran": 0})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No due schedules to run.")
				return nil
			}

			results := make([]map[string]any, 0, len(runItems))
			for _, it := range runItems {
				err := executeScheduleItem(cmd, flags, it)
				if err != nil {
					results = append(results, map[string]any{"id": it.ID, "ok": false, "error": err.Error()})
					keepItems = append(keepItems, it)
					continue
				}
				results = append(results, map[string]any{"id": it.ID, "ok": true})
			}
			f.Items = keepItems
			if err := saveScheduleFile(f); err != nil {
				return err
			}

			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"ok": true, "results": results, "remaining": len(keepItems)})
			}
			for _, r := range results {
				if r["ok"] == true {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ran: %s\n", r["id"])
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "failed: %s (%s)\n", r["id"], r["error"])
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Run only this schedule id")
	return cmd
}

func executeScheduleItem(cmd *cobra.Command, flags *rootFlags, it scheduleItem) error {
	f := *flags
	if strings.TrimSpace(it.Room) != "" {
		f.Name = strings.TrimSpace(it.Room)
	}
	ctx := cmd.Context()
	switch it.Action {
	case "play":
		c, err := coordinatorClient(ctx, &f)
		if err != nil {
			return err
		}
		return c.Play(ctx)
	case "tv":
		c, err := newSourceClient(ctx, &f)
		if err != nil {
			return err
		}
		mem, err := resolveTargetMember(ctx, &f)
		if err != nil {
			return err
		}
		uri := "x-sonos-htastream:" + mem.UUID + ":spdif"
		if err := c.SetAVTransportURI(ctx, uri, ""); err != nil {
			return err
		}
		return c.Play(ctx)
	case "music":
		c, err := newSourceClient(ctx, &f)
		if err != nil {
			return err
		}
		mem, err := resolveTargetMember(ctx, &f)
		if err != nil {
			return err
		}
		uri := "x-rincon-queue:" + mem.UUID + "#0"
		if err := c.SetAVTransportURI(ctx, uri, ""); err != nil {
			return err
		}
		return c.Play(ctx)
	case "mode":
		c, err := coordinatorClient(ctx, &f)
		if err != nil {
			return err
		}
		return applyMode(c, ctx, strings.TrimSpace(strings.ToLower(it.Payload)))
	case "say":
		c, err := coordinatorClient(ctx, &f)
		if err != nil {
			return err
		}
		uri := strings.TrimSpace(it.Payload)
		if uri == "" {
			return errors.New("say payload must be audio URI")
		}
		return c.PlayURI(ctx, uri, "")
	default:
		return fmt.Errorf("unsupported action: %s", it.Action)
	}
}

func applyMode(c *sonos.Client, ctx context.Context, mode string) error {
	switch mode {
	case "shuffle":
		return c.SetPlayMode(ctx, sonos.PlayModeShuffle)
	case "shuffle-norepeat":
		return c.SetPlayMode(ctx, sonos.PlayModeShuffleNoRepeat)
	case "repeat":
		return c.SetPlayMode(ctx, sonos.PlayModeRepeatAll)
	case "repeat-one":
		return c.SetPlayMode(ctx, sonos.PlayModeRepeatOne)
	case "normal", "":
		return c.SetPlayMode(ctx, sonos.PlayModeNormal)
	default:
		return fmt.Errorf("unsupported mode payload: %s", mode)
	}
}

func scheduleFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonoscli", "schedule.json"), nil
}

func loadScheduleFile() (scheduleFile, error) {
	p, err := scheduleFilePath()
	if err != nil {
		return scheduleFile{}, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return scheduleFile{Version: 1, Items: nil}, nil
		}
		return scheduleFile{}, err
	}
	var f scheduleFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return scheduleFile{}, fmt.Errorf("parse schedule file: %w", err)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	return f, nil
}

func saveScheduleFile(f scheduleFile) error {
	p, err := scheduleFilePath()
	if err != nil {
		return err
	}
	if f.Version == 0 {
		f.Version = 1
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func newScheduleID() string {
	return fmt.Sprintf("sch_%d", time.Now().UnixNano())
}
