package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/steipete/sonoscli/internal/appconfig"
)

func TestBuildDoctorReport_WarnsWhenExternalBinaryIsUsedInsideRepo(t *testing.T) {
	repoRoot := makeFakeRepo(t)
	external := filepath.Join(t.TempDir(), "opt", "homebrew", "bin", "sonos")
	mustWriteFile(t, external, "#!/bin/sh\n")

	restore := stubDoctorEnv(t, repoRoot, external, external)
	defer restore()

	report, err := buildDoctorReport()
	if err != nil {
		t.Fatalf("buildDoctorReport: %v", err)
	}
	if report.OK {
		t.Fatalf("expected warnings, got ok report")
	}
	if !report.Binary.RepoBinaryExists {
		t.Fatalf("expected repo binary to exist")
	}
	if report.Binary.RunningRepoBinary {
		t.Fatalf("expected runningRepoBinary=false")
	}
	if len(report.Warnings) < 2 {
		t.Fatalf("expected >=2 warnings, got %#v", report.Warnings)
	}
	if !strings.Contains(report.Warnings[0], "not the repo binary") {
		t.Fatalf("unexpected first warning: %q", report.Warnings[0])
	}
}

func TestBuildDoctorReport_OKWhenRunningRepoBinary(t *testing.T) {
	repoRoot := makeFakeRepo(t)
	repoBinary := filepath.Join(repoRoot, "sonos")

	restore := stubDoctorEnv(t, repoRoot, repoBinary, repoBinary)
	defer restore()

	report, err := buildDoctorReport()
	if err != nil {
		t.Fatalf("buildDoctorReport: %v", err)
	}
	if !report.OK {
		t.Fatalf("expected ok report, got warnings %#v", report.Warnings)
	}
	if !report.Binary.RunningRepoBinary {
		t.Fatalf("expected runningRepoBinary=true")
	}
	if !report.Binary.PathMatchesCurrent {
		t.Fatalf("expected pathMatchesCurrent=true")
	}
}

func TestDoctorCmd_JSONIncludesWarnings(t *testing.T) {
	oldCollect := collectDoctorReport
	t.Cleanup(func() { collectDoctorReport = oldCollect })

	collectDoctorReport = func() (doctorReport, error) {
		return doctorReport{
			OK:      false,
			Version: Version,
			Binary: doctorBinaryInfo{
				ExecutableRealPath: "/opt/homebrew/bin/sonos",
				RepoRoot:           "/tmp/repo",
				RepoBinaryPath:     "/tmp/repo/sonos",
			},
			Warnings: []string{"plain `sonos` resolves to /opt/homebrew/bin/sonos instead of the repo binary /tmp/repo/sonos"},
		}, nil
	}

	flags := &rootFlags{Format: formatJSON}
	out, err := execute(t, newDoctorCmd(flags))
	if err != nil {
		t.Fatalf("doctor json: %v", err)
	}
	if !strings.Contains(out, "\"action\": \"doctor\"") || !strings.Contains(out, "\"capability\": \"doctor\"") || !strings.Contains(out, "\"operation\": \"report\"") {
		t.Fatalf("unexpected envelope output: %q", out)
	}
	if !strings.Contains(out, "\"warnings\"") || !strings.Contains(out, "/opt/homebrew/bin/sonos") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\"ok\": false") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRootWarnsWhenRepoBinaryMismatchDetected(t *testing.T) {
	oldLoad := loadAppConfig
	oldCollect := collectDoctorReport
	oldConfigStore := newConfigStore
	t.Cleanup(func() {
		loadAppConfig = oldLoad
		collectDoctorReport = oldCollect
		newConfigStore = oldConfigStore
	})

	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{Format: "plain"}.Normalize(), nil
	}
	collectDoctorReport = func() (doctorReport, error) {
		return doctorReport{
			OK: false,
			Binary: doctorBinaryInfo{
				RepoRoot:       "/tmp/repo",
				RepoBinaryPath: "/tmp/repo/sonos",
			},
			Warnings: []string{"current executable is /opt/homebrew/bin/sonos, not the repo binary /tmp/repo/sonos"},
		}, nil
	}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	root, _, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}

	var stdout captureWriter
	var stderr captureWriter
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"config", "path"})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: current executable is /opt/homebrew/bin/sonos") {
		t.Fatalf("expected warning on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "hint: cd /tmp/repo && go build -o ./sonos ./cmd/sonos") {
		t.Fatalf("expected hint on stderr, got %q", stderr.String())
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatalf("expected command output on stdout")
	}
}

func makeFakeRepo(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "go.mod"), "module github.com/steipete/sonoscli\n\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(repoRoot, "sonos"), "#!/bin/sh\n")
	return repoRoot
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func stubDoctorEnv(t *testing.T, wd string, execPath string, lookupPath string) func() {
	t.Helper()
	oldExec := osExecutablePath
	oldLookup := lookPathExec
	oldWD := currentWorkingDir
	oldEval := evalPathSymlinks
	oldRead := readFilePath
	oldBuildInfo := readRuntimeBuildInfo

	osExecutablePath = func() (string, error) { return execPath, nil }
	lookPathExec = func(file string) (string, error) { return lookupPath, nil }
	currentWorkingDir = func() (string, error) { return wd, nil }
	evalPathSymlinks = func(path string) (string, error) { return filepath.Abs(path) }
	readFilePath = os.ReadFile
	readRuntimeBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{GoVersion: "go1.22.0"}, true
	}

	return func() {
		osExecutablePath = oldExec
		lookPathExec = oldLookup
		currentWorkingDir = oldWD
		evalPathSymlinks = oldEval
		readFilePath = oldRead
		readRuntimeBuildInfo = oldBuildInfo
	}
}
