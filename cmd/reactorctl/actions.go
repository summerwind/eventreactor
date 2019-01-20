package main

import (
	"github.com/spf13/cobra"
)

func NewActionsCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "actions",
		Short: "Manage action",
		Long:  "Manage action.",
	}

	cmd.AddCommand(NewActionsListCommand())
	cmd.AddCommand(NewActionsGetCommand())
	cmd.AddCommand(NewActionsLogsCommand())

	return cmd
}
