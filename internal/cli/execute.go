package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type executeTarget struct {
	Room          string `json:"room,omitempty"`
	Name          string `json:"name,omitempty"`
	IP            string `json:"ip,omitempty"`
	SpeakerIP     string `json:"speakerIP,omitempty"`
	CoordinatorIP string `json:"coordinatorIP,omitempty"`
}

type executeRequestPayload struct {
	Action     string         `json:"action,omitempty"`
	Capability string         `json:"capability,omitempty"`
	Operation  string         `json:"operation,omitempty"`
	Target     executeTarget  `json:"target,omitempty"`
	Request    map[string]any `json:"request,omitempty"`
}

type executeActionAlias struct {
	Capability string
	Operation  string
	Request    map[string]any
}

var executeActionAliases = map[string]executeActionAlias{
	"doctor":                {Capability: "doctor", Operation: "report"},
	"doctor.room":           {Capability: "doctor", Operation: "room"},
	"doctor.ncm":            {Capability: "doctor", Operation: "ncm"},
	"discover":              {Capability: "discover", Operation: "scan"},
	"status":                {Capability: "transport.status", Operation: "get"},
	"play":                  {Capability: "transport", Operation: "play"},
	"pause":                 {Capability: "transport", Operation: "pause"},
	"stop":                  {Capability: "transport", Operation: "stop"},
	"next":                  {Capability: "transport", Operation: "next"},
	"prev":                  {Capability: "transport", Operation: "prev"},
	"mode.get":              {Capability: "transport.mode", Operation: "get"},
	"mode.shuffle":          {Capability: "transport.mode", Operation: "set", Request: map[string]any{"mode": "shuffle"}},
	"mode.shuffle-norepeat": {Capability: "transport.mode", Operation: "set", Request: map[string]any{"mode": "shuffle-norepeat"}},
	"mode.repeat":           {Capability: "transport.mode", Operation: "set", Request: map[string]any{"mode": "repeat"}},
	"mode.repeat-one":       {Capability: "transport.mode", Operation: "set", Request: map[string]any{"mode": "repeat-one"}},
	"mode.normal":           {Capability: "transport.mode", Operation: "set", Request: map[string]any{"mode": "normal"}},
	"volume.get":            {Capability: "transport.volume", Operation: "get"},
	"volume.set":            {Capability: "transport.volume", Operation: "set"},
	"mute.get":              {Capability: "transport.mute", Operation: "get"},
	"mute.on":               {Capability: "transport.mute", Operation: "set", Request: map[string]any{"mute": true}},
	"mute.off":              {Capability: "transport.mute", Operation: "set", Request: map[string]any{"mute": false}},
	"mute.toggle":           {Capability: "transport.mute", Operation: "toggle"},
	"play-uri":              {Capability: "transport.source", Operation: "play-uri"},
	"linein":                {Capability: "transport.source", Operation: "linein"},
	"tv":                    {Capability: "transport.source", Operation: "tv"},
	"music":                 {Capability: "transport.source", Operation: "music"},
	"queue.list":            {Capability: "queue", Operation: "list"},
	"queue.clear":           {Capability: "queue", Operation: "clear"},
	"queue.play":            {Capability: "queue", Operation: "play"},
	"queue.remove":          {Capability: "queue", Operation: "remove"},
	"favorites.list":        {Capability: "favorites", Operation: "list"},
	"favorites.open":        {Capability: "favorites", Operation: "open"},
	"smapi.search":          {Capability: "music.smapi", Operation: "search"},
	"smapi.browse":          {Capability: "music.smapi", Operation: "browse"},
	"auth.smapi.begin":      {Capability: "auth.smapi", Operation: "begin"},
	"auth.smapi.complete":   {Capability: "auth.smapi", Operation: "complete"},
	"open":                  {Capability: "music.spotify", Operation: "open"},
	"enqueue":               {Capability: "music.spotify", Operation: "enqueue"},
	"group.status":          {Capability: "group", Operation: "status"},
	"group.join":            {Capability: "group", Operation: "join"},
	"group.unjoin":          {Capability: "group", Operation: "unjoin"},
	"group.solo":            {Capability: "group", Operation: "solo"},
	"group.party":           {Capability: "group", Operation: "party"},
	"group.dissolve":        {Capability: "group", Operation: "dissolve"},
	"group.volume.get":      {Capability: "group.volume", Operation: "get"},
	"group.volume.set":      {Capability: "group.volume", Operation: "set"},
	"group.mute.get":        {Capability: "group.mute", Operation: "get"},
	"group.mute.on":         {Capability: "group.mute", Operation: "set", Request: map[string]any{"mute": true}},
	"group.mute.off":        {Capability: "group.mute", Operation: "set", Request: map[string]any{"mute": false}},
	"group.mute.toggle":     {Capability: "group.mute", Operation: "toggle"},
	"group.mute.set":        {Capability: "group.mute", Operation: "set"},
	"ncm.play":              {Capability: "music.netease", Operation: "play"},
	"ncm.lucky":             {Capability: "music.netease", Operation: "lucky"},
	"say":                   {Capability: "say", Operation: "announce"},
}

