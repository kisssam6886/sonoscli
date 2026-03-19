package cli

import "testing"

func TestNewNCMSearchCmd_HidesServiceFlagAndDefaultsToNetease(t *testing.T) {
	flags := &rootFlags{Format: formatPlain}
	cmd := newNCMSearchCmd(flags)
	f := cmd.Flags().Lookup("service")
	if f == nil {
		t.Fatalf("service flag missing")
	}
	if !f.Hidden {
		t.Fatalf("expected service flag to be hidden")
	}
	if got := f.Value.String(); got != neteaseServiceName {
		t.Fatalf("service default = %q, want %q", got, neteaseServiceName)
	}
}

func TestIsSMAPIEmptyTokenPair(t *testing.T) {
	if !isSMAPIEmptyTokenPair(assertErr("empty token pair in response")) {
		t.Fatalf("expected true for empty token pair")
	}
	if isSMAPIEmptyTokenPair(nil) {
		t.Fatalf("expected false for nil")
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

func assertErr(msg string) error { return testErr(msg) }
