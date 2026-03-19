package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/sonos"
)

const (
	sayProviderMacOS = "macos"
	sayProviderAzure = "azure"
)

type saySynthesisOptions struct {
	Provider    string
	Text        string
	Lang        string
	Voice       string
	Style       string
	Rate        int
	HoldSeconds int

	StyleDegree float64
	Role        string

	AzureKey    string
	AzureRegion string
}

type generatedAudioFile struct {
	Filename    string
	ContentType string
	Data        []byte
}

type sayTransportSnapshot struct {
	CurrentURI  string
	CurrentMeta string
	Track       string
	RelTime     string
	State       string
}

var resolveLocalReachableIP = localReachableIPFor

func newSayCmd(flags *rootFlags) *cobra.Command {
	var (
		audioURI    string
		provider    string
		title       string
		radio       bool
		tempVolume  int
		lang        string
		voice       string
		style       string
		role        string
		rate        int
		styleDegree float64
		holdSeconds int
		azureKey    string
		azureRegion string
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
			provider, err := normalizeSayProvider(provider)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := coordinatorClient(ctx, flags)
			if err != nil {
				return err
			}
			snapshot, _ := captureSayTransportSnapshot(ctx, c)

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
				generatedURI, generatedVoice, release, genErr := generateAndServeTTS(ctx, c.IP, saySynthesisOptions{
					Provider:    provider,
					Text:        text,
					Lang:        lang,
					Voice:       voice,
					Style:       style,
					Rate:        rate,
					HoldSeconds: holdSeconds,
					StyleDegree: styleDegree,
					Role:        role,
					AzureKey:    azureKey,
					AzureRegion: azureRegion,
				})
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

			if err := restoreSayTransportSnapshot(ctx, c, snapshot); err != nil {
				return err
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
				"provider":     provider,
				"lang":         lang,
				"voice":        strings.TrimSpace(voice),
				"style":        style,
				"rate":         rate,
				"holdSeconds":  holdSeconds,
				"usedAudioURI": audioURI,
			}
			if styleDegree > 0 {
				request["styleDegree"] = styleDegree
			}
			if strings.TrimSpace(role) != "" {
				request["role"] = strings.TrimSpace(role)
			}
			if hadExplicitAudioURI {
				request["inputMode"] = "audio-uri"
				delete(request, "provider")
			}
			if strings.TrimSpace(title) != "" {
				request["title"] = strings.TrimSpace(title)
			}
			result := map[string]any{
				"audioURI":   audioURI,
				"provider":   provider,
				"voice":      voice,
				"radio":      radio,
				"tempVolume": tempVolume,
			}
			if hadExplicitAudioURI {
				delete(result, "provider")
			}

			return writeExecutionOK(cmd, flags, "say", newExecutionOutput("say", "announce", target, request, result), map[string]any{
				"coordinatorIP": c.IP,
				"text":          text,
				"audioURI":      audioURI,
				"provider":      request["provider"],
				"radio":         radio,
				"tempVolume":    tempVolume,
				"lang":          lang,
				"voice":         voice,
				"style":         style,
			})
		},
	}

	cmd.Flags().StringVar(&audioURI, "audio-uri", "", "Publicly reachable TTS audio URL (skip local generation)")
	cmd.Flags().StringVar(&provider, "provider", sayProviderMacOS, "TTS provider: macos|azure")
	cmd.Flags().StringVar(&title, "title", "", "Display title for radio-style playback")
	cmd.Flags().BoolVar(&radio, "radio", false, "Force radio-style playback (mainly for continuous stream URIs)")
	cmd.Flags().IntVar(&tempVolume, "temp-volume", -1, "Temporarily set volume during announcement (0-100), then restore")
	cmd.Flags().StringVar(&lang, "lang", "zh", "Language profile: zh|yue")
	cmd.Flags().StringVar(&voice, "voice", "", "Voice override (macOS `say` voice name)")
	cmd.Flags().StringVar(&style, "style", "normal", "Speech style hint. macos: normal|warm|cheerful. azure: normal|cheerful|calm|empathetic|sad|customerservice...")
	cmd.Flags().Float64Var(&styleDegree, "style-degree", 0, "Provider-specific style strength (Azure expressive voices)")
	cmd.Flags().StringVar(&role, "role", "", "Provider-specific speaking role (Azure expressive voices)")
	cmd.Flags().IntVar(&rate, "rate", 0, "Speech rate override (words/min for macOS say)")
	cmd.Flags().IntVar(&holdSeconds, "hold-seconds", 20, "How long to keep local audio HTTP alive for Sonos fetch/play")
	cmd.Flags().StringVar(&azureKey, "azure-key", "", "Azure Speech key (or set AZURE_SPEECH_KEY)")
	cmd.Flags().StringVar(&azureRegion, "azure-region", "", "Azure Speech region (or set AZURE_SPEECH_REGION)")

	return cmd
}