func newExecuteCmd(flags *rootFlags) *cobra.Command {
	var (
		file string
		data string
	)

	cmd := &cobra.Command{
		Use:          "execute",
		Short:        "Execute a machine-readable request",
		Long:         "Accepts a small JSON request and dispatches it to existing Sonos commands. This is a minimal execution-layer entrypoint for agents and automations.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := readExecuteRequest(cmd, file, data)
			if err != nil {
				return err
			}

			runFlags := cloneRootFlags(flags)
			applyExecuteTarget(&runFlags, req.Target)

			targetCmd, targetArgs, err := buildExecuteCommand(&runFlags, req)
			if err != nil {
				return err
			}

			targetCmd.SetContext(cmd.Context())
			targetCmd.SetOut(cmd.OutOrStdout())
			targetCmd.SetErr(cmd.ErrOrStderr())
			targetCmd.SetIn(cmd.InOrStdin())
			targetCmd.SilenceErrors = true
			targetCmd.SilenceUsage = true
			targetCmd.SetArgs(targetArgs)
			return targetCmd.ExecuteContext(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "Path to request JSON (`-` reads stdin)")
	cmd.Flags().StringVar(&data, "data", "", "Inline request JSON")
	cmd.Flags().SortFlags = true
	return cmd
}

func cloneRootFlags(flags *rootFlags) rootFlags {
	if flags == nil {
		return rootFlags{}
	}
	return *flags
}

func readExecuteRequest(cmd *cobra.Command, file, data string) (executeRequestPayload, error) {
	file = strings.TrimSpace(file)
	data = strings.TrimSpace(data)
	if file != "" && data != "" {
		return executeRequestPayload{}, newInvalidArgumentError("use only one of --file or --data", map[string]any{
			"action": "execute",
		})
	}
	if file == "" && data == "" {
		return executeRequestPayload{}, newInvalidArgumentError("provide --file or --data", map[string]any{
			"action": "execute",
		})
	}

	var raw []byte
	var err error
	switch {
	case data != "":
		raw = []byte(data)
	case file == "-":
		raw, err = io.ReadAll(cmd.InOrStdin())
	default:
		raw, err = os.ReadFile(file)
	}
	if err != nil {
		return executeRequestPayload{}, newInvalidArgumentError("failed to read execute request", map[string]any{
			"action": "execute",
			"file":   file,
			"cause":  err.Error(),
		})
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return executeRequestPayload{}, newInvalidArgumentError("execute request is empty", map[string]any{
			"action": "execute",
		})
	}

	var req executeRequestPayload
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&req); err != nil {
		return executeRequestPayload{}, newInvalidArgumentError("invalid execute request JSON", map[string]any{
			"action": "execute",
			"cause":  err.Error(),
		})
	}
	if req.Request == nil {
		req.Request = map[string]any{}
	}
	if err := normalizeExecuteRequest(&req); err != nil {
		return executeRequestPayload{}, err
	}
	return req, nil
}

