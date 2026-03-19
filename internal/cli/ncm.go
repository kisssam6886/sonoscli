package cli

import (
	"github.com/spf13/cobra"
)

const neteaseServiceName = "网易云音乐"

func newNCMCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ncm",
		Short: "网易云音乐快捷命令（基于 Sonos SMAPI）",
		Long:  "针对 Sonos 已接入的网易云音乐服务提供中文友好的快捷命令。",
	}
	cmd.AddCommand(newNCMCategoriesCmd(flags))
	cmd.AddCommand(newNCMBrowseCmd(flags))
	cmd.AddCommand(newNCMSearchCmd(flags))
	cmd.AddCommand(newNCMPlayCmd(flags))
	cmd.AddCommand(newNCMLuckyCmd(flags))
	cmd.AddCommand(newNCMAuthCmd(flags))
	return cmd
}

func newNCMCategoriesCmd(flags *rootFlags) *cobra.Command {
	cmd := newSMAPICategoriesCmd(flags)
	cmd.Use = "categories"
	cmd.Short = "列出网易云音乐支持的搜索分类"
	cmd.Args = cobra.NoArgs
	_ = cmd.Flags().Set("service", neteaseServiceName)
	if f := cmd.Flags().Lookup("service"); f != nil {
		f.Hidden = true
	}
	return cmd
}

func newNCMBrowseCmd(flags *rootFlags) *cobra.Command {
	cmd := newSMAPIBrowseCmd(flags)
	cmd.Use = "browse"
	cmd.Short = "浏览网易云音乐容器/目录"
	cmd.Args = cobra.NoArgs
	_ = cmd.Flags().Set("service", neteaseServiceName)
	if f := cmd.Flags().Lookup("service"); f != nil {
		f.Hidden = true
	}
	return cmd
}

func newNCMSearchCmd(flags *rootFlags) *cobra.Command {
	cmd := newSMAPISearchCmd(flags)
	cmd.Use = "search <query>"
	cmd.Short = "搜索网易云音乐"
	_ = cmd.Flags().Set("service", neteaseServiceName)
	if f := cmd.Flags().Lookup("service"); f != nil {
		f.Hidden = true
	}
	return cmd
}

func newNCMAuthCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "网易云音乐授权",
	}
	cmd.AddCommand(newNCMAuthBeginCmd(flags))
	cmd.AddCommand(newNCMAuthCompleteCmd(flags))
	return cmd
}

func newNCMAuthBeginCmd(flags *rootFlags) *cobra.Command {
	cmd := newSMAPIAuthBeginCmd(flags)
	cmd.Use = "begin"
	cmd.Short = "开始网易云音乐授权"
	cmd.Args = cobra.NoArgs
	_ = cmd.Flags().Set("service", neteaseServiceName)
	if f := cmd.Flags().Lookup("service"); f != nil {
		f.Hidden = true
	}
	return cmd
}

func newNCMAuthCompleteCmd(flags *rootFlags) *cobra.Command {
	cmd := newSMAPIAuthCompleteCmd(flags)
	cmd.Use = "complete"
	cmd.Short = "完成网易云音乐授权"
	cmd.Args = cobra.NoArgs
	_ = cmd.Flags().Set("service", neteaseServiceName)
	if f := cmd.Flags().Lookup("service"); f != nil {
		f.Hidden = true
	}
	return cmd
}
