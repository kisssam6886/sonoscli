package cli

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestScheduleListJSONIncludesExecutionEnvelope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	flags := &rootFlags{Format: formatJSON}
	cmd := newScheduleCmd(flags)
	var out captureWriter
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "\"action\": \"schedule.list\"") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if !strings.Contains(out.String(), "\"capability\": \"schedule\"") {
		t.Fatalf("missing execution envelope: %q", out.String())
	}
}

func TestScheduleAddJSONIncludesExecutionEnvelope(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("XDG_CONFIG_HOME", temp)

	flags := &rootFlags{Format: formatJSON, Name: "客厅"}
	cmd := newScheduleCmd(flags)
	var out captureWriter
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	when := time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)
	cmd.SetArgs([]string{"add", "--action", "music", "--at", when, "--name", "Morning Music"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "\"action\": \"schedule.add\"") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if !strings.Contains(out.String(), "\"operation\": \"add\"") {
		t.Fatalf("missing operation: %q", out.String())
	}
	if !strings.Contains(out.String(), "\"room\": \"客厅\"") {
		t.Fatalf("missing target room: %q", out.String())
	}
}