func normalizeExecuteRequest(req *executeRequestPayload) error {
	if req == nil {
		return newInvalidArgumentError("execute request is required", map[string]any{
			"action": "execute",
		})
	}

	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.Capability = strings.ToLower(strings.TrimSpace(req.Capability))
	req.Operation = strings.ToLower(strings.TrimSpace(req.Operation))

	switch req.Action {
	case "play.spotify":
		if req.Capability == "" {
			req.Capability = "music.spotify"
		} else if req.Capability != "music.spotify" {
			return newInvalidArgumentError("action conflicts with capability", map[string]any{
				"action":      "execute",
				"inputAction": req.Action,
				"capability":  req.Capability,
			})
		}
		if req.Operation == "" {
			enqueueOnly, ok, err := executeBoolField(req.Request, "enqueueOnly")
			if err != nil {
				return err
			}
			if ok && enqueueOnly {
				req.Operation = "enqueue"
			} else {
				req.Operation = "play"
			}
		}
	case "search.spotify":
		if req.Capability == "" {
			req.Capability = "music.spotify"
		} else if req.Capability != "music.spotify" {
			return newInvalidArgumentError("action conflicts with capability", map[string]any{
				"action":      "execute",
				"inputAction": req.Action,
				"capability":  req.Capability,
			})
		}
		if req.Operation == "" {
			selectionAction, _, err := executeStringField(req.Request, "selectionAction")
			if err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(selectionAction)) {
			case "open":
				req.Operation = "search_open"
			case "enqueue":
				req.Operation = "search_enqueue"
			default:
				req.Operation = "search"
			}
		}
	}

	if alias, ok := executeActionAliases[req.Action]; ok {
		if req.Capability == "" {
			req.Capability = alias.Capability
		} else if req.Capability != alias.Capability {
			return newInvalidArgumentError("action conflicts with capability", map[string]any{
				"action":      "execute",
				"inputAction": req.Action,
				"capability":  req.Capability,
			})
		}
		if req.Operation == "" {
			req.Operation = alias.Operation
		} else if req.Operation != alias.Operation {
			return newInvalidArgumentError("action conflicts with operation", map[string]any{
				"action":      "execute",
				"inputAction": req.Action,
				"operation":   req.Operation,
			})
		}
		for key, value := range alias.Request {
			if _, exists := req.Request[key]; !exists {
				req.Request[key] = value
			}
		}
	}

	if req.Capability == "" || req.Operation == "" {
		return newInvalidArgumentError("execute request requires capability and operation", map[string]any{
			"action":     "execute",
			"capability": req.Capability,
			"operation":  req.Operation,
		})
	}
	return nil
}

func applyExecuteTarget(flags *rootFlags, target executeTarget) {
	if flags == nil {
		return
	}
	if room := strings.TrimSpace(firstNonEmpty(target.Room, target.Name)); room != "" {
		flags.Name = room
	}
	if ip := strings.TrimSpace(firstNonEmpty(target.IP, target.CoordinatorIP, target.SpeakerIP)); ip != "" {
		flags.IP = ip
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildExecuteCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Capability {
	case "doctor":
		switch req.Operation {
		case "report":
			return newDoctorCmd(flags), nil, nil
		case "room":
			return newDoctorRoomCmd(flags), nil, nil
		case "ncm":
			args := []string{}
			query, _, err := executeStringField(req.Request, "query")
			if err != nil {
				return nil, nil, err
			}
			if strings.TrimSpace(query) != "" {
				args = append(args, "--query", query)
			}
			category, _, err := executeStringField(req.Request, "category")
			if err != nil {
				return nil, nil, err
			}
			if strings.TrimSpace(category) != "" {
				args = append(args, "--category", category)
			}
			limit, ok, err := executeIntField(req.Request, "limit")
			if err != nil {
				return nil, nil, err
			}
			if ok {
				args = append(args, "--limit", strconv.Itoa(limit))
			}
			return newDoctorNCMCmd(flags), args, nil
		default:
			return nil, nil, unsupportedExecuteOperation(req, "report", "room", "ncm")
		}
	case "discover":
		if req.Operation != "scan" {
			return nil, nil, unsupportedExecuteOperation(req, "scan")
		}
		args := []string{}
		all, ok, err := executeBoolField(req.Request, "all")
		if err != nil {
			return nil, nil, err
		}
		if ok && all {
			args = append(args, "--all")
		}
		return newDiscoverCmd(flags), args, nil
	case "transport":
		return buildExecuteTransportCommand(flags, req)
	case "transport.status":
		if req.Operation != "get" {
			return nil, nil, unsupportedExecuteOperation(req, "get")
		}
		return newStatusCmd(flags), nil, nil
	case "transport.mode":
		return buildExecuteModeCommand(flags, req)
	case "transport.volume":
		return buildExecuteVolumeCommand(flags, req)
	case "transport.mute":
		return buildExecuteMuteCommand(flags, req)
	case "transport.source":
		return buildExecuteSourceCommand(flags, req)
	case "queue":
		return buildExecuteQueueCommand(flags, req)
	case "favorites":
		return buildExecuteFavoritesCommand(flags, req)
	case "group":
		return buildExecuteGroupCommand(flags, req)
	case "group.volume":
		return buildExecuteGroupVolumeCommand(flags, req)
	case "group.mute":
		return buildExecuteGroupMuteCommand(flags, req)
	case "auth.smapi":
		return buildExecuteAuthSMAPICommand(flags, req)
	case "music.netease":
		return buildExecuteNCMCommand(flags, req)
	case "music.smapi":
		return buildExecuteSMAPICommand(flags, req)
	case "music.spotify":
		return buildExecuteSpotifyCommand(flags, req)
	case "say":
		return buildExecuteSayCommand(flags, req)
	default:
		return nil, nil, newInvalidArgumentError("unsupported execute capability: "+req.Capability, map[string]any{
			"action":     "execute",
			"capability": req.Capability,
			"operation":  req.Operation,
		})
	}
}

func buildExecuteTransportCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "play":
		return newPlayCmd(flags), nil, nil
	case "pause":
		return newPauseCmd(flags), nil, nil
	case "stop":
		return newStopCmd(flags), nil, nil
	case "next":
		return newNextCmd(flags), nil, nil
	case "prev":
		return newPrevCmd(flags), nil, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "play", "pause", "stop", "next", "prev")
	}
}

func buildExecuteModeCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "get":
		return newModeCmd(flags), []string{"get"}, nil
	case "set":
		mode, err := executeModeValue(req.Request)
		if err != nil {
			return nil, nil, err
		}
		return newModeCmd(flags), []string{mode}, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "get", "set")
	}
}

func buildExecuteVolumeCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "get":
		return newVolumeCmd(flags), []string{"get"}, nil
	case "set":
		volume, ok, err := executeIntField(req.Request, "volume")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, missingExecuteRequestField(req, "volume")
		}
		return newVolumeCmd(flags), []string{"set", strconv.Itoa(volume)}, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "get", "set")
	}
}

func buildExecuteMuteCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "get":
		return newMuteCmd(flags), []string{"get"}, nil
	case "toggle":
		return newMuteCmd(flags), []string{"toggle"}, nil
	case "set":
		mute, ok, err := executeBoolField(req.Request, "mute")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, missingExecuteRequestField(req, "mute")
		}
		if mute {
			return newMuteCmd(flags), []string{"on"}, nil
		}
		return newMuteCmd(flags), []string{"off"}, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "get", "set", "toggle")
	}
}

func buildExecuteSourceCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "play-uri":
		args := []string{}
		title, _, err := executeStringField(req.Request, "title")
		if err != nil {
			return nil, nil, err
		}
		if title != "" {
			args = append(args, "--title", title)
		}
		radio, ok, err := executeBoolField(req.Request, "radio")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, fmt.Sprintf("--radio=%t", radio))
		}
		uri, ok, err := executeStringField(req.Request, "uri")
		if err != nil {
			return nil, nil, err
		}
		if !ok || uri == "" {
			return nil, nil, missingExecuteRequestField(req, "uri")
		}
		args = append(args, uri)
		return newPlayURICmd(flags), args, nil
	case "linein":
		args := []string{}
		from, ok, err := executeStringField(req.Request, "from")
		if err != nil {
			return nil, nil, err
		}
		if ok && from != "" {
			args = append(args, "--from", from)
		}
		return newLineInCmd(flags), args, nil
	case "tv":
		return newTVCmd(flags), nil, nil
	case "music":
		return newMusicCmd(flags), nil, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "play-uri", "linein", "tv", "music")
	}
}

func buildExecuteQueueCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "list":
		start, ok, err := executeIntField(req.Request, "start")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--start", strconv.Itoa(start))
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
	case "clear":
	case "play", "remove":
		pos, ok, err := executeIntField(req.Request, "pos")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, missingExecuteRequestField(req, "pos")
		}
		args = append(args, strconv.Itoa(pos))
	default:
		return nil, nil, unsupportedExecuteOperation(req, "list", "clear", "play", "remove")
	}
	return newQueueCmd(flags), args, nil
}

func buildExecuteFavoritesCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "list":
		start, ok, err := executeIntField(req.Request, "start")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--start", strconv.Itoa(start))
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
	case "open":
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
			return newFavoritesCmd(flags), args, nil
		}
		title, ok, err := executeStringField(req.Request, "title")
		if err != nil {
			return nil, nil, err
		}
		if !ok || title == "" {
			return nil, nil, newInvalidArgumentError("favorites.open requires request.index or request.title", map[string]any{
				"action":     "execute",
				"capability": req.Capability,
				"operation":  req.Operation,
			})
		}
		args = append(args, title)
	default:
		return nil, nil, unsupportedExecuteOperation(req, "list", "open")
	}
	return newFavoritesCmd(flags), args, nil
}

func buildExecuteAuthSMAPICommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	serviceName, ok, err := executeServiceNameField(req.Request)
	if err != nil {
		return nil, nil, err
	}

	switch req.Operation {
	case "begin":
		args := []string{}
		if ok && serviceName != "" {
			args = append(args, "--service", serviceName)
		}
		return newSMAPIAuthBeginCmd(flags), args, nil
	case "complete":
		args := []string{}
		if ok && serviceName != "" {
			args = append(args, "--service", serviceName)
		}
		code, ok, err := executeStringField(req.Request, "code")
		if err != nil {
			return nil, nil, err
		}
		if !ok || code == "" {
			return nil, nil, missingExecuteRequestField(req, "code")
		}
		args = append(args, "--code", code)
		linkDeviceID, ok, err := executeStringField(req.Request, "linkDeviceID", "linkDeviceId", "linkDevice")
		if err != nil {
			return nil, nil, err
		}
		if ok && linkDeviceID != "" {
			args = append(args, "--link-device-id", linkDeviceID)
		}
		wait, ok, err := executeDurationField(req.Request, "wait")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--wait", wait.String())
		}
		return newSMAPIAuthCompleteCmd(flags), args, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "begin", "complete")
	}
}

func buildExecuteGroupCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "status":
		all, ok, err := executeBoolField(req.Request, "all")
		if err != nil {
			return nil, nil, err
		}
		if ok && all {
			args = append(args, "--all")
		}
	case "join", "party":
		to, ok, err := executeMemberRefField(req.Request, "to")
		if err != nil {
			return nil, nil, err
		}
		if !ok || to == "" {
			return nil, nil, missingExecuteRequestField(req, "to")
		}
		args = append(args, "--to", to)
	case "unjoin", "solo", "dissolve":
	default:
		return nil, nil, unsupportedExecuteOperation(req, "status", "join", "unjoin", "solo", "party", "dissolve")
	}
	return newGroupCmd(flags), args, nil
}

func buildExecuteGroupVolumeCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "get":
	case "set":
		volume, ok, err := executeIntField(req.Request, "volume")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, missingExecuteRequestField(req, "volume")
		}
		args = append(args, strconv.Itoa(volume))
	default:
		return nil, nil, unsupportedExecuteOperation(req, "get", "set")
	}
	return newGroupVolumeCmd(flags), args, nil
}

func buildExecuteGroupMuteCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "get", "toggle":
	case "set":
		mute, ok, err := executeBoolField(req.Request, "mute")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, missingExecuteRequestField(req, "mute")
		}
		if mute {
			args = []string{"on"}
		} else {
			args = []string{"off"}
		}
	default:
		return nil, nil, unsupportedExecuteOperation(req, "get", "set", "toggle")
	}
	return newGroupMuteCmd(flags), args, nil
}

func buildExecuteNCMCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	args := []string{req.Operation}
	switch req.Operation {
	case "play":
		category, ok, err := executeStringField(req.Request, "category")
		if err != nil {
			return nil, nil, err
		}
		if ok && category != "" {
			args = append(args, "--category", category)
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
		}
	case "lucky":
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
	default:
		return nil, nil, unsupportedExecuteOperation(req, "play", "lucky")
	}
	query, ok, err := executeStringField(req.Request, "query")
	if err != nil {
		return nil, nil, err
	}
	if !ok || query == "" {
		return nil, nil, missingExecuteRequestField(req, "query")
	}
	args = append(args, query)
	return newNCMCmd(flags), args, nil
}

func buildExecuteSMAPICommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	serviceName, ok, err := executeServiceNameField(req.Request)
	if err != nil {
		return nil, nil, err
	}
	args := []string{}
	if ok && serviceName != "" {
		args = append(args, "--service", serviceName)
	}

	switch req.Operation {
	case "search":
		category, ok, err := executeStringField(req.Request, "category")
		if err != nil {
			return nil, nil, err
		}
		if ok && category != "" {
			args = append(args, "--category", category)
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
		open, ok, err := executeBoolField(req.Request, "open")
		if err != nil {
			return nil, nil, err
		}
		enqueue, ok2, err := executeBoolField(req.Request, "enqueue")
		if err != nil {
			return nil, nil, err
		}
		if ok && ok2 && open && enqueue {
			return nil, nil, newInvalidArgumentError("use only one of request.open or request.enqueue", map[string]any{
				"action":     "execute",
				"capability": req.Capability,
				"operation":  req.Operation,
			})
		}
		if open {
			args = append(args, "--open")
		}
		if enqueue {
			args = append(args, "--enqueue")
		}
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
		}
		query, ok, err := executeStringField(req.Request, "query")
		if err != nil {
			return nil, nil, err
		}
		if !ok || query == "" {
			return nil, nil, missingExecuteRequestField(req, "query")
		}
		args = append(args, query)
		return newSMAPISearchCmd(flags), args, nil
	case "browse":
		id, ok, err := executeStringField(req.Request, "id")
		if err != nil {
			return nil, nil, err
		}
		if ok && id != "" {
			args = append(args, "--id", id)
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
		recursive, ok, err := executeBoolField(req.Request, "recursive")
		if err != nil {
			return nil, nil, err
		}
		if ok && recursive {
			args = append(args, "--recursive")
		}
		open, ok, err := executeBoolField(req.Request, "open")
		if err != nil {
			return nil, nil, err
		}
		enqueue, ok2, err := executeBoolField(req.Request, "enqueue")
		if err != nil {
			return nil, nil, err
		}
		if ok && ok2 && open && enqueue {
			return nil, nil, newInvalidArgumentError("use only one of request.open or request.enqueue", map[string]any{
				"action":     "execute",
				"capability": req.Capability,
				"operation":  req.Operation,
			})
		}
		if open {
			args = append(args, "--open")
		}
		if enqueue {
			args = append(args, "--enqueue")
		}
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
		}
		return newSMAPIBrowseCmd(flags), args, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "search", "browse")
	}
}

func buildExecuteSpotifyCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	switch req.Operation {
	case "open":
		args := []string{}
		title, ok, err := executeStringField(req.Request, "title", "titleOverride")
		if err != nil {
			return nil, nil, err
		}
		if ok && title != "" {
			args = append(args, "--title", title)
		}
		asNext, ok, err := executeBoolField(req.Request, "asNext", "next")
		if err != nil {
			return nil, nil, err
		}
		if ok && asNext {
			args = append(args, "--next")
		}
		ref, ok, err := executeStringField(req.Request, "ref", "uri")
		if err != nil {
			return nil, nil, err
		}
		if !ok || ref == "" {
			return nil, nil, missingExecuteRequestField(req, "ref")
		}
		args = append(args, ref)
		return newOpenCmd(flags), args, nil
	case "enqueue":
		if _, ok, _ := executeStringField(req.Request, "ref", "uri"); ok {
			args := []string{}
			title, ok, err := executeStringField(req.Request, "title", "titleOverride")
			if err != nil {
				return nil, nil, err
			}
			if ok && title != "" {
				args = append(args, "--title", title)
			}
			asNext, ok, err := executeBoolField(req.Request, "asNext", "next")
			if err != nil {
				return nil, nil, err
			}
			if ok && asNext {
				args = append(args, "--next")
			}
			ref, _, err := executeStringField(req.Request, "ref", "uri")
			if err != nil {
				return nil, nil, err
			}
			args = append(args, ref)
			return newEnqueueCmd(flags), args, nil
		}
		fallthrough
	case "play":
		args := []string{}
		serviceName, ok, err := executeServiceNameField(req.Request)
		if err != nil {
			return nil, nil, err
		}
		if ok && serviceName != "" {
			args = append(args, "--service", serviceName)
		}
		category, ok, err := executeStringField(req.Request, "category")
		if err != nil {
			return nil, nil, err
		}
		if ok && category != "" {
			args = append(args, "--category", category)
		}
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
		}
		title, ok, err := executeStringField(req.Request, "title", "titleOverride")
		if err != nil {
			return nil, nil, err
		}
		if ok && title != "" {
			args = append(args, "--title", title)
		}
		if req.Operation == "enqueue" {
			args = append(args, "--enqueue")
		}
		query, ok, err := executeStringField(req.Request, "query")
		if err != nil {
			return nil, nil, err
		}
		if !ok || query == "" {
			return nil, nil, missingExecuteRequestField(req, "query")
		}
		args = append(args, query)
		return newPlaySpotifyCmd(flags), args, nil
	case "search", "search_open", "search_enqueue":
		args := []string{}
		searchType, ok, err := executeStringField(req.Request, "type")
		if err != nil {
			return nil, nil, err
		}
		if ok && searchType != "" {
			args = append(args, "--type", searchType)
		}
		limit, ok, err := executeIntField(req.Request, "limit")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--limit", strconv.Itoa(limit))
		}
		market, ok, err := executeStringField(req.Request, "market")
		if err != nil {
			return nil, nil, err
		}
		if ok && market != "" {
			args = append(args, "--market", market)
		}
		clientID, ok, err := executeStringField(req.Request, "clientID", "clientId")
		if err != nil {
			return nil, nil, err
		}
		if ok && clientID != "" {
			args = append(args, "--client-id", clientID)
		}
		clientSecret, ok, err := executeStringField(req.Request, "clientSecret")
		if err != nil {
			return nil, nil, err
		}
		if ok && clientSecret != "" {
			args = append(args, "--client-secret", clientSecret)
		}
		index, ok, err := executeIntField(req.Request, "index")
		if err != nil {
			return nil, nil, err
		}
		if ok {
			args = append(args, "--index", strconv.Itoa(index))
		}
		switch req.Operation {
		case "search_open":
			args = append(args, "--open")
		case "search_enqueue":
			args = append(args, "--enqueue")
		}
		query, ok, err := executeStringField(req.Request, "query")
		if err != nil {
			return nil, nil, err
		}
		if !ok || query == "" {
			return nil, nil, missingExecuteRequestField(req, "query")
		}
		args = append(args, query)
		return newSearchSpotifyCmd(flags), args, nil
	default:
		return nil, nil, unsupportedExecuteOperation(req, "open", "enqueue", "play", "search", "search_open", "search_enqueue")
	}
}

