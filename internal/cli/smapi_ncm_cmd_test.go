package cli

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/sonos"
)

type fakeNetEaseSMAPIServer struct {
	srv *httptest.Server

	addURIToQueueCalls atomic.Int32
	playCalls          atomic.Int32
	transportCalls     atomic.Int32
}

func newFakeNetEaseSMAPIServer(t *testing.T) *fakeNetEaseSMAPIServer {
	t.Helper()

	fs := &fakeNetEaseSMAPIServer{}

	mux := http.NewServeMux()

	mux.HandleFunc("/xml/device_description.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<root>
  <device>
    <deviceType>urn:schemas-upnp-org:device:ZonePlayer:1</deviceType>
    <manufacturer>Sonos, Inc.</manufacturer>
    <roomName>Living Room</roomName>
    <UDN>uuid:RINCON_LIVING1400</UDN>
  </device>
</root>`))
	})

	mux.HandleFunc("/MusicServices/Control", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPACTION")
		if !strings.Contains(action, "MusicServices:1#ListAvailableServices") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		servicesXML := `<Services SchemaVersion="1">
  <Service Id="165" Name="网易云音乐" Version="1.1" Uri="http://example" SecureUri="` + fs.srv.URL + `/smapi" ContainerType="MService" Capabilities="513">
    <Presentation>
      <PresentationMap Version="2" Uri="` + fs.srv.URL + `/pmap" />
    </Presentation>
  </Service>
</Services>`
		escaped := strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(servicesXML)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:ListAvailableServicesResponse xmlns:u="urn:schemas-upnp-org:service:MusicServices:1">
      <AvailableServiceDescriptorList>` + escaped + `</AvailableServiceDescriptorList>
    </u:ListAvailableServicesResponse>
  </s:Body>
</s:Envelope>`))
	})

	mux.HandleFunc("/DeviceProperties/Control", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPACTION")
		if !strings.Contains(action, "DeviceProperties:1#GetHouseholdID") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:GetHouseholdIDResponse xmlns:u="urn:schemas-upnp-org:service:DeviceProperties:1">
      <CurrentHouseholdID>Sonos_TEST</CurrentHouseholdID>
    </u:GetHouseholdIDResponse>
  </s:Body>
</s:Envelope>`))
	})

	mux.HandleFunc("/SystemProperties/Control", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPACTION")
		if !strings.Contains(action, "SystemProperties:1#GetString") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:GetStringResponse xmlns:u="urn:schemas-upnp-org:service:SystemProperties:1">
      <StringValue>RINCON_DEVICEID</StringValue>
    </u:GetStringResponse>
  </s:Body>
</s:Envelope>`))
	})

	mux.HandleFunc("/MediaRenderer/AVTransport/Control", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "AVTransport:1#AddURIToQueue"):
			fs.addURIToQueueCalls.Add(1)
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:AddURIToQueueResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
      <FirstTrackNumberEnqueued>3</FirstTrackNumberEnqueued>
    </u:AddURIToQueueResponse>
  </s:Body>
</s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#Seek"):
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SeekResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SeekResponse></s:Body></s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#Play"):
			fs.playCalls.Add(1)
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:PlayResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PlayResponse></s:Body></s:Envelope>`))
		case strings.Contains(action, "AVTransport:1#GetTransportInfo"):
			call := fs.transportCalls.Add(1)
			state := "PLAYING"
			if call == 1 {
				state = "TRANSITIONING"
			}
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:GetTransportInfoResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
      <CurrentTransportState>` + state + `</CurrentTransportState>
      <CurrentTransportStatus>OK</CurrentTransportStatus>
      <CurrentSpeed>1</CurrentSpeed>
    </u:GetTransportInfoResponse>
  </s:Body>
</s:Envelope>`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/pmap", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<PresentationMap>
  <Nested>
    <SearchCategories>
      <Category id="tracks" mappedId="search:track"/>
    </SearchCategories>
  </Nested>
</PresentationMap>`))
	})

	mux.HandleFunc("/smapi", func(w http.ResponseWriter, r *http.Request) {
		action := strings.Trim(r.Header.Get("SOAPACTION"), `"`)
		switch action {
		case "http://www.sonos.com/Services/1.1#search":
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <searchResponse xmlns="http://www.sonos.com/Services/1.1">
      <searchResult>
        <index>0</index>
        <count>1</count>
        <total>1</total>
        <mediaMetadata>
          <id>SONG:12345</id>
          <itemType>track</itemType>
          <title>斗破苍穹</title>
          <mimeType>audio/x-sonos-smapi</mimeType>
          <trackMetadata>
            <artist>天蚕土豆</artist>
            <album>有声书</album>
          </trackMetadata>
        </mediaMetadata>
      </searchResult>
    </searchResponse>
  </s:Body>
</s:Envelope>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fs.srv = httptest.NewServer(mux)
	t.Cleanup(fs.srv.Close)
	return fs
}

func TestSMAPISearchCmd_OpenNetEaseSongUsesQueueExecutor(t *testing.T) {
	fs := newFakeNetEaseSMAPIServer(t)
	u, _ := url.Parse(fs.srv.URL)
	port, _ := strconv.Atoi(u.Port())

	oldNew := newSonosClient
	oldStore := newSMAPITokenStore
	t.Cleanup(func() {
		newSonosClient = oldNew
		newSMAPITokenStore = oldStore
	})

	newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) { return &memTokenStore{}, nil }
	newSonosClient = func(ip string, timeout time.Duration) *sonos.Client {
		return &sonos.Client{IP: u.Hostname(), Port: port, HTTP: fs.srv.Client()}
	}

	flags := &rootFlags{IP: u.Hostname(), Timeout: 2 * time.Second, Format: formatJSON}
	out, err := execute(t, newSMAPISearchCmd(flags), "--service", "网易云音乐", "--category", "tracks", "--open", "--index", "1", "斗破苍穹")
	if err != nil {
		t.Fatalf("smapi search --open: %v", err)
	}
	if !strings.Contains(out, "\"playback\"") {
		t.Fatalf("expected playback result in json: %q", out)
	}
	if !strings.Contains(out, "\"capability\": \"music.smapi\"") || !strings.Contains(out, "\"operation\": \"search\"") {
		t.Fatalf("missing execution envelope: %q", out)
	}
	if !strings.Contains(out, "\"finalState\": \"PLAYING\"") {
		t.Fatalf("expected PLAYING final state: %q", out)
	}
	if !strings.Contains(out, "\"recoveryAttempts\": 1") {
		t.Fatalf("expected one recovery attempt: %q", out)
	}
	if fs.addURIToQueueCalls.Load() != 1 {
		t.Fatalf("addURIToQueueCalls = %d, want 1", fs.addURIToQueueCalls.Load())
	}
	if fs.playCalls.Load() != 2 {
		t.Fatalf("playCalls = %d, want 2", fs.playCalls.Load())
	}
}
