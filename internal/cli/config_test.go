package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/appconfig"
)

func TestConfigSetGetUnset(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	// set defaultRoom
	{
		cmd := newConfigSetCmd(flags)
		cmd.SetOut(newDiscardWriter())
		cmd.SetErr(newDiscardWriter())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"defaultRoom", "Office"})
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("set defaultRoom: %v", err)
		}
	}

	// get
	{
		cmd := newConfigGetCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("get: %v", err)
		}
		if !strings.Contains(out.String(), "defaultRoom=Office") {
			t.Fatalf("unexpected get output: %s", out.String())
		}
	}

	// unset
	{
		cmd := newConfigUnsetCmd(flags)
		cmd.SetOut(newDiscardWriter())
		cmd.SetErr(newDiscardWriter())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"defaultRoom"})
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("unset: %v", err)
		}
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultRoom != "" {
		t.Fatalf("expected defaultRoom to be empty, got %q", cfg.DefaultRoom)
	}
}

func TestConfigSetRejectsInvalidFormat(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	cmd := newConfigSetCmd(flags)
	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"format", "banana"})
	if err := cmd.ExecuteContext(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfigPathPlainAndJSON(t *testing.T) {
	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	{
		flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}
		cmd := newConfigPathCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("path plain: %v", err)
		}
		if strings.TrimSpace(out.String()) != store.Path() {
			t.Fatalf("unexpected path output: %q", out.String())
		}
	}

	{
		flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
		cmd := newConfigPathCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("path json: %v", err)
		}
		if !strings.Contains(out.String(), "\"action\": \"config.path\"") || !strings.Contains(out.String(), "\"capability\": \"config\"") || !strings.Contains(out.String(), "\"operation\": \"path\"") || !strings.Contains(out.String(), store.Path()) || !strings.Contains(out.String(), "\"path\"") {
			t.Fatalf("unexpected json output: %q", out.String())
		}
	}
}

func TestConfigGetJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := store.Save(appconfig.Config{DefaultRoom: "Office", Format: "json"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	out, err := execute(t, newConfigGetCmd(flags))
	if err != nil {
		t.Fatalf("config get json: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"config.get\"") || !strings.Contains(out, "\"capability\": \"config\"") || !strings.Contains(out, "\"operation\": \"get\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"defaultRoom\": \"Office\"") || !strings.Contains(out, "\"format\": \"json\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestConfigSetJSONIncludesExecutionEnvelope(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	out, err := execute(t, newConfigSetCmd(flags), "defaultRoom", "Office")
	if err != nil {
		t.Fatalf("config set json: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"config.set\"") || !strings.Contains(out, "\"capability\": \"config\"") || !strings.Contains(out, "\"operation\": \"set\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"key\": \"defaultRoom\"") || !strings.Contains(out, "\"value\": \"Office\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestGetConfigKey(t *testing.T) {
	cfg := appconfig.Config{DefaultRoom: "Office", Format: "json"}
	if v, ok := getConfigKey(cfg, "defaultRoom"); !ok || v != "Office" {
		t.Fatalf("defaultRoom: ok=%v v=%q", ok, v)
	}
	if v, ok := getConfigKey(cfg, "format"); !ok || v != "json" {
		t.Fatalf("format: ok=%v v=%q", ok, v)
	}
	if _, ok := getConfigKey(cfg, "nope"); ok {
		t.Fatalf("expected ok=false")
	}
}
