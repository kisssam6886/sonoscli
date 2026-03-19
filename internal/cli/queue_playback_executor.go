package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

const (
	errCodeQueueEmpty           = "ERR_QUEUE_EMPTY"
	errCodeQueueClearFailed     = "ERR_QUEUE_CLEAR_FAILED"
	errCodeQueueEnqueueFailed   = "ERR_QUEUE_ENQUEUE_FAILED"
	errCodeQueuePositionUnknown = "ERR_QUEUE_POSITION_UNKNOWN"
	errCodeQueuePlayPosition    = "ERR_QUEUE_PLAY_POSITION_FAILED"
	errCodePlayCommandFailed    = "ERR_PLAY_COMMAND_FAILED"
	errCodeTransportInfoFailed  = "ERR_TRANSPORT_INFO_FAILED"
	errCodeTransitionStuck      = "ERR_TRANSITION_STUCK"
	defaultTransitionRetryCount = 3
	defaultTransitionRetryDelay = 1200 * time.Millisecond
)

type codedError struct {
	Code    string
	Message string
	Err     error
	Fields  map[string]any
}

func (e *codedError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	errMsg := ""
	if e.Err != nil {
		errMsg = strings.TrimSpace(e.Err.Error())
	}
	switch {
	case e.Code != "" && msg != "" && errMsg != "":
		return e.Code + ": " + msg + ": " + errMsg
	case e.Code != "" && msg != "":
		return e.Code + ": " + msg
	case e.Code != "" && errMsg != "":
		return e.Code + ": " + errMsg
	case e.Code != "":
		return e.Code
	case msg != "" && errMsg != "":
		return msg + ": " + errMsg
	case msg != "":
		return msg
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "unknown error"
	}
}

func (e *codedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newCodedError(code, message string, err error, fields map[string]any) error {
	return &codedError{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
		Err:     err,
		Fields:  cloneFields(fields),
	}
}

func errorCode(err error) string {
	var coded *codedError
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}

func cloneFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		out[k] = v
	}
	return out
}

type queuePlaybackClient interface {
	AddURIToQueue(ctx context.Context, enqueuedURI, enqueuedMeta string, desiredFirstTrackNumber int, enqueueAsNext bool) (int, error)
	ClearQueue(ctx context.Context) error
	GetTransportInfo(ctx context.Context) (sonos.TransportInfo, error)
	Play(ctx context.Context) error
	PlayQueuePosition(ctx context.Context, position int) error
}

type queuePlaybackItem struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	URI      string `json:"uri"`
	Metadata string `json:"metadata,omitempty"`
}

type queuePlaybackOptions struct {
	Source                  string
	ClearQueue              bool
	PlayNow                 bool
	SelectedIndex           int
	TransitionRetryCount    int
	TransitionRetryInterval time.Duration
}

type queuePlaybackResult struct {
	Source           string `json:"source,omitempty"`
	ClearedQueue     bool   `json:"clearedQueue,omitempty"`
	EnqueuedCount    int    `json:"enqueuedCount"`
	QueuedPositions  []int  `json:"queuedPositions,omitempty"`
	PlayNow          bool   `json:"playNow"`
	PlayedPosition   int    `json:"playedPosition,omitempty"`
	RecoveryApplied  bool   `json:"recoveryApplied,omitempty"`
	RecoveryAttempts int    `json:"recoveryAttempts,omitempty"`
	FinalState       string `json:"finalState,omitempty"`
}

