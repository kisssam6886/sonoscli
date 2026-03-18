package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
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
		Short: "Manage Sonos schedules (MVP local store)",
		Long:  "MVP for sonos schedule. This command stores schedule definitions locally for planning/integration. Runtime execution worker will be added in next phase.",
	}

	cmd.AddCommand(newScheduleListCmd(flags))
	cmd.AddCommand(newScheduleAddCmd(flags))
	cmd.AddCommand(newScheduleRemoveCmd(flags))
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
			case "say", "play", "mode", "tv", "music", "scene":
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
	cmd.Flags().StringVar(&action, "action", "", "Action: say|play|mode|tv|music|scene")
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
