package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/utils"
)

var ipFilterFields = []ipFilterField{
	{flag: "ip", field: "ip", needsValue: true},
	{flag: "country", field: "country", needsValue: true},
	{flag: "region", field: "region", needsValue: true},
	{flag: "city", field: "city", needsValue: true},
	{flag: "all", field: "ALL", needsValue: false},
	{flag: "local", field: "LOCAL", needsValue: false},
	{flag: "remote", field: "REMOTE", needsValue: false},
}

type ipFilter struct{}

type ipFilterField struct {
	flag       string
	field      string
	needsValue bool
}

func (f *ipFilter) new() *cobra.Command {
	c := &cobra.Command{
		Use:   "ip-filter",
		Short: "manage IP filtering rules",
	}
	c.AddCommand(f.newIPFilterList())
	c.AddCommand(f.newIPFilterAdd())
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

func (f *ipFilter) newIPFilterAdd() *cobra.Command {
	return f.newIPFilterMutate("add", "add IP filtering rules", false)
}

func (f *ipFilter) newIPFilterDelete() *cobra.Command {
	return f.newIPFilterMutate("delete", "delete IP filtering rules", true)
}

func (f *ipFilter) newIPFilterMutate(use, short string, delete bool) *cobra.Command {
	c := &cobra.Command{
		Use:   use + " [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			status, field, value, err := f.parseIPFilterFlags(cmd, posArgs)
			if err != nil {
				utils.ExitOnErr(cmd, err)
				return
			}
			if delete {
				utils.ExitOnErr(cmd, args.DeleteIPFilter(cmd, cfg, status, field, value))
				return
			}
			utils.ExitOnErr(cmd, args.UpsertIPFilter(cmd, cfg, status, field, value))
		},
	}
	c.Flags().Bool("allow", false, "allow matching traffic")
	c.Flags().Bool("block", false, "block matching traffic")
	c.Flags().Bool("ip", false, "match by IP address (requires value)")
	c.Flags().Bool("country", false, "match by country (requires value)")
	c.Flags().Bool("region", false, "match by region (requires value)")
	c.Flags().Bool("city", false, "match by city (requires value)")
	c.Flags().Bool("all", false, "match all traffic")
	c.Flags().Bool("local", false, "match local network traffic")
	c.Flags().Bool("remote", false, "match remote network traffic")
	return c
}

func (f *ipFilter) parseIPFilterFlags(cmd *cobra.Command, posArgs []string) (status int16, field, value string, err error) {

	allow, _ := cmd.Flags().GetBool("allow")
	block, _ := cmd.Flags().GetBool("block")
	switch {
	case allow && block:
		return 0, "", "", fmt.Errorf("specify exactly one of --allow or --block")
	case !allow && !block:
		return 0, "", "", fmt.Errorf("specify one of --allow or --block")
	case allow:
		status = 1
	default:
		status = 0
	}

	var selected *ipFilterField
	for i := range ipFilterFields {
		set, _ := cmd.Flags().GetBool(ipFilterFields[i].flag)
		if !set {
			continue
		}
		if selected != nil {
			return 0, "", "", fmt.Errorf("specify exactly one of --ip, --country, --region, --city, --all, --local, --remote")
		}
		spec := ipFilterFields[i]
		selected = &spec
	}
	if selected == nil {
		return 0, "", "", fmt.Errorf("specify one of --ip, --country, --region, --city, --all, --local, --remote")
	}

	field = selected.field
	if selected.needsValue {
		if len(posArgs) != 1 {
			return 0, "", "", fmt.Errorf("value is required for --%s", selected.flag)
		}
		value = posArgs[0]
	} else if len(posArgs) > 0 {
		return 0, "", "", fmt.Errorf("value must not be set for --%s", selected.flag)
	}

	return status, field, value, nil

}
