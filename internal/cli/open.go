package cli

import (
	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

func writeSpotifyQueueExecutionOK(cmd *cobra.Command, flags *rootFlags, action, coordinatorIP, ref, title string, asNext, playNow bool, pos int) error {
	return writeExecutionOK(cmd, flags, action, newExecutionOutput("music.spotify", action, executionTargetFromFlags(flags), map[string]any{
		"ref":     ref,
		"title":   title,
		"asNext":  asNext,
		"playNow": playNow,
	}, map[string]any{
		"pos":           pos,
		"coordinatorIP": coordinatorIP,
	}), map[string]any{
		"coordinatorIP": coordinatorIP,
		"pos":           pos,
	})
}

func newOpenCmd(flags *rootFlags) *cobra.Command {
	var title string
	var asNext bool

	cmd := &cobra.Command{
		Use:   "open <spotify-uri-or-link>",
		Short: "Enqueue a Spotify item and start playback",
		Long:  "Adds a Spotify item to the Sonos queue using AVTransport.AddURIToQueue, then starts playback on the coordinator.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ref := args[0]
			_, ok := sonos.ParseSpotifyRef(ref)
			if !ok {
				return newUnsupportedRefError("currently only Spotify refs are supported by `open`", map[string]any{
					"action": "open",
					"ref":    ref,
				})
			}
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			pos, err := c.EnqueueSpotify(ctx, ref, sonos.EnqueueOptions{
				Title:   title,
				AsNext:  asNext,
				PlayNow: true,
			})
			if err != nil {
				return err
			}
			return writeSpotifyQueueExecutionOK(cmd, flags, "open", c.IP, ref, title, asNext, true, pos)
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Optional display title for the queued item")
	cmd.Flags().BoolVar(&asNext, "next", false, "Enqueue as next (shuffle mode only)")
	return cmd
}

func newEnqueueCmd(flags *rootFlags) *cobra.Command {
	var title string
	var asNext bool

	cmd := &cobra.Command{
		Use:   "enqueue <spotify-uri-or-link>",
		Short: "Enqueue a Spotify item (does not start playback)",
		Long:  "Adds a Spotify item to the Sonos queue using AVTransport.AddURIToQueue (no Play).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ref := args[0]
			_, ok := sonos.ParseSpotifyRef(ref)
			if !ok {
				return newUnsupportedRefError("currently only Spotify refs are supported by `enqueue`", map[string]any{
					"action": "enqueue",
					"ref":    ref,
				})
			}
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			pos, err := c.EnqueueSpotify(ctx, ref, sonos.EnqueueOptions{
				Title:   title,
				AsNext:  asNext,
				PlayNow: false,
			})
			if err != nil {
				return err
			}
			return writeSpotifyQueueExecutionOK(cmd, flags, "enqueue", c.IP, ref, title, asNext, false, pos)
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Optional display title for the queued item")
	cmd.Flags().BoolVar(&asNext, "next", false, "Enqueue as next (shuffle mode only)")
	return cmd
}
