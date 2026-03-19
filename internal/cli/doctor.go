package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

const sonosModulePath = "module github.com/steipete/sonoscli"

type doctorReport struct {
	OK           bool               `json:"ok"`
	Version      string             `json:"version"`
	Build        doctorBuildInfo    `json:"build"`
	Binary       doctorBinaryInfo   `json:"binary"`
	Capabilities doctorCapabilities `json:"capabilities"`
	Warnings     []string           `json:"warnings,omitempty"`
}

type doctorBuildInfo struct {
	GoVersion   string `json:"goVersion,omitempty"`
	MainVersion string `json:"mainVersion,omitempty"`
	VCSRevision string `json:"vcsRevision,omitempty"`
	VCSTime     string `json:"vcsTime,omitempty"`
	VCSModified string `json:"vcsModified,omitempty"`
}

type doctorBinaryInfo struct {
	ExecutablePath     string `json:"executablePath,omitempty"`
	ExecutableRealPath string `json:"executableRealPath,omitempty"`
	PathLookup         string `json:"pathLookup,omitempty"`
	PathLookupRealPath string `json:"pathLookupRealPath,omitempty"`
	PathMatchesCurrent bool   `json:"pathMatchesCurrent"`
	RepoRoot           string `json:"repoRoot,omitempty"`
	RepoBinaryPath     string `json:"repoBinaryPath,omitempty"`
	RepoBinaryExists   bool   `json:"repoBinaryExists"`
	RunningRepoBinary  bool   `json:"runningRepoBinary"`
}

type doctorCapabilities struct {
	SMAPISongOpenQueuePath bool `json:"smapiSongOpenQueuePath"`
	QueueTransitionHeal    bool `json:"queueTransitionHeal"`
	MachineReadableErrors  bool `json:"machineReadableErrors"`
}

func doctorExecutionResult(report doctorReport) map[string]any {
	return compactMap(map[string]any{
		"reportOK":           report.OK,
		"warningCount":       len(report.Warnings),
		"repoBinaryExists":   report.Binary.RepoBinaryExists,
		"runningRepoBinary":  report.Binary.RunningRepoBinary,
		"pathMatchesCurrent": report.Binary.PathMatchesCurrent,
	})
}

func doctorJSONOutput(report doctorReport) map[string]any {
	out := map[string]any{
		"action":       "doctor",
		"execution":    buildExecutionEnvelope(newExecutionOutput("doctor", "report", nil, nil, doctorExecutionResult(report))),
		"ok":           report.OK,
		"version":      report.Version,
		"build":        report.Build,
		"binary":       report.Binary,
		"capabilities": report.Capabilities,
	}
	if len(report.Warnings) > 0 {
		out["warnings"] = report.Warnings
	}
	return out
}

var collectDoctorReport = buildDoctorReport

var (
	osExecutablePath     = os.Executable
	lookPathExec         = exec.LookPath
	currentWorkingDir    = os.Getwd
	evalPathSymlinks     = filepath.EvalSymlinks
	readFilePath         = os.ReadFile
	readRuntimeBuildInfo = debug.ReadBuildInfo
)

func newDoctorCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "doctor",
		Short:        "Self-check the current CLI binary, path resolution, and build info",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := collectDoctorReport()
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, doctorJSONOutput(report))
			}
			printDoctorPlain(cmd, report)
			return nil
		},
	}
	cmd.AddCommand(newDoctorRoomCmd(flags))
	return cmd
}

func buildDoctorReport() (doctorReport, error) {
	report := doctorReport{
		Version: Version,
		Build:   collectBuildInfo(),
		Capabilities: doctorCapabilities{
			SMAPISongOpenQueuePath: true,
			QueueTransitionHeal:    true,
			MachineReadableErrors:  true,
		},
	}

	binaryInfo, warnings, err := collectBinaryInfo()
	if err != nil {
		return report, err
	}
	report.Binary = binaryInfo
	report.Warnings = warnings
	report.OK = len(warnings) == 0
	return report, nil
}

func collectBuildInfo() doctorBuildInfo {
	out := doctorBuildInfo{}
	if bi, ok := readRuntimeBuildInfo(); ok {
		out.GoVersion = strings.TrimSpace(bi.GoVersion)
		out.MainVersion = strings.TrimSpace(bi.Main.Version)
		for _, setting := range bi.Settings {
			switch strings.TrimSpace(setting.Key) {
			case "vcs.revision":
				out.VCSRevision = strings.TrimSpace(setting.Value)
			case "vcs.time":
				out.VCSTime = strings.TrimSpace(setting.Value)
			case "vcs.modified":
				out.VCSModified = strings.TrimSpace(setting.Value)
			}
		}
	}
	return out
}

