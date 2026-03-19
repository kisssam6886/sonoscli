package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

type topologyGetter interface {
	GetTopology(ctx context.Context) (sonos.Topology, error)
}

type groupingClient interface {
	JoinGroup(ctx context.Context, coordinatorUUID string) error
	LeaveGroup(ctx context.Context) error
}

var newTopologyGetter = func(ctx context.Context, timeout time.Duration) (topologyGetter, error) {
	devs, err := sonos.Discover(ctx, sonos.DiscoverOptions{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, newTargetNotFoundError("no speakers found", nil, map[string]any{
			"action":     "group.topology",
			"resolution": "discover",
		})
	}
	return sonos.NewClient(devs[0].IP, timeout), nil
}

var newGroupingClient = func(ip string, timeout time.Duration) groupingClient {
	return sonos.NewClient(ip, timeout)
}

func executionGroupFields(group sonos.Group) map[string]any {
	return compactMap(map[string]any{
		"id":          strings.TrimSpace(group.ID),
		"coordinator": executionMemberFields(group.Coordinator),
		"memberCount": len(group.Members),
	})
}

func newGroupCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Inspect and control grouping",
	}
	cmd.AddCommand(newGroupStatusCmd(flags))
	cmd.AddCommand(newGroupJoinCmd(flags))
	cmd.AddCommand(newGroupUnjoinCmd(flags))
	cmd.AddCommand(newGroupSoloCmd(flags))
	cmd.AddCommand(newGroupPartyCmd(flags))
	cmd.AddCommand(newGroupDissolveCmd(flags))
	cmd.AddCommand(newGroupVolumeCmd(flags))
	cmd.AddCommand(newGroupMuteCmd(flags))
	return cmd
}

func newGroupStatusCmd(flags *rootFlags) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Show current groups and members",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			if !all {
				for i := range top.Groups {
					var visible []sonos.Member
					for _, m := range top.Groups[i].Members {
						if m.IsVisible {
							visible = append(visible, m)
						}
					}
					top.Groups[i].Members = visible
				}
			}

			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "group.status", newExecutionOutput("group", "status", nil, map[string]any{
					"all": all,
				}, map[string]any{
					"groupCount": len(top.Groups),
				}), map[string]any{
					"groups": top.Groups,
					"byName": top.ByName,
					"byIP":   top.ByIP,
				})
			}
			if isTSV(flags) {
				for _, g := range top.Groups {
					coord := g.Coordinator
					for _, m := range g.Members {
						if !all && !m.IsVisible {
							continue
						}
						role := "member"
						if m.IsCoordinator {
							role = "coordinator"
						}
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\t%s\n", g.ID, coord.Name, coord.IP, m.Name, m.IP, role)
					}
				}
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			for _, g := range top.Groups {
				coord := g.Coordinator
				_, _ = fmt.Fprintf(w, "Group:\t%s\t(%s)\n", coord.Name, coord.IP)
				for _, m := range g.Members {
					if !all && !m.IsVisible {
						continue
					}
					mark := " "
					if m.IsCoordinator {
						mark = "*"
					}
					_, _ = fmt.Fprintf(w, "  %s\t%s\t(%s)\n", mark, m.Name, m.IP)
				}
				_, _ = fmt.Fprintln(w)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Include invisible/bonded devices (advanced)")
	return cmd
}

func newGroupJoinCmd(flags *rootFlags) *cobra.Command {
	var to string

	cmd := &cobra.Command{
		Use:          "join --to <name-or-ip>",
		Short:        "Join another group",
		Long:         "Makes the target speaker (via --name/--ip) join the group coordinated by --to.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			to = strings.TrimSpace(to)
			if to == "" {
				return newInvalidArgumentError("--to is required", map[string]any{
					"action": "group.join",
					"flag":   "to",
				})
			}

			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			joiner, err := resolveMember(top, flags.Name, flags.IP)
			if err != nil {
				return err
			}
			dest, err := resolveMember(top, to, "")
			if err != nil {
				return err
			}

			joinerGroup, _ := top.GroupForIP(joiner.IP)
			destGroup, ok := top.GroupForIP(dest.IP)
			if !ok {
				return newStateInconsistentError("destination speaker not found in any group", map[string]any{
					"action":    "group.join",
					"speaker":   dest.Name,
					"speakerIP": dest.IP,
				})
			}

			if joinerGroup.ID != "" && joinerGroup.ID == destGroup.ID {
				return writeExecutionOK(cmd, flags, "group.join", newExecutionOutput("group", "join", executionTargetFromFlags(flags), map[string]any{
					"to": executionMemberFields(dest),
				}, map[string]any{
					"joiner":  executionMemberFields(joiner),
					"to":      executionMemberFields(dest),
					"skipped": true,
				}), map[string]any{
					"joiner":  joiner,
					"to":      dest,
					"skipped": true,
				})
			}
			if destGroup.Coordinator.UUID == "" {
				return newStateInconsistentError("destination group coordinator UUID missing", map[string]any{
					"action":    "group.join",
					"speaker":   dest.Name,
					"speakerIP": dest.IP,
					"groupID":   destGroup.ID,
				})
			}

			c := newGroupingClient(joiner.IP, flags.Timeout)
			if err := c.JoinGroup(cmd.Context(), destGroup.Coordinator.UUID); err != nil {
				return err
			}
			return writeExecutionOK(cmd, flags, "group.join", newExecutionOutput("group", "join", executionTargetFromFlags(flags), map[string]any{
				"to": executionMemberFields(dest),
			}, map[string]any{
				"joiner": executionMemberFields(joiner),
				"to":     executionMemberFields(dest),
			}), map[string]any{
				"joiner": joiner,
				"to":     dest,
			})
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Destination speaker name or IP to join")
	return cmd
}

func newGroupUnjoinCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "unjoin",
		Short:        "Leave the current group",
		Long:         "Makes the target speaker (via --name/--ip) become a standalone coordinator.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			member, err := resolveMember(top, flags.Name, flags.IP)
			if err != nil {
				return err
			}
			c := newGroupingClient(member.IP, flags.Timeout)
			if err := c.LeaveGroup(cmd.Context()); err != nil {
				return err
			}
			return writeExecutionOK(cmd, flags, "group.unjoin", newExecutionOutput("group", "unjoin", executionTargetFromFlags(flags), nil, map[string]any{
				"member": executionMemberFields(member),
			}), map[string]any{"member": member})
		},
	}
	return cmd
}

func newGroupSoloCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "solo",
		Short:        "Make this room play by itself",
		Long:         "Ungroups every other visible member of the target speaker's current group, then makes the target a standalone coordinator.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}

			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			target, err := resolveMember(top, flags.Name, flags.IP)
			if err != nil {
				return err
			}
			group, ok := top.GroupForIP(target.IP)
			if !ok {
				return newStateInconsistentError("speaker not found in any group", map[string]any{
					"action":    "group.solo",
					"speaker":   target.Name,
					"speakerIP": target.IP,
				})
			}

			var others []sonos.Member
			for _, m := range group.Members {
				if !m.IsVisible {
					continue
				}
				if m.IP == target.IP {
					continue
				}
				others = append(others, m)
			}
			sort.SliceStable(others, func(i, j int) bool { return others[i].Name < others[j].Name })

			var results []groupOpResult
			var errs []error

			for _, m := range others {
				c := newGroupingClient(m.IP, flags.Timeout)
				if err := c.LeaveGroup(cmd.Context()); err != nil {
					errs = append(errs, fmt.Errorf("%s (%s): %w", m.Name, m.IP, err))
					results = append(results, groupOpResult{Action: "leave", Target: m.Name, IP: m.IP, Error: err.Error()})
					continue
				}
				results = append(results, groupOpResult{Action: "leave", Target: m.Name, IP: m.IP})
			}

			// Finally, ensure the target is standalone (idempotent).
			c := newGroupingClient(target.IP, flags.Timeout)
			if err := c.LeaveGroup(cmd.Context()); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): %w", target.Name, target.IP, err))
				results = append(results, groupOpResult{Action: "leave", Target: target.Name, IP: target.IP, Error: err.Error()})
			} else {
				results = append(results, groupOpResult{Action: "leave", Target: target.Name, IP: target.IP})
			}

			if len(errs) > 0 {
				return newGroupPartialFailureError("group.solo", results, errs, map[string]any{
					"target": target,
					"group":  group,
				})
			}

			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "group.solo", newExecutionOutput("group", "solo", executionTargetFromFlags(flags), nil, map[string]any{
					"target":   executionMemberFields(target),
					"group":    executionGroupFields(group),
					"affected": len(results),
				}), map[string]any{"target": target, "group": group, "results": results})
			}
			return nil
		},
	}
	return cmd
}

