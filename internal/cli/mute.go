package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newMuteCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "mute <on|off|toggle|get>",
		Short: "Get or set mute",
		Long:  "Controls RenderingControl mute on the group coordinator.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}

			switch strings.ToLower(args[0]) {
			case "get":
				v, err := c.GetMute(ctx)
				if err != nil {
					return err
				}
				if isJSON(flags) {
					return writeExecutionOK(cmd, flags, "mute.get", newExecutionOutput("transport.mute", "get", executionTargetFromFlags(flags), nil, map[string]any{
						"mute":          v,
						"coordinatorIP": c.IP,
					}), map[string]any{"mute": v, "coordinatorIP": c.IP})
				}
				if isTSV(flags) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "mute\t%v\n", v)
					return nil
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
				return nil
			case "on":
				if err := c.SetMute(ctx, true); err != nil {
					return err
				}
				return writeExecutionOK(cmd, flags, "mute.on", newExecutionOutput("transport.mute", "set", executionTargetFromFlags(flags), map[string]any{
					"mute": true,
				}, map[string]any{
					"mute":          true,
					"coordinatorIP": c.IP,
				}), map[string]any{"coordinatorIP": c.IP, "mute": true})
			case "off":
				if err := c.SetMute(ctx, false); err != nil {
					return err
				}
				return writeExecutionOK(cmd, flags, "mute.off", newExecutionOutput("transport.mute", "set", executionTargetFromFlags(flags), map[string]any{
					"mute": false,
				}, map[string]any{
					"mute":          false,
					"coordinatorIP": c.IP,
				}), map[string]any{"coordinatorIP": c.IP, "mute": false})
			case "toggle":
				v, err := c.GetMute(ctx)
				if err != nil {
					return err
				}
				if err := c.SetMute(ctx, !v); err != nil {
					return err
				}
				return writeExecutionOK(cmd, flags, "mute.toggle", newExecutionOutput("transport.mute", "toggle", executionTargetFromFlags(flags), nil, map[string]any{
					"mute":          !v,
					"coordinatorIP": c.IP,
				}), map[string]any{"coordinatorIP": c.IP, "mute": !v})
			default:
				return newInvalidArgumentError("expected on|off|toggle|get", map[string]any{
					"action": "mute",
					"value":  strings.TrimSpace(args[0]),
				})
			}
		},
	}
}
