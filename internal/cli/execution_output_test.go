package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestWriteExecutionOKIncludesExecutionEnvelope(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	flags := &rootFlags{Format: formatJSON}
	err := writeExecutionOK(cmd, flags, "scene.apply", newExecutionOutput("scene", "apply", map[string]any{
		"room": "客厅",
	}, map[string]any{
		"name": "Movie Night",
	}, map[string]any{
		"name": "Movie Night",
	}), map[string]any{
		"name": "Movie Night",
	})
	if err != nil {
		t.Fatalf("writeExecutionOK: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got["action"] != "scene.apply" {
		t.Fatalf("action = %#v", got["action"])
	}
	exec, ok := got["execution"].(map[string]any)
	if !ok {
		t.Fatalf("missing execution envelope: %#v", got["execution"])
	}
	if exec["version"] != executionEnvelopeVersion {
		t.Fatalf("version = %#v", exec["version"])
	}
	if exec["capability"] != "scene" || exec["operation"] != "apply" {
		t.Fatalf("execution envelope = %#v", exec)
	}
}