func captureSayTransportSnapshot(ctx context.Context, c *sonos.Client) (sayTransportSnapshot, error) {
	if c == nil {
		return sayTransportSnapshot{}, errors.New("sonos client is required")
	}
	media, err := c.GetMediaInfo(ctx)
	if err != nil {
		return sayTransportSnapshot{}, err
	}
	transport, err := c.GetTransportInfo(ctx)
	if err != nil {
		return sayTransportSnapshot{}, err
	}
	position, err := c.GetPositionInfo(ctx)
	if err != nil {
		return sayTransportSnapshot{}, err
	}
	return sayTransportSnapshot{
		CurrentURI:  strings.TrimSpace(media.CurrentURI),
		CurrentMeta: strings.TrimSpace(media.CurrentURIMetaData),
		Track:       strings.TrimSpace(position.Track),
		RelTime:     normalizeRelTime(position.RelTime),
		State:       normalizeTransportState(transport.State),
	}, nil
}

func restoreSayTransportSnapshot(ctx context.Context, c *sonos.Client, snapshot sayTransportSnapshot) error {
	if c == nil {
		return errors.New("sonos client is required")
	}
	if strings.TrimSpace(snapshot.CurrentURI) == "" {
		return nil
	}
	if isQueueSourceURI(snapshot.CurrentURI) {
		return restoreSayQueueSnapshot(ctx, c, snapshot)
	}
	return restoreSayDirectSourceSnapshot(ctx, c, snapshot)
}

func restoreSayQueueSnapshot(ctx context.Context, c *sonos.Client, snapshot sayTransportSnapshot) error {
	trackNo, hasTrack := parseSayTrackNumber(snapshot.Track)
	if saySnapshotShouldResume(snapshot.State) && hasTrack {
		if err := c.PlayQueuePosition(ctx, trackNo); err != nil {
			return err
		}
		if relTime := normalizeRelTime(snapshot.RelTime); relTime != "" && relTime != "0:00:00" {
			if err := c.SeekRelTime(ctx, relTime); err != nil {
				return err
			}
		}
		return recoverSayPlayback(ctx, c)
	}

	if err := c.SetAVTransportURI(ctx, snapshot.CurrentURI, snapshot.CurrentMeta); err != nil {
		return err
	}
	if hasTrack {
		if err := c.SeekTrackNumber(ctx, trackNo); err != nil {
			return err
		}
	}
	if relTime := normalizeRelTime(snapshot.RelTime); relTime != "" && relTime != "0:00:00" {
		if err := c.SeekRelTime(ctx, relTime); err != nil {
			return err
		}
	}
	return nil
}

func restoreSayDirectSourceSnapshot(ctx context.Context, c *sonos.Client, snapshot sayTransportSnapshot) error {
	if err := c.SetAVTransportURI(ctx, snapshot.CurrentURI, snapshot.CurrentMeta); err != nil {
		return err
	}
	if !saySnapshotShouldResume(snapshot.State) {
		return nil
	}
	return c.Play(ctx)
}

