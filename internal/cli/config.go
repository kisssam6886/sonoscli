package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/sonoscli/internal/appconfig"
)

var newConfigStore = func() (appconfig.Store, error) { return appconfig.NewDefaultStore() }

func configExecutionResult(cfg appconfig.Config) map[string]any {
	return compactMap(map[string]any{
		"defaultRoom": strings.TrimSpace(cfg.DefaultRoom),
		"format":      strings.TrimSpace(cfg.Format),
	})
}

func newConfigCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage local CLI defaults",
		Long:  "Stores small, local defaults under your user config directory (e.g. ~/.config/sonoscli/config.json).",
	}
	cmd.AddCommand(newConfigGetCmd(flags))
	cmd.AddCommand(newConfigSetCmd(flags))
	cmd.AddCommand(newConfigUnsetCmd(flags))
	cmd.AddCommand(newConfigPathCmd(flags))
	return cmd
}

func newConfigPathCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newConfigStore()
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "config.path", newExecutionOutput("config", "path", nil, nil, map[string]any{
					"path": s.Path(),
				}), map[string]any{"path": s.Path()})
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), s.Path())
			return nil
		},
	}
}

func newConfigGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get current config (or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newConfigStore()
			if err != nil {
				return err
			}
			cfg, err := s.Load()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				if isJSON(flags) {
					return writeExecutionOK(cmd, flags, "config.get", newExecutionOutput("config", "get", nil, nil, configExecutionResult(cfg)), map[string]any{
						"defaultRoom": cfg.DefaultRoom,
						"format":      cfg.Format,
					})
				}
				printConfigPlain(cmd, cfg)
				return nil
			}

			key := strings.TrimSpace(args[0])
			val, ok := getConfigKey(cfg, key)
			if !ok {
				return errors.New("unknown key: " + key)
			}
			if isJSON(flags) {
				return writeExecutionOK(cmd, flags, "config.get", newExecutionOutput("config", "get", nil, map[string]any{
					"key": key,
				}, map[string]any{
					"key":   key,
					"value": val,
				}), map[string]any{key: val})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, val)
			return nil
		},
	}
}

func newConfigSetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newConfigStore()
			if err != nil {
				return err
			}
			cfg, err := s.Load()
			if err != nil {
				return err
			}

			key := strings.TrimSpace(args[0])
			value := args[1]
			cfg, err = setConfigKey(cfg, key, value)
			if err != nil {
				return err
			}
			if err := s.Save(cfg); err != nil {
				return err
			}
			return writeExecutionOK(cmd, flags, "config.set", newExecutionOutput("config", "set", nil, map[string]any{
				"key":   key,
				"value": value,
			}, map[string]any{
				"key":   key,
				"value": value,
			}), map[string]any{"key": key, "value": value})
		},
	}
}

func newConfigUnsetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a config key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newConfigStore()
			if err != nil {
				return err
			}
			cfg, err := s.Load()
			if err != nil {
				return err
			}

			key := strings.TrimSpace(args[0])
			cfg, err = unsetConfigKey(cfg, key)
			if err != nil {
				return err
			}
			if err := s.Save(cfg); err != nil {
				return err
			}
			return writeExecutionOK(cmd, flags, "config.unset", newExecutionOutput("config", "unset", nil, map[string]any{
				"key": key,
			}, map[string]any{
				"key": key,
			}), map[string]any{"key": key})
		},
	}
}

func printConfigPlain(cmd *cobra.Command, cfg appconfig.Config) {
	entries := map[string]string{
		"defaultRoom": cfg.DefaultRoom,
		"format":      cfg.Format,
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", k, entries[k])
	}
}

func getConfigKey(cfg appconfig.Config, key string) (string, bool) {
	switch key {
	case "defaultRoom":
		return cfg.DefaultRoom, true
	case "format":
		return cfg.Format, true
	default:
		return "", false
	}
}

func setConfigKey(cfg appconfig.Config, key, value string) (appconfig.Config, error) {
	switch key {
	case "defaultRoom":
		cfg.DefaultRoom = value
		return cfg, nil
	case "format":
		value = strings.TrimSpace(value)
		switch strings.ToLower(value) {
		case "", "plain", "json", "tsv":
			cfg.Format = value
		default:
			return appconfig.Config{}, errors.New("invalid format (expected plain|json|tsv): " + value)
		}
		return cfg, nil
	default:
		return appconfig.Config{}, errors.New("unknown key: " + key)
	}
}

func unsetConfigKey(cfg appconfig.Config, key string) (appconfig.Config, error) {
	switch key {
	case "defaultRoom":
		cfg.DefaultRoom = ""
		return cfg, nil
	case "format":
		cfg.Format = ""
		return cfg, nil
	default:
		return appconfig.Config{}, errors.New("unknown key: " + key)
	}
}
