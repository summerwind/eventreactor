package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

func NewActionsListCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "list [flags]",
		Short: "Print the list of action",
		Long:  "Print the list of action.",
		RunE:  actionsListRun,
	}

	flags := cmd.Flags()
	flags.StringP("pipeline", "p", "", "Filter actions by pipeline")
	flags.StringP("event", "e", "", "Filter actions by event")
	flags.IntP("limit", "l", 50, "Number of actions to show")

	return cmd
}

func actionsListRun(cmd *cobra.Command, args []string) error {
	selector := map[string]string{}

	flags := cmd.Flags()

	pipelineName, err := flags.GetString("pipeline")
	if err != nil {
		return err
	}
	eventName, err := flags.GetString("event")
	if err != nil {
		return err
	}
	limit, err := flags.GetInt("limit")
	if err != nil {
		return err
	}

	if pipelineName != "" {
		selector[v1alpha1.KeyPipelineName] = pipelineName
	}
	if eventName != "" {
		selector[v1alpha1.KeyEventName] = eventName
	}

	opts := client.MatchingLabels(selector)
	opts.Namespace = namespace

	actionList := &v1alpha1.ActionList{}
	err = c.List(context.TODO(), opts, actionList)
	if err != nil {
		return err
	}

	actionLen := len(actionList.Items)

	if actionLen == 0 {
		fmt.Println("No resources found.")
		return nil
	}

	start := actionLen - limit
	if start < 0 {
		start = 0
	}

	actions := actionList.Items[start:]

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(writer, "NAME\tPIPELINE\tEVENT\tSTATUS\tDATE")

	for i, a := range actions {
		if i >= limit {
			break
		}

		status := "Pending"
		switch a.CompletionStatus() {
		case v1alpha1.CompletionStatusSuccess:
			status = "Succeeded"
		case v1alpha1.CompletionStatusFailure:
			reason := a.FailedReason()
			if reason != "" {
				status = fmt.Sprintf("Failed (%s)", reason)
			} else {
				status = "Failed"
			}
		case v1alpha1.CompletionStatusNeutral:
			status = "Neutral"
		case v1alpha1.CompletionStatusUnknown:
			status = "Running"
		}

		date := a.ObjectMeta.CreationTimestamp.Format("2006-01-02 15:04:05")
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", a.Name, a.Spec.Pipeline.Name, a.Spec.Event.Name, status, date)
	}
	writer.Flush()

	return nil
}
