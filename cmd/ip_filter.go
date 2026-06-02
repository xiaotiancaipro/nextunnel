package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/utils"
)

var (
	ipFilterValueRules = []ipFilterRuleSpec{
		{use: "ip", field: "ip", arg: true},
		{use: "country", field: "country", arg: true},
		{use: "region", field: "region", arg: true},
		{use: "city", field: "city", arg: true},
	}
	ipFilterCategoryRules = []ipFilterRuleSpec{
		{use: "all", field: "ALL", arg: false},
		{use: "local", field: "LOCAL", arg: false},
		{use: "remote", field: "REMOTE", arg: false},
	}
)

type ipFilter struct{}

type ipFilterRuleSpec struct {
	use   string
	field string
	arg   bool
}

func (f *ipFilter) new() *cobra.Command {
	c := &cobra.Command{
		Use:   "ip-filter",
		Short: "manage IP filtering rules",
	}
	c.AddCommand(f.newIPFilterList())
	c.AddCommand(f.newIPFilterAction("allow", "add allow rules", 1, false))
	c.AddCommand(f.newIPFilterAction("block", "add block rules", 0, false))
	c.AddCommand(f.newIPFilterDelete())
	return c
}

func (f *ipFilter) newIPFilterList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list current IP filtering rules",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.ListIPFilters(cmd, cfg))
		},
	}
}

func (f *ipFilter) newIPFilterDelete() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "delete IP filtering rules",
	}
	c.AddCommand(f.newIPFilterAction("allow", "delete allow rules", 1, true))
	c.AddCommand(f.newIPFilterAction("block", "delete block rules", 0, true))
	return c
}

func (f *ipFilter) newIPFilterAction(use, short string, status int16, delete bool) *cobra.Command {
	parent := &cobra.Command{
		Use:   use,
		Short: short,
	}
	specs := append(append([]ipFilterRuleSpec{}, ipFilterValueRules...), ipFilterCategoryRules...)
	for i := range specs {
		spec := specs[i]
		c := &cobra.Command{Use: spec.use}
		if spec.arg {
			c.Use = spec.use + " [value]"
			c.Args = cobra.ExactArgs(1)
		} else {
			c.Args = cobra.NoArgs
		}
		c.Run = func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			value := ""
			if spec.arg {
				value = posArgs[0]
			}
			if delete {
				utils.ExitOnErr(cmd, args.DeleteIPFilter(cmd, cfg, status, spec.field, value))
				return
			}
			utils.ExitOnErr(cmd, args.UpsertIPFilter(cmd, cfg, status, spec.field, value))
		}
		parent.AddCommand(c)
	}
	return parent
}
