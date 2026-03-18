package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

const executionEnvelopeVersion = "v1"

type executionOutput struct {
	Capability string
	Operation  string
	Status     string
	Target     map[string]any
	Request    map[string]any
	Result     map[string]any
}

type executionEnvelope struct {
	Version    string         `json:"version"`
	Capability string         `json:"capability"`
	Operation  string         `json:"operation"`
	Status     string         `json:"status,omitempty"`
	Target     map[string]any `json:"target,omitempty"`
	Request    map[string]any `json:"request,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
}

func newExecutionOutput(capability, operation string, target, request, result map[string]any) executionOutput {
	return executionOutput{
		Capability: capability,
		Operation:  operation,
		Status:     "completed",
		Target:     compactMap(cloneFields(target)),
		Request:    compactMap(cloneFields(request)),
		Result:     compactMap(cloneFields(result)),
	}
}

func executionErrorOutput(capability, operation string, request map[string]any) executionOutput {
	return executionOutput{
		Capability: capability,
		Operation:  operation,
		Status:     "error",
		Request:    compactMap(cloneFields(request)),
	}
}

func writeExecutionOK(cmd *cobra.Command, flags *rootFlags, action string, exec executionOutput, extra map[string]any) error {
	if !isJSON(flags) {
		return nil
	}
	out := map[string]any{
		"ok":        true,
		"action":    action,
		"execution": buildExecutionEnvelope(exec),
	}
	for k, v := range extra {
		out[k] = v
	}
	return writeJSON(cmd, out)
}

func executionJSONLine(action string, exec executionOutput, extra map[string]any) map[string]any {
	out := map[string]any{
		"action":    action,
		"execution": buildExecutionEnvelope(exec),
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func buildExecutionEnvelope(exec executionOutput) executionEnvelope {
	status := exec.Status
	if status == "" {
		status = "completed"
	}
	return executionEnvelope{
		Version:    executionEnvelopeVersion,
		Capability: exec.Capability,
		Operation:  exec.Operation,
		Status:     status,
		Target:     compactMap(cloneFields(exec.Target)),
		Request:    compactMap(cloneFields(exec.Request)),
		Result:     compactMap(cloneFields(exec.Result)),
	}
}

func compactMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch vv := v.(type) {
		case nil:
			continue
		case string:
			if vv == "" {
				continue
			}
		case []string:
			if len(vv) == 0 {
				continue
			}
		case []any:
			if len(vv) == 0 {
				continue
			}
		case map[string]any:
			if len(vv) == 0 {
				continue
			}
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func executionTargetFromFlags(flags *rootFlags) map[string]any {
	if flags == nil {
		return nil
	}
	out := map[string]any{}
	if room := compactString(flags.Name); room != "" {
		out["room"] = room
	}
	if ip := compactString(flags.IP); ip != "" {
		out["ip"] = ip
	}
	return compactMap(out)
}

func compactString(s string) string {
	return strings.TrimSpace(s)
}
