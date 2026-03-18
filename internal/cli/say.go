package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

func newSayCmd(flags *rootFlags) *cobra.Command {
	var (
		audioURI   string
		title      string
		radio      bool
		tempVolume int
	)

	cmd := &cobra.Command{
		Use:   "say <text>",
		Short: "Play a TTS clip on Sonos (MVP: requires --audio-uri)",
		Long: "MVP implementation for sonos say. Provide text plus --audio-uri (pre-generated TTS asset), then Sonos plays it immediately. Optional --temp-volume temporarily raises/lowers volume and restores it after start.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.TrimSpace(strings.Join(args, " "))
			if text == "" {
				return fmt.Errorf("text is required")
			}
			audioURI = strings.TrimSpace(audioURI)
			if audioURI == "" {
				return fmt.Errorf("--audio-uri is required in current MVP")
			}

			ctx := cmd.Context()
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}

			restored := false
			var prevVolume int
			if tempVolume >= 0 {
				prevVolume, err = c.GetVolume(ctx)
				if err == nil {
					_ = c.SetVolume(ctx, tempVolume)
					defer func() {
						if !restored {
							_ = c.SetVolume(ctx, prevVolume)
						}
					}()
				}
			}

			meta := ""
			playURI := audioURI
			if radio {
				if strings.TrimSpace(title) == "" {
					title = text
				}
				playURI = sonos.ForceRadioURI(audioURI)
				meta = sonos.BuildRadioMeta(title)
			}

			if err := c.PlayURI(ctx, playURI, meta); err != nil {
				return err
			}

			if tempVolume >= 0 && prevVolume >= 0 {
				_ = c.SetVolume(ctx, prevVolume)
				restored = true
			}

			return writeOK(cmd, flags, "say", map[string]any{
				"coordinatorIP": c.IP,
				"text":          text,
				"audioURI":      audioURI,
				"radio":         radio,
				"tempVolume":    tempVolume,
			})
		},
	}

	cmd.Flags().StringVar(&audioURI, "audio-uri", "", "Publicly reachable TTS audio URL")
	cmd.Flags().StringVar(&title, "title", "", "Display title for radio-style playback")
	cmd.Flags().BoolVar(&radio, "radio", true, "Force radio-style playback for HTTP stream URIs")
	cmd.Flags().IntVar(&tempVolume, "temp-volume", -1, "Temporarily set volume during announcement (0-100), then restore")

	return cmd
}