func recoverSayPlayback(ctx context.Context, c *sonos.Client) error {
	state, transportErr := readTransportState(ctx, c)
	if transportErr == nil && state == "PLAYING" {
		return nil
	}
	for attempt := 1; attempt <= defaultTransitionRetryCount; attempt++ {
		if err := c.Play(ctx); err != nil && attempt == defaultTransitionRetryCount {
			return err
		}
		state, transportErr = readTransportState(ctx, c)
		if transportErr == nil && state == "PLAYING" {
			return nil
		}
		if err := sleepWithContext(ctx, defaultTransitionRetryDelay); err != nil {
			return err
		}
	}
	if transportErr != nil {
		return transportErr
	}
	return newCodedError(errCodeTransitionStuck, "say restore failed to reach PLAYING", nil, map[string]any{
		"source":     "say.restore",
		"finalState": fallbackTransportState(state),
	})
}

func isQueueSourceURI(uri string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(uri)), "x-rincon-queue:")
}

func saySnapshotShouldResume(state string) bool {
	switch normalizeTransportState(state) {
	case "PLAYING", "TRANSITIONING":
		return true
	default:
		return false
	}
}

func parseSayTrackNumber(track string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(track))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func normalizeRelTime(relTime string) string {
	relTime = strings.TrimSpace(relTime)
	switch relTime {
	case "", "NOT_IMPLEMENTED":
		return ""
	default:
		return relTime
	}
}

func normalizeSayProvider(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", sayProviderMacOS:
		return sayProviderMacOS, nil
	case sayProviderAzure:
		return sayProviderAzure, nil
	case "openai":
		return "", newInvalidArgumentError("provider openai is not implemented yet; use provider macos or azure", map[string]any{
			"action":   "say",
			"provider": "openai",
		})
	default:
		return "", newInvalidArgumentError("unsupported say provider: "+strings.TrimSpace(provider), map[string]any{
			"action":   "say",
			"provider": strings.TrimSpace(provider),
		})
	}
}

func generateAndServeTTS(ctx context.Context, sonosIP string, opts saySynthesisOptions) (string, string, func(), error) {
	provider, err := normalizeSayProvider(opts.Provider)
	if err != nil {
		return "", "", nil, err
	}
	opts.Provider = provider

	switch provider {
	case sayProviderMacOS:
		return generateAndServeLocalTTS(ctx, sonosIP, opts.Text, opts.Lang, opts.Voice, opts.Style, opts.Rate, opts.HoldSeconds)
	case sayProviderAzure:
		return generateAndServeAzureTTS(ctx, sonosIP, opts)
	default:
		return "", "", nil, newInvalidArgumentError("unsupported say provider: "+provider, map[string]any{
			"action":   "say",
			"provider": provider,
		})
	}
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
	uri, release, err := serveGeneratedFile(ctx, sonosIP, outputPath, outputName, "audio/mp4", func() {
		_ = os.RemoveAll(tmpDir)
	})
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, err
	}
	_ = holdSeconds
	return uri, selectedVoice, release, nil
}

func serveGeneratedFile(ctx context.Context, sonosIP, path, filename, contentType string, cleanup func()) (string, func(), error) {
	localIP, err := resolveLocalReachableIP(sonosIP)
	if err != nil {
		return "", nil, err
	}
	ln, err := net.Listen("tcp", localIP+":0")
	if err != nil {
		return "", nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/"+filename, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, path)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	uri := fmt.Sprintf("http://%s/%s", ln.Addr().String(), filename)
	release := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = ln.Close()
		if cleanup != nil {
			cleanup()
		}
	}
	return uri, release, nil
}

func writeGeneratedAudioFile(dir string, file generatedAudioFile) (string, error) {
	if strings.TrimSpace(file.Filename) == "" {
		return "", errors.New("generated audio filename is required")
	}
	if len(file.Data) == 0 {
		return "", errors.New("generated audio is empty")
	}
	path := filepath.Join(dir, file.Filename)
	if err := os.WriteFile(path, file.Data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func sayXMLEscapeText(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
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

func readHTTPBody(resp *http.Response, limit int64) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, limit))
	return strings.TrimSpace(string(body))
}