func buildExecuteSayCommand(flags *rootFlags, req executeRequestPayload) (*cobra.Command, []string, error) {
	if req.Operation != "announce" {
		return nil, nil, unsupportedExecuteOperation(req, "announce")
	}

	args := []string{}
	audioURI, ok, err := executeStringField(req.Request, "audioURI", "usedAudioURI")
	if err != nil {
		return nil, nil, err
	}
	if ok && audioURI != "" {
		args = append(args, "--audio-uri", audioURI)
	}
	title, ok, err := executeStringField(req.Request, "title")
	if err != nil {
		return nil, nil, err
	}
	if ok && title != "" {
		args = append(args, "--title", title)
	}
	radio, ok, err := executeBoolField(req.Request, "radio")
	if err != nil {
		return nil, nil, err
	}
	if ok {
		args = append(args, fmt.Sprintf("--radio=%t", radio))
	}
	tempVolume, ok, err := executeIntField(req.Request, "tempVolume")
	if err != nil {
		return nil, nil, err
	}
	if ok {
		args = append(args, "--temp-volume", strconv.Itoa(tempVolume))
	}
	lang, ok, err := executeStringField(req.Request, "lang")
	if err != nil {
		return nil, nil, err
	}
	if ok && lang != "" {
		args = append(args, "--lang", lang)
	}
	voice, ok, err := executeStringField(req.Request, "voice")
	if err != nil {
		return nil, nil, err
	}
	if ok && voice != "" {
		args = append(args, "--voice", voice)
	}
	style, ok, err := executeStringField(req.Request, "style")
	if err != nil {
		return nil, nil, err
	}
	if ok && style != "" {
		args = append(args, "--style", style)
	}
	rate, ok, err := executeIntField(req.Request, "rate")
	if err != nil {
		return nil, nil, err
	}
	if ok {
		args = append(args, "--rate", strconv.Itoa(rate))
	}
	holdSeconds, ok, err := executeIntField(req.Request, "holdSeconds")
	if err != nil {
		return nil, nil, err
	}
	if ok {
		args = append(args, "--hold-seconds", strconv.Itoa(holdSeconds))
	}
	text, ok, err := executeStringField(req.Request, "text")
	if err != nil {
		return nil, nil, err
	}
	if !ok || text == "" {
		return nil, nil, missingExecuteRequestField(req, "text")
	}
	args = append(args, text)
	return newSayCmd(flags), args, nil
}

func executeModeValue(fields map[string]any) (string, error) {
	mode, ok, err := executeStringField(fields, "mode", "playMode")
	if err != nil {
		return "", err
	}
	if !ok || mode == "" {
		return "", newInvalidArgumentError("transport.mode set requires request.mode", map[string]any{
			"action": "execute",
		})
	}

	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "normal", "shuffle", "shuffle-norepeat", "repeat", "repeat-one":
		return normalized, nil
	case "shuffle_norepeat":
		return "shuffle-norepeat", nil
	case "repeat_all":
		return "repeat", nil
	case "repeat_one":
		return "repeat-one", nil
	default:
		return normalized, nil
	}
}

func executeStringField(fields map[string]any, keys ...string) (string, bool, error) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || value == nil {
			continue
		}
		str, ok := value.(string)
		if !ok {
			return "", false, newInvalidArgumentError("request."+key+" must be a string", map[string]any{
				"action": "execute",
				"field":  key,
			})
		}
		return strings.TrimSpace(str), true, nil
	}
	return "", false, nil
}

