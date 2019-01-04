package main

import (
	"github.com/spf13/cobra"
)

func NewEventCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "events",
		Short: "Manage event",
		Long:  "Manage event.",
	}

	cmd.AddCommand(NewEventListCommand())

	return cmd
}
