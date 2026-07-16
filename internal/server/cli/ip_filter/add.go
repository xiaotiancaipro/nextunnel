package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
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

type ipFilterField struct {
	flag       string
	field      string
	needsValue bool
}

func NewAddCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "add IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		Run:   addRun,
	}
	setFlags(c)
	return c
}

func addRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	status, field, value, err := parseIPFilterFlags(cmd, args)
	utils.ExitOnErr(cmd, err)
	utils.ExitOnErr(cmd, utils.UpsertIPFilter(cmd, cfg, status, field, value))
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("allow", false, "allow matching traffic")
	cmd.Flags().Bool("block", false, "block matching traffic")
	cmd.Flags().Bool("ip", false, "match by IP address (requires value)")
	cmd.Flags().Bool("country", false, "match by country (requires value)")
	cmd.Flags().Bool("region", false, "match by region (requires value)")
	cmd.Flags().Bool("city", false, "match by city (requires value)")
	cmd.Flags().Bool("all", false, "match all traffic")
	cmd.Flags().Bool("local", false, "match local network traffic")
	cmd.Flags().Bool("remote", false, "match remote network traffic")
}

func parseIPFilterFlags(cmd *cobra.Command, posArgs []string) (status int16, field, value string, err error) {

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
		selected = new(ipFilterFields[i])
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
