package cli

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type memPlayabilityStore struct {
	path string
	file playabilityFile
}

func (m *memPlayabilityStore) Path() string { return m.path }

func (m *memPlayabilityStore) Load() (playabilityFile, error) {
	return normalizePlayabilityFile(m.file), nil
}

func (m *memPlayabilityStore) Save(f playabilityFile) error {
	m.file = normalizePlayabilityFile(f)
	return nil
}

func TestDoctorNCMCmd_JSONReportsRecommendedExactCandidate(t *testing.T) {
	fs := newFakeNetEaseSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	oldPlayability := newPlayabilityStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
		newPlayabilityStore = oldPlayability
	})

	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return &memTokenStore{}, nil }
	newPlayabilityStore = func() (playabilityStore, error) {
		return &memPlayabilityStore{path: "/tmp/playability.json", file: playabilityFile{Version: 1}}, nil
	}
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{IP: u.Hostname(), Timeout: 2 * time.Second, Format: formatJSON}
	out, err := execute(t, newDoctorNCMCmd(flags), "--query", "天蚕土豆 斗破苍穹")
	if err != nil {
		t.Fatalf("doctor ncm: %v", err)
	}
	if !strings.Contains(out, `"action": "doctor.ncm"`) || !strings.Contains(out, `"operation": "ncm"`) {
		t.Fatalf("missing execution envelope: %s", out)
	}
	if !strings.Contains(out, `"searchReady": true`) || !strings.Contains(out, `"rawTrackCount": 1`) {
		t.Fatalf("unexpected readiness output: %s", out)
	}
	if !strings.Contains(out, `"recommendedTitle": "斗破苍穹"`) || !strings.Contains(out, `"recommendedArtist": "天蚕土豆"`) {
		t.Fatalf("unexpected recommended output: %s", out)
	}
	if !strings.Contains(out, `"matchLevel": "exact_nonlive"`) {
		t.Fatalf("unexpected match level output: %s", out)
	}
}

func TestDoctorNCMCmd_JSONWarnsWhenOnlyCandidateIsBlocked(t *testing.T) {
	fs := newFakeNetEaseSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	oldPlayability := newPlayabilityStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
		newPlayabilityStore = oldPlayability
	})

	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return &memTokenStore{}, nil }
	newPlayabilityStore = func() (playabilityStore, error) {
		return &memPlayabilityStore{
			path: "/tmp/playability.json",
			file: playabilityFile{
				Version: 1,
				Entries: []playabilityEntry{
					{
						ServiceID:   "165",
						ServiceName: neteaseServiceName,
						ItemID:      "SONG:12345",
						Title:       "斗破苍穹",
						Artist:      "天蚕土豆",
						Status:      playabilityStatusBlocked,
						Reason:      playabilityReasonTransitionStuck,
					},
				},
			},
		}, nil
	}
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{IP: u.Hostname(), Timeout: 2 * time.Second, Format: formatJSON}
	out, err := execute(t, newDoctorNCMCmd(flags), "--query", "天蚕土豆 斗破苍穹")
	if err != nil {
		t.Fatalf("doctor ncm blocked: %v", err)
	}
	if !strings.Contains(out, `"blockedFiltered": 1`) || !strings.Contains(out, `"candidateCount": 0`) {
		t.Fatalf("unexpected blocked filtering output: %s", out)
	}
	if !strings.Contains(out, `"ok": false`) || !strings.Contains(out, `当前搜索结果里没有可用的网易云候选`) {
		t.Fatalf("expected warning output, got: %s", out)
	}
}

func TestExecuteCmd_DoctorNCMActionAlias(t *testing.T) {
	fs := newFakeNetEaseSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	oldPlayability := newPlayabilityStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
		newPlayabilityStore = oldPlayability
	})

	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return &memTokenStore{}, nil }
	newPlayabilityStore = func() (playabilityStore, error) {
		return &memPlayabilityStore{path: "/tmp/playability.json", file: playabilityFile{Version: 1}}, nil
	}
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
	cmd := newExecuteCmd(flags)

	out, err := execute(t, cmd, "--data", `{"action":"doctor.ncm","target":{"ip":"`+u.Hostname()+`"},"request":{"query":"天蚕土豆 斗破苍穹"}}`)
	if err != nil {
		t.Fatalf("execute doctor.ncm: %v", err)
	}
	if !strings.Contains(out, `"action": "doctor.ncm"`) || !strings.Contains(out, `"recommendedTitle": "斗破苍穹"`) {
		t.Fatalf("unexpected execute output: %s", out)
	}
}
