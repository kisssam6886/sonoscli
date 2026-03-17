package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

func newModeCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "mode <get|shuffle|shuffle-norepeat|repeat|repeat-one|normal>",
		Short: "Get or set play mode (shuffle/repeat)",
		Long: `Controls playback mode (shuffle/repeat) on the group coordinator.

Modes:
  get              Show current play mode
  shuffle          Enable shuffle with repeat
  shuffle-norepeat Enable shuffle without repeat
  repeat           Enable repeat all without shuffle
  repeat-one       Enable repeat current track
  normal           Disable shuffle and repeat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(args[0])) {
			case "get":
				settings, err := c.GetTransportSettings(ctx)
				if err != nil {
					return err
				}
				if isJSON(flags) {
					return writeJSON(cmd, map[string]any{
						"playMode":      string(settings.PlayMode),
						"coordinatorIP": c.IP,
					})
				}
				if isTSV(flags) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "playMode\t%s\n", settings.PlayMode)
					return nil
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), settings.PlayMode)
				return nil
			case "shuffle":
				return setPlayMode(cmd, flags, c, sonos.PlayModeShuffle, "mode.shuffle")
			case "shuffle-norepeat":
				return setPlayMode(cmd, flags, c, sonos.PlayModeShuffleNoRepeat, "mode.shuffle-norepeat")
			case "repeat":
				return setPlayMode(cmd, flags, c, sonos.PlayModeRepeatAll, "mode.repeat")
			case "repeat-one":
				return setPlayMode(cmd, flags, c, sonos.PlayModeRepeatOne, "mode.repeat-one")
			case "normal":
				return setPlayMode(cmd, flags, c, sonos.PlayModeNormal, "mode.normal")
			default:
				return errors.New("expected get|shuffle|shuffle-norepeat|repeat|repeat-one|normal")
			}
		},
	}
}

func setPlayMode(cmd *cobra.Command, flags *rootFlags, c *sonos.Client, mode sonos.PlayMode, action string) error {
	if err := c.SetPlayMode(cmd.Context(), mode); err != nil {
		return err
	}
	return writeOK(cmd, flags, action, map[string]any{"coordinatorIP": c.IP, "playMode": string(mode)})
}
