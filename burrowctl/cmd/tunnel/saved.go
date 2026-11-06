package tunnel

import (
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var savedCmd = &cobra.Command{Use: "saved", Short: "List private local aliases (not a live server snapshot)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	server := viper.GetString("server")
	owner, err := resolveToken(cmd, server)
	if err != nil {
		return err
	}
	items, err := config.ListSaved(server, owner)
	if err != nil {
		return err
	}
	out := clioutput.FromCommand(cmd)
	if out.Structured() {
		return out.Emit("result", "", items)
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", item.Name, item.State, item.URL, item.ObservedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return err
		}
	}
	return nil
}}