func executeQueuePlayback(ctx context.Context, c queuePlaybackClient, items []queuePlaybackItem, opts queuePlaybackOptions) (queuePlaybackResult, error) {
	result := queuePlaybackResult{
		Source:  strings.TrimSpace(opts.Source),
		PlayNow: opts.PlayNow,
	}
	if len(items) == 0 {
		return result, newCodedError(errCodeQueueEmpty, "no queueable tracks", nil, map[string]any{
			"source": result.Source,
		})
	}
	if c == nil {
		return result, errors.New("queue playback client is required")
	}
	if opts.PlayNow && (opts.SelectedIndex < 0 || opts.SelectedIndex >= len(items)) {
		return result, newCodedError(errCodeQueuePositionUnknown, fmt.Sprintf("selected index %d out of range for %d tracks", opts.SelectedIndex, len(items)), nil, map[string]any{
			"selectedIndex": opts.SelectedIndex,
			"items":         len(items),
			"source":        result.Source,
		})
	}

	logger := slog.With(
		"source", result.Source,
		"play_now", opts.PlayNow,
		"items", len(items),
	)

	if opts.ClearQueue {
		if err := c.ClearQueue(ctx); err != nil {
			return result, newCodedError(errCodeQueueClearFailed, "clear queue failed", err, map[string]any{
				"source": result.Source,
			})
		}
		result.ClearedQueue = true
		logger.Debug("queue playback cleared queue")
	}

	result.QueuedPositions = make([]int, 0, len(items))
	for idx, item := range items {
		pos, err := c.AddURIToQueue(ctx, item.URI, item.Metadata, 0, false)
		if err != nil {
			return result, newCodedError(errCodeQueueEnqueueFailed, fmt.Sprintf("enqueue %q failed", queuePlaybackTitle(item)), err, map[string]any{
				"source":    result.Source,
				"index":     idx,
				"itemID":    item.ID,
				"itemTitle": queuePlaybackTitle(item),
			})
		}
		result.QueuedPositions = append(result.QueuedPositions, pos)
		logger.Debug("queue playback enqueued item",
			"index", idx,
			"item_id", strings.TrimSpace(item.ID),
			"title", queuePlaybackTitle(item),
			"position", pos,
		)
	}
	result.EnqueuedCount = len(result.QueuedPositions)
	if !opts.PlayNow {
		return result, nil
	}

	playPos := result.QueuedPositions[opts.SelectedIndex]
	if playPos <= 0 {
		if opts.ClearQueue {
			playPos = opts.SelectedIndex + 1
		} else {
			return result, newCodedError(errCodeQueuePositionUnknown, "queue position missing from AddURIToQueue response", nil, map[string]any{
				"source":        result.Source,
				"selectedIndex": opts.SelectedIndex,
			})
		}
	}
	result.PlayedPosition = playPos

	if err := c.PlayQueuePosition(ctx, playPos); err != nil {
		return result, newCodedError(errCodeQueuePlayPosition, fmt.Sprintf("play queue position %d failed", playPos), err, map[string]any{
			"source":   result.Source,
			"position": playPos,
		})
	}
	logger.Debug("queue playback issued play queue position", "position", playPos)

	retries := opts.TransitionRetryCount
	if retries <= 0 {
		retries = defaultTransitionRetryCount
	}
	delay := opts.TransitionRetryInterval
	if delay <= 0 {
		delay = defaultTransitionRetryDelay
	}

	state, transportErr := readTransportState(ctx, c)
	result.FinalState = state
	if transportErr == nil && state == "PLAYING" {
		logger.Debug("queue playback reached playing state", "position", playPos)
		return result, nil
	}

	for attempt := 1; attempt <= retries; attempt++ {
		result.RecoveryApplied = true
		result.RecoveryAttempts = attempt

		logger.Warn("queue playback recovery retry",
			"attempt", attempt,
			"max_attempts", retries,
			"position", playPos,
			"state", state,
			"transport_error", errString(transportErr),
		)

		if err := sleepWithContext(ctx, delay); err != nil {
			return result, err
		}
		if err := c.Play(ctx); err != nil {
			if attempt == retries {
				return result, newCodedError(errCodePlayCommandFailed, fmt.Sprintf("play recovery failed after %d attempts", attempt), err, map[string]any{
					"source":   result.Source,
					"position": playPos,
				})
			}
			continue
		}

		state, transportErr = readTransportState(ctx, c)
		result.FinalState = state
		if transportErr == nil && state == "PLAYING" {
			logger.Debug("queue playback recovered to playing state",
				"attempt", attempt,
				"position", playPos,
			)
			return result, nil
		}
	}

	if transportErr != nil {
		return result, newCodedError(errCodeTransportInfoFailed, "unable to confirm transport state after playback recovery", transportErr, map[string]any{
			"source":   result.Source,
			"position": playPos,
		})
	}

	return result, newCodedError(errCodeTransitionStuck, fmt.Sprintf("transport failed to reach PLAYING after %d recovery attempts (last state: %s)", retries, fallbackTransportState(result.FinalState)), nil, map[string]any{
		"source":           result.Source,
		"position":         playPos,
		"recoveryAttempts": retries,
		"finalState":       fallbackTransportState(result.FinalState),
	})
}

func queuePlaybackTitle(item queuePlaybackItem) string {
	title := strings.TrimSpace(item.Title)
	if title != "" {
		return title
	}
	id := strings.TrimSpace(item.ID)
	if id != "" {
		return id
	}
	return strings.TrimSpace(item.URI)
}

func readTransportState(ctx context.Context, c queuePlaybackClient) (string, error) {
	ti, err := c.GetTransportInfo(ctx)
	if err != nil {
		return "", err
	}
	return normalizeTransportState(ti.State), nil
}

func normalizeTransportState(state string) string {
	return strings.ToUpper(strings.TrimSpace(state))
}

func fallbackTransportState(state string) string {
	state = normalizeTransportState(state)
	if state == "" {
		return "UNKNOWN"
	}
	return state
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
