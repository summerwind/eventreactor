package main

import (
	"github.com/spf13/cobra"
)

func NewEventCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "events",
		Short: "Manage event resource",
		Long:  "Manage event resource.",
	}

	cmd.AddCommand(NewEventListCommand())
	//cmd.AddCommand(NewEventGetCommand())

	return cmd
}
