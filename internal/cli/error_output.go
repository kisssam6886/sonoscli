package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	errCodeCommandFailed     = "ERR_COMMAND_FAILED"
	errCodeInvalidArgument   = "ERR_INVALID_ARGUMENT"
	errCodePartialFailure    = "ERR_PARTIAL_FAILURE"
	errCodeTargetRequired    = "ERR_TARGET_REQUIRED"
	errCodeTargetNotFound    = "ERR_TARGET_NOT_FOUND"
	errCodeTargetAmbiguous   = "ERR_TARGET_AMBIGUOUS"
	errCodeQueryRequired     = "ERR_QUERY_REQUIRED"
	errCodeNoResults         = "ERR_NO_RESULTS"
	errCodeIndexOutOfRange   = "ERR_INDEX_OUT_OF_RANGE"
	errCodeServiceRequired   = "ERR_SERVICE_REQUIRED"
	errCodeServiceNotFound   = "ERR_SERVICE_NOT_FOUND"
	errCodeServiceAmbiguous  = "ERR_SERVICE_AMBIGUOUS"
	errCodeNotFound          = "ERR_NOT_FOUND"
	errCodeUnsupportedRef    = "ERR_UNSUPPORTED_REF"
	errCodeStateInconsistent = "ERR_STATE_INCONSISTENT"
)

type cliErrorEnvelope struct {
	OK    bool            `json:"ok"`
	Error cliErrorPayload `json:"error"`
}

type cliErrorPayload struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func handleCLIExecuteError(cmd *cobra.Command, flags *rootFlags, args []string, err error) error {
	if err == nil {
		return nil
	}
	if wantsJSONErrorOutput(flags, args) {
		_ = writeJSON(cmd, buildCLIErrorEnvelope(err))
		return err
	}
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	return err
}

func wantsJSONErrorOutput(flags *rootFlags, args []string) bool {
	format := ""
	if flags != nil {
		format = strings.ToLower(strings.TrimSpace(flags.Format))
		if flags.JSON && format == formatPlain {
			format = formatJSON
		}
	}

	argFormat, ok := extractFormatArg(args)
	if ok {
		return strings.EqualFold(argFormat, formatJSON)
	}
	if hasJSONArg(args) {
		return true
	}
	return format == formatJSON
}

func extractFormatArg(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "--format=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--format=")), true
		}
		if arg == "--format" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1]), true
		}
	}
	return "", false
}

func hasJSONArg(args []string) bool {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "--json" || arg == "--json=true" {
			return true
		}
	}
	return false
}