func executeDurationField(fields map[string]any, keys ...string) (time.Duration, bool, error) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			d, err := time.ParseDuration(strings.TrimSpace(v))
			if err != nil {
				return 0, false, newInvalidArgumentError("request."+key+" must be a duration string", map[string]any{
					"action": "execute",
					"field":  key,
					"cause":  err.Error(),
				})
			}
			return d, true, nil
		case int:
			return time.Duration(v) * time.Second, true, nil
		case int64:
			return time.Duration(v) * time.Second, true, nil
		case float64:
			return time.Duration(v * float64(time.Second)), true, nil
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				return 0, false, newInvalidArgumentError("request."+key+" must be a duration string", map[string]any{
					"action": "execute",
					"field":  key,
					"cause":  err.Error(),
				})
			}
			return time.Duration(n) * time.Second, true, nil
		default:
			return 0, false, newInvalidArgumentError("request."+key+" must be a duration string", map[string]any{
				"action": "execute",
				"field":  key,
			})
		}
	}
	return 0, false, nil
}

func executeIntField(fields map[string]any, keys ...string) (int, bool, error) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case int:
			return v, true, nil
		case int32:
			return int(v), true, nil
		case int64:
			return int(v), true, nil
		case float64:
			return int(v), true, nil
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				return 0, false, newInvalidArgumentError("request."+key+" must be an integer", map[string]any{
					"action": "execute",
					"field":  key,
					"cause":  err.Error(),
				})
			}
			return int(n), true, nil
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return 0, false, newInvalidArgumentError("request."+key+" must be an integer", map[string]any{
					"action": "execute",
					"field":  key,
					"cause":  err.Error(),
				})
			}
			return n, true, nil
		default:
			return 0, false, newInvalidArgumentError("request."+key+" must be an integer", map[string]any{
				"action": "execute",
				"field":  key,
			})
		}
	}
	return 0, false, nil
}

func executeBoolField(fields map[string]any, keys ...string) (bool, bool, error) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "1", "on", "yes":
				return true, true, nil
			case "false", "0", "off", "no":
				return false, true, nil
			}
			return false, false, newInvalidArgumentError("request."+key+" must be a boolean", map[string]any{
				"action": "execute",
				"field":  key,
			})
		default:
			return false, false, newInvalidArgumentError("request."+key+" must be a boolean", map[string]any{
				"action": "execute",
				"field":  key,
			})
		}
	}
	return false, false, nil
}

func executeMemberRefField(fields map[string]any, key string) (string, bool, error) {
	value, ok := fields[key]
	if !ok || value == nil {
		return "", false, nil
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true, nil
	case map[string]any:
		for _, candidate := range []string{"name", "room", "ip"} {
			if ref, ok, err := executeStringField(v, candidate); err != nil {
				return "", false, err
			} else if ok && ref != "" {
				return ref, true, nil
			}
		}
		return "", false, newInvalidArgumentError("request."+key+" must include name, room, or ip", map[string]any{
			"action": "execute",
			"field":  key,
		})
	default:
		return "", false, newInvalidArgumentError("request."+key+" must be a string or object", map[string]any{
			"action": "execute",
			"field":  key,
		})
	}
}

func executeServiceNameField(fields map[string]any) (string, bool, error) {
	value, ok := fields["service"]
	if ok && value != nil {
		switch v := value.(type) {
		case string:
			return strings.TrimSpace(v), true, nil
		case map[string]any:
			name, ok, err := executeStringField(v, "name")
			if err != nil {
				return "", false, err
			}
			if !ok || name == "" {
				return "", false, newInvalidArgumentError("request.service must include name", map[string]any{
					"action": "execute",
					"field":  "service",
				})
			}
			return name, true, nil
		default:
			return "", false, newInvalidArgumentError("request.service must be a string or object", map[string]any{
				"action": "execute",
				"field":  "service",
			})
		}
	}
	return executeStringField(fields, "serviceName")
}

func missingExecuteRequestField(req executeRequestPayload, field string) error {
	return newInvalidArgumentError("execute request missing request."+field, map[string]any{
		"action":     "execute",
		"capability": req.Capability,
		"operation":  req.Operation,
		"field":      field,
	})
}

func unsupportedExecuteOperation(req executeRequestPayload, supported ...string) error {
	details := map[string]any{
		"action":     "execute",
		"capability": req.Capability,
		"operation":  req.Operation,
	}
	if len(supported) > 0 {
		details["supportedOperations"] = append([]string(nil), supported...)
	}
	return newInvalidArgumentError("unsupported execute operation: "+req.Capability+"/"+req.Operation, details)
}
