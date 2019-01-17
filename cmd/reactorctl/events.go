package main

import (
	"github.com/spf13/cobra"
)

func NewEventsCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "events",
		Short: "Manage event",
		Long:  "Manage event.",
	}

	cmd.AddCommand(NewEventsListCommand())
	cmd.AddCommand(NewEventsPublishCommand())

	return cmd
}