type groupOpResult struct {
	Action  string `json:"action"`
	Target  string `json:"target"`
	IP      string `json:"ip"`
	Skipped bool   `json:"skipped"`
	Error   string `json:"error,omitempty"`
}

type groupOpFailure struct {
	Target string `json:"target"`
	IP     string `json:"ip"`
	Error  string `json:"error"`
}

func newGroupPartialFailureError(action string, results []groupOpResult, errs []error, fields map[string]any) error {
	failures := make([]groupOpFailure, 0, len(results))
	attempted := 0
	succeeded := 0
	skipped := 0

	for _, result := range results {
		if result.Skipped {
			skipped++
			continue
		}
		attempted++
		if strings.TrimSpace(result.Error) != "" {
			failures = append(failures, groupOpFailure{
				Target: result.Target,
				IP:     result.IP,
				Error:  result.Error,
			})
			continue
		}
		succeeded++
	}

	failed := len(failures)
	message := fmt.Sprintf("%s partially failed (%d failed, %d succeeded, %d skipped)", action, failed, succeeded, skipped)
	details := mergeErrorDetails(fields, map[string]any{
		"action":    action,
		"attempted": attempted,
		"succeeded": succeeded,
		"failed":    failed,
		"skipped":   skipped,
		"results":   append([]groupOpResult(nil), results...),
		"failures":  failures,
	})

	return newPartialFailureError(message, errors.Join(errs...), details)
}

func newGroupPartyCmd(flags *rootFlags) *cobra.Command {
	var to string

	cmd := &cobra.Command{
		Use:          "party --to <name-or-ip>",
		Short:        "Join all speakers to a target group",
		Long:         "Makes all visible speakers join the group coordinated by --to.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			to = strings.TrimSpace(to)
			if to == "" {
				return newInvalidArgumentError("--to is required", map[string]any{
					"action": "group.party",
					"flag":   "to",
				})
			}

			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			dest, err := resolveMember(top, to, "")
			if err != nil {
				return err
			}
			destGroup, ok := top.GroupForIP(dest.IP)
			if !ok {
				return newStateInconsistentError("destination speaker not found in any group", map[string]any{
					"action":    "group.party",
					"speaker":   dest.Name,
					"speakerIP": dest.IP,
				})
			}
			if destGroup.Coordinator.UUID == "" {
				return newStateInconsistentError("destination group coordinator UUID missing", map[string]any{
					"action":    "group.party",
					"speaker":   dest.Name,
					"speakerIP": dest.IP,
					"groupID":   destGroup.ID,
				})
			}

			inDest := map[string]struct{}{}
			for _, m := range destGroup.Members {
				inDest[m.IP] = struct{}{}
			}

			var results []groupOpResult
			var errs []error
			for _, g := range top.Groups {
				for _, m := range g.Members {
					if !m.IsVisible {
						continue
					}
					if _, ok := inDest[m.IP]; ok {
						results = append(results, groupOpResult{Action: "join", Target: m.Name, IP: m.IP, Skipped: true})
						continue
					}
					c := newGroupingClient(m.IP, flags.Timeout)
					if err := c.JoinGroup(cmd.Context(), destGroup.Coordinator.UUID); err != nil {
						errs = append(errs, fmt.Errorf("%s (%s): %w", m.Name, m.IP, err))
						results = append(results, groupOpResult{Action: "join", Target: m.Name, IP: m.IP, Error: err.Error()})
						continue
					}
					results = append(results, groupOpResult{Action: "join", Target: m.Name, IP: m.IP})
				}
			}

			if len(errs) > 0 {
				return newGroupPartialFailureError("group.party", results, errs, map[string]any{
					"to": dest,
				})
			}

			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "group.party", newExecutionOutput("group", "party", nil, map[string]any{
					"to": executionMemberFields(dest),
				}, map[string]any{
					"to":       executionMemberFields(dest),
					"affected": len(results),
				}), map[string]any{"to": dest, "results": results})
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Destination speaker name or IP to join")
	return cmd
}

func newGroupDissolveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "dissolve",
		Short:        "Ungroup all members of a group",
		Long:         "Makes every member of the target speaker's group (via --name/--ip) become a standalone coordinator.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}

			tg, err := newTopologyGetter(cmd.Context(), flags.Timeout)
			if err != nil {
				return err
			}
			top, err := tg.GetTopology(cmd.Context())
			if err != nil {
				return err
			}

			member, err := resolveMember(top, flags.Name, flags.IP)
			if err != nil {
				return err
			}
			group, ok := top.GroupForIP(member.IP)
			if !ok {
				return newStateInconsistentError("speaker not found in any group", map[string]any{
					"action":    "group.dissolve",
					"speaker":   member.Name,
					"speakerIP": member.IP,
				})
			}

			var members []sonos.Member
			for _, m := range group.Members {
				if !m.IsVisible {
					continue
				}
				members = append(members, m)
			}
			sort.SliceStable(members, func(i, j int) bool {
				if members[i].IsCoordinator == members[j].IsCoordinator {
					return members[i].Name < members[j].Name
				}
				return !members[i].IsCoordinator && members[j].IsCoordinator
			})

			var results []groupOpResult
			var errs []error
			for _, m := range members {
				c := newGroupingClient(m.IP, flags.Timeout)
				if err := c.LeaveGroup(cmd.Context()); err != nil {
					errs = append(errs, fmt.Errorf("%s (%s): %w", m.Name, m.IP, err))
					results = append(results, groupOpResult{Action: "leave", Target: m.Name, IP: m.IP, Error: err.Error()})
					continue
				}
				results = append(results, groupOpResult{Action: "leave", Target: m.Name, IP: m.IP})
			}

			if len(errs) > 0 {
				return newGroupPartialFailureError("group.dissolve", results, errs, map[string]any{
					"group": group,
				})
			}

			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "group.dissolve", newExecutionOutput("group", "dissolve", executionTargetFromFlags(flags), nil, map[string]any{
					"group":    executionGroupFields(group),
					"affected": len(results),
				}), map[string]any{"group": group, "results": results})
			}

			return nil
		},
	}
	return cmd
}

func resolveMember(top sonos.Topology, name string, ip string) (sonos.Member, error) {
	if strings.TrimSpace(ip) != "" {
		mem, ok := top.FindByIP(strings.TrimSpace(ip))
		if !ok {
			return sonos.Member{}, newTargetNotFoundError("speaker ip not found in topology: "+ip, nil, map[string]any{
				"resolution": "topology",
				"speakerIP":  strings.TrimSpace(ip),
			})
		}
		return mem, nil
	}

	// If name looks like an IP address, treat it as such (for --to).
	name = strings.TrimSpace(name)
	if name != "" && net.ParseIP(name) != nil {
		mem, ok := top.FindByIP(name)
		if !ok {
			return sonos.Member{}, newTargetNotFoundError("speaker ip not found in topology: "+name, nil, map[string]any{
				"resolution": "topology",
				"speakerIP":  name,
			})
		}
		return mem, nil
	}

	mem, ok := top.FindByName(name)
	if !ok {
		// Try case-insensitive match
		for k, v := range top.ByName {
			if strings.EqualFold(k, name) {
				mem = v
				ok = true
				break
			}
		}
	}
	if !ok {
		// Try fuzzy substring match (case-insensitive). If ambiguous, return suggestions.
		needle := strings.ToLower(name)
		if needle != "" {
			matches := make([]string, 0, 4)
			for k := range top.ByName {
				if strings.Contains(strings.ToLower(k), needle) {
					matches = append(matches, k)
				}
			}
			if len(matches) == 1 {
				return top.ByName[matches[0]], nil
			}
			if len(matches) > 1 {
				sort.Strings(matches)
				return sonos.Member{}, newTargetAmbiguousError(fmt.Sprintf("ambiguous speaker name %q; matches: %s", name, strings.Join(matches, ", ")), nil, map[string]any{
					"room":    name,
					"matches": matches,
				})
			}
		}
		return sonos.Member{}, newTargetNotFoundError("speaker name not found in topology: "+name, nil, map[string]any{
			"resolution": "topology",
			"room":       name,
		})
	}
	return mem, nil
}
