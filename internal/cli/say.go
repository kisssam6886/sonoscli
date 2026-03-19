package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

func newSayCmd(flags *rootFlags) *cobra.Command {
	var (
		audioURI    string
		title       string
		radio       bool
		tempVolume  int
		lang        string
		voice       string
		style       string
		rate        int
		holdSeconds int
	)

	cmd := &cobra.Command{
		Use:   "say <text>",
		Short: "Play a TTS announcement on Sonos",
		Long:  "If --audio-uri is provided, play that URI. Otherwise on macOS, generate local TTS via `say` (with zh/yue voice defaults), transcode it to M4A, host it briefly over local HTTP, and play on Sonos.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.TrimSpace(strings.Join(args, " "))
			if text == "" {
				return fmt.Errorf("text is required")
			}
			hadExplicitAudioURI := strings.TrimSpace(audioURI) != ""

			ctx := cmd.Context()
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}

			audioURI = strings.TrimSpace(audioURI)
			lang = strings.ToLower(strings.TrimSpace(lang))
			style = strings.ToLower(strings.TrimSpace(style))
			if lang == "" {
				lang = "zh"
			}
			if style == "" {
				style = "normal"
			}
			if holdSeconds <= 0 {
				holdSeconds = 20
			}

			cleanup := func() {}
			if audioURI == "" {
				generatedURI, generatedVoice, release, genErr := generateAndServeLocalTTS(ctx, c.IP, text, lang, voice, style, rate, holdSeconds)
				if genErr != nil {
					return genErr
				}
				audioURI = generatedURI
				voice = generatedVoice
				cleanup = release
			}
			defer cleanup()

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
			if holdSeconds > 0 {
				t := time.NewTimer(time.Duration(holdSeconds) * time.Second)
				select {
				case <-ctx.Done():
					t.Stop()
				case <-t.C:
				}
			}

			if tempVolume >= 0 && prevVolume >= 0 {
				_ = c.SetVolume(ctx, prevVolume)
				restored = true
			}

			target := map[string]any{
				"coordinatorIP": c.IP,
			}
			if strings.TrimSpace(flags.Name) != "" {
				target["room"] = strings.TrimSpace(flags.Name)
			}
			if strings.TrimSpace(flags.IP) != "" {
				target["ip"] = strings.TrimSpace(flags.IP)
			}
			request := map[string]any{
				"text":         text,
				"inputMode":    "auto-tts",
				"radio":        radio,
				"tempVolume":   tempVolume,
				"lang":         lang,
				"voice":        strings.TrimSpace(voice),
				"style":        style,
				"rate":         rate,
				"holdSeconds":  holdSeconds,
				"usedAudioURI": audioURI,
			}
			if hadExplicitAudioURI {
				request["inputMode"] = "audio-uri"
			}
			if strings.TrimSpace(title) != "" {
				request["title"] = strings.TrimSpace(title)
			}
			result := map[string]any{
				"audioURI":   audioURI,
				"voice":      voice,
				"radio":      radio,
				"tempVolume": tempVolume,
			}

			return writeExecutionOK(cmd, flags, "say", newExecutionOutput("say", "announce", target, request, result), map[string]any{
				"coordinatorIP": c.IP,
				"text":          text,
				"audioURI":      audioURI,
				"radio":         radio,
				"tempVolume":    tempVolume,
				"lang":          lang,
				"voice":         voice,
				"style":         style,
			})
		},
	}

	cmd.Flags().StringVar(&audioURI, "audio-uri", "", "Publicly reachable TTS audio URL (skip local generation)")
	cmd.Flags().StringVar(&title, "title", "", "Display title for radio-style playback")
	cmd.Flags().BoolVar(&radio, "radio", false, "Force radio-style playback (mainly for continuous stream URIs)")
	cmd.Flags().IntVar(&tempVolume, "temp-volume", -1, "Temporarily set volume during announcement (0-100), then restore")
	cmd.Flags().StringVar(&lang, "lang", "zh", "Language profile: zh|yue")
	cmd.Flags().StringVar(&voice, "voice", "", "Voice override (macOS `say` voice name)")
	cmd.Flags().StringVar(&style, "style", "normal", "Speech style hint: normal|warm|cheerful")
	cmd.Flags().IntVar(&rate, "rate", 0, "Speech rate override (words/min for macOS say)")
	cmd.Flags().IntVar(&holdSeconds, "hold-seconds", 20, "How long to keep local audio HTTP alive for Sonos fetch/play")

	return cmd
}

func generateAndServeLocalTTS(ctx context.Context, sonosIP, text, lang, voice, style string, rate, holdSeconds int) (string, string, func(), error) {
	if runtime.GOOS != "darwin" {
		return "", "", nil, errors.New("auto TTS generation currently supports macOS only; pass --audio-uri on other OS")
	}
	selectedVoice := strings.TrimSpace(voice)
	if selectedVoice == "" {
		selectedVoice = defaultVoiceFor(lang)
	}
	if rate <= 0 {
		rate = defaultRateFor(style)
	}

	tmpDir, err := os.MkdirTemp("", "sonos-say-")
	if err != nil {
		return "", "", nil, err
	}
	inputName := "tts.aiff"
	inputPath := filepath.Join(tmpDir, inputName)
	outputName := "tts.m4a"
	outputPath := filepath.Join(tmpDir, outputName)

	cmd := exec.CommandContext(ctx, "say", "-v", selectedVoice, "-r", fmt.Sprintf("%d", rate), "-o", inputPath, text)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, fmt.Errorf("macOS say failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	transcode := exec.CommandContext(ctx, "afconvert", "-f", "m4af", "-d", "aac", inputPath, outputPath)
	out, err = transcode.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, fmt.Errorf("local TTS transcode failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}

	localIP, err := localReachableIPFor(sonosIP)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, err
	}
	ln, err := net.Listen("tcp", localIP+":0")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/"+outputName, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mp4")
		http.ServeFile(w, r, outputPath)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	uri := fmt.Sprintf("http://%s/%s", ln.Addr().String(), outputName)
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = ln.Close()
		_ = os.RemoveAll(tmpDir)
	}
	_ = holdSeconds
	return uri, selectedVoice, cleanup, nil
}

func defaultVoiceFor(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "yue", "cantonese", "zh-yue":
		return "Sinji"
	case "zh", "mandarin", "zh-cn":
		return "Tingting"
	default:
		return "Tingting"
	}
}

func defaultRateFor(style string) int {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "warm":
		return 168
	case "cheerful":
		return 188
	default:
		return 178
	}
}

func localReachableIPFor(targetIP string) (string, error) {
	targetIP = strings.TrimSpace(targetIP)
	if targetIP == "" {
		return "", errors.New("missing target speaker IP")
	}
	conn, err := net.DialTimeout("udp", net.JoinHostPort(targetIP, "1400"), 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("determine local IP for Sonos route failed: %w", err)
	}
	defer conn.Close()
	ua, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || ua.IP == nil {
		return "", errors.New("unable to resolve local route IP")
	}
	return ua.IP.String(), nil
}