func buildCLIErrorEnvelope(err error) cliErrorEnvelope {
	code, message, details := classifyCLIError(err)
	return cliErrorEnvelope{
		OK: false,
		Error: cliErrorPayload{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

func classifyCLIError(err error) (string, string, map[string]any) {
	if err == nil {
		return errCodeCommandFailed, "command failed", nil
	}

	var coded *codedError
	if errors.As(err, &coded) {
		details := cloneFields(coded.Fields)
		if coded.Err != nil {
			if details == nil {
				details = map[string]any{}
			}
			details["cause"] = coded.Err.Error()
		}
		message := strings.TrimSpace(coded.Message)
		if message == "" {
			message = strings.TrimSpace(err.Error())
		}
		code := strings.TrimSpace(coded.Code)
		if code == "" {
			code = inferCLIErrorCode(message)
		}
		return code, message, details
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "command failed"
	}
	return inferCLIErrorCode(message), message, nil
}

func inferCLIErrorCode(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(msg, "provide --ip or --name"), strings.Contains(msg, "require --ip or --name"):
		return errCodeTargetRequired
	case strings.Contains(msg, "no speakers found"), strings.Contains(msg, "speaker name not found"), strings.Contains(msg, "speaker ip not found"), strings.Contains(msg, "speaker not found for --only"):
		return errCodeTargetNotFound
	case strings.Contains(msg, "ambiguous speaker name"):
		return errCodeTargetAmbiguous
	case strings.Contains(msg, "--service is required"):
		return errCodeServiceRequired
	case strings.Contains(msg, "service not found"):
		return errCodeServiceNotFound
	case strings.Contains(msg, "ambiguous --service"):
		return errCodeServiceAmbiguous
	case strings.Contains(msg, "scene not found"), strings.Contains(msg, "favorite not found"):
		return errCodeNotFound
	case strings.Contains(msg, "partially failed"):
		return errCodePartialFailure
	case strings.Contains(msg, "query is required"):
		return errCodeQueryRequired
	case strings.Contains(msg, "no results"), strings.Contains(msg, "no playable tracks"):
		return errCodeNoResults
	case strings.Contains(msg, "out of range"):
		return errCodeIndexOutOfRange
	case strings.Contains(msg, "not a supported spotify ref"), strings.Contains(msg, "not auto-playable"), strings.Contains(msg, "only spotify refs are supported"), strings.Contains(msg, "not a playable spotify ref"):
		return errCodeUnsupportedRef
	case strings.Contains(msg, "missing after dedupe"):
		return errCodeStateInconsistent
	case strings.Contains(msg, "expected "), strings.Contains(msg, "invalid "), strings.Contains(msg, "unknown key"), strings.Contains(msg, "must be "):
		return errCodeInvalidArgument
	default:
		return errCodeCommandFailed
	}
}

func newTargetRequiredError(flags *rootFlags) error {
	return newCodedError(errCodeTargetRequired, "provide --ip or --name (or run `sonos discover`)", nil, targetErrorDetails(flags))
}

func newTargetActionRequiredError(message string, flags *rootFlags, fields map[string]any) error {
	return newCodedError(errCodeTargetRequired, message, nil, mergeErrorDetails(targetErrorDetails(flags), fields))
}

func newTargetNotFoundError(message string, flags *rootFlags, fields map[string]any) error {
	return newCodedError(errCodeTargetNotFound, message, nil, mergeErrorDetails(targetErrorDetails(flags), fields))
}

func newTargetAmbiguousError(message string, flags *rootFlags, fields map[string]any) error {
	return newCodedError(errCodeTargetAmbiguous, message, nil, mergeErrorDetails(targetErrorDetails(flags), fields))
}

func newInvalidArgumentError(message string, fields map[string]any) error {
	return newCodedError(errCodeInvalidArgument, message, nil, cloneFields(fields))
}

func newPartialFailureError(message string, err error, fields map[string]any) error {
	return newCodedError(errCodePartialFailure, message, err, cloneFields(fields))
}

func newQueryRequiredError(fields map[string]any) error {
	return newCodedError(errCodeQueryRequired, "query is required", nil, cloneFields(fields))
}

func newNoResultsError(message string, fields map[string]any) error {
	if strings.TrimSpace(message) == "" {
		message = "no results"
	}
	return newCodedError(errCodeNoResults, message, nil, cloneFields(fields))
}

func newIndexOutOfRangeError(index, total int, fields map[string]any) error {
	details := mergeErrorDetails(fields, map[string]any{
		"index": index,
		"total": total,
	})
	return newCodedError(errCodeIndexOutOfRange, fmt.Sprintf("--index %d out of range (got %d results)", index, total), nil, details)
}

func newPlayableIndexOutOfRangeError(index, total int, fields map[string]any) error {
	details := mergeErrorDetails(fields, map[string]any{
		"index": index,
		"total": total,
	})
	return newCodedError(errCodeIndexOutOfRange, fmt.Sprintf("--index %d out of range (got %d playable tracks)", index, total), nil, details)
}

func newServiceRequiredError() error {
	return newCodedError(errCodeServiceRequired, "--service is required", nil, nil)
}

func newServiceNotFoundError(name string, available []string) error {
	details := map[string]any{
		"service": strings.TrimSpace(name),
	}
	if len(available) > 0 {
		details["available"] = append([]string(nil), available...)
	}
	return newCodedError(errCodeServiceNotFound, "service not found: "+strings.TrimSpace(name), nil, details)
}

func newServiceAmbiguousError(name string, matches []string) error {
	details := map[string]any{
		"service": strings.TrimSpace(name),
	}
	if len(matches) > 0 {
		details["matches"] = append([]string(nil), matches...)
	}
	return newCodedError(errCodeServiceAmbiguous, fmt.Sprintf("ambiguous --service %q; matches: %s", strings.TrimSpace(name), strings.Join(matches, ", ")), nil, details)
}

func newNotFoundError(message string, fields map[string]any) error {
	return newCodedError(errCodeNotFound, message, nil, cloneFields(fields))
}

func newUnsupportedRefError(message string, fields map[string]any) error {
	return newCodedError(errCodeUnsupportedRef, message, nil, cloneFields(fields))
}

func newStateInconsistentError(message string, fields map[string]any) error {
	return newCodedError(errCodeStateInconsistent, message, nil, cloneFields(fields))
}

func targetErrorDetails(flags *rootFlags) map[string]any {
	if flags == nil {
		return nil
	}
	return map[string]any{
		"ip":      strings.TrimSpace(flags.IP),
		"name":    strings.TrimSpace(flags.Name),
		"timeout": flags.Timeout.String(),
	}
}

func mergeErrorDetails(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := cloneFields(base)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