func collectBinaryInfo() (doctorBinaryInfo, []string, error) {
	info := doctorBinaryInfo{}

	execPath, err := osExecutablePath()
	if err != nil {
		return info, nil, err
	}
	info.ExecutablePath = execPath
	info.ExecutableRealPath = resolvePath(execPath)

	if lookedUp, err := lookPathExec("sonos"); err == nil {
		info.PathLookup = lookedUp
		info.PathLookupRealPath = resolvePath(lookedUp)
		info.PathMatchesCurrent = sameResolvedPath(info.PathLookupRealPath, info.ExecutableRealPath)
	}

	wd, err := currentWorkingDir()
	if err != nil {
		return info, nil, err
	}

	repoRoot, ok := findSonosRepoRoot(wd)
	if ok {
		info.RepoRoot = repoRoot
		info.RepoBinaryPath = filepath.Join(repoRoot, "sonos")
		info.RepoBinaryExists = fileExists(info.RepoBinaryPath)
		info.RunningRepoBinary = sameResolvedPath(resolvePath(info.RepoBinaryPath), info.ExecutableRealPath)
	}

	return info, doctorWarnings(info), nil
}

func findSonosRepoRoot(start string) (string, bool) {
	dir := strings.TrimSpace(start)
	if dir == "" {
		return "", false
	}
	dir = filepath.Clean(dir)

	for {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := readFilePath(goModPath)
		if err == nil && bytes.Contains(data, []byte(sonosModulePath)) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func doctorWarnings(info doctorBinaryInfo) []string {
	if !looksLikeSonosExecutable(info.ExecutableRealPath) {
		return nil
	}
	if strings.TrimSpace(info.RepoRoot) == "" {
		return nil
	}

	warnings := []string{}
	repoBinaryReal := resolvePath(info.RepoBinaryPath)
	if info.RepoBinaryExists && !sameResolvedPath(repoBinaryReal, info.ExecutableRealPath) {
		warnings = append(warnings, fmt.Sprintf("current executable is %s, not the repo binary %s", info.ExecutableRealPath, info.RepoBinaryPath))
	}
	if info.RepoBinaryExists && info.PathLookupRealPath != "" && !sameResolvedPath(info.PathLookupRealPath, repoBinaryReal) {
		warnings = append(warnings, fmt.Sprintf("plain `sonos` resolves to %s instead of the repo binary %s", info.PathLookupRealPath, info.RepoBinaryPath))
	}
	if !info.RepoBinaryExists && !pathInsideRepo(info.ExecutableRealPath, info.RepoRoot) {
		warnings = append(warnings, fmt.Sprintf("repo binary %s is missing; build it before validating SMAPI/Netease fixes", info.RepoBinaryPath))
	}
	return warnings
}

func printDoctorPlain(cmd *cobra.Command, report doctorReport) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", report.Version)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Build Go: %s\n", valueOrDash(report.Build.GoVersion))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Build Revision: %s\n", valueOrDash(report.Build.VCSRevision))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Build Time: %s\n", valueOrDash(report.Build.VCSTime))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Build Modified: %s\n", valueOrDash(report.Build.VCSModified))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Executable: %s\n", valueOrDash(report.Binary.ExecutableRealPath))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PATH sonos: %s\n", valueOrDash(report.Binary.PathLookupRealPath))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo Root: %s\n", valueOrDash(report.Binary.RepoRoot))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo Binary: %s\n", valueOrDash(report.Binary.RepoBinaryPath))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo Binary Exists: %t\n", report.Binary.RepoBinaryExists)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Running Repo Binary: %t\n", report.Binary.RunningRepoBinary)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Capabilities: smapiSongOpenQueuePath=%t queueTransitionHeal=%t machineReadableErrors=%t\n",
		report.Capabilities.SMAPISongOpenQueuePath,
		report.Capabilities.QueueTransitionHeal,
		report.Capabilities.MachineReadableErrors,
	)

	if len(report.Warnings) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings: none")
		return
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Warnings:")
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", warning)
	}
	if strings.TrimSpace(report.Binary.RepoRoot) != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Suggested fix: cd %s && go build -o ./sonos ./cmd/sonos && ./sonos doctor --format json\n", report.Binary.RepoRoot)
	}
}

func maybeWarnBinaryMismatch(cmd *cobra.Command, flags *rootFlags) {
	if cmd == nil {
		return
	}
	if cmd.CommandPath() == "sonos" || cmd.CommandPath() == "sonos doctor" {
		return
	}

	report, err := collectDoctorReport()
	if err != nil || len(report.Warnings) == 0 {
		return
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", report.Warnings[0])
	if len(report.Warnings) > 1 {
		for _, warning := range report.Warnings[1:] {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
		}
	}
	if strings.TrimSpace(report.Binary.RepoRoot) != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "hint: cd %s && go build -o ./sonos ./cmd/sonos && ./sonos doctor --format json\n", report.Binary.RepoRoot)
	}

	_ = flags
}

func looksLikeSonosExecutable(path string) bool {
	return strings.EqualFold(filepath.Base(strings.TrimSpace(path)), "sonos")
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func resolvePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if resolved, err := evalPathSymlinks(path); err == nil && strings.TrimSpace(resolved) != "" {
		return resolved
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return filepath.Clean(path)
}

func sameResolvedPath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func pathInsideRepo(path string, repoRoot string) bool {
	path = strings.TrimSpace(path)
	repoRoot = strings.TrimSpace(repoRoot)
	if path == "" || repoRoot == "" {
		return false
	}
	path = filepath.Clean(path)
	repoRoot = filepath.Clean(repoRoot)
	if path == repoRoot {
		return true
	}
	return strings.HasPrefix(path, repoRoot+string(filepath.Separator))
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}
