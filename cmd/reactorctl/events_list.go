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

func NewEventsListCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "list [flags]",
		Short: "Print the list of event",
		Long:  "Print the list of event.",
		RunE:  eventsListRun,
	}

	flags := cmd.Flags()
	flags.StringP("type", "t", "", "Filter events by type")
	flags.IntP("limit", "l", 50, "Number of events to show")

	return cmd
}

func eventsListRun(cmd *cobra.Command, args []string) error {
	selector := map[string]string{}

	flags := cmd.Flags()

	eventType, err := flags.GetString("type")
	if err != nil {
		return err
	}
	limit, err := flags.GetInt("limit")
	if err != nil {
		return err
	}

	if eventType != "" {
		selector[v1alpha1.KeyEventType] = eventType
	}

	opts := client.MatchingLabels(selector)
	opts.Namespace = namespace

	eventList := &v1alpha1.EventList{}
	err = c.List(context.TODO(), opts, eventList)
	if err != nil {
		return err
	}

	eventLen := len(eventList.Items)

	if eventLen == 0 {
		fmt.Println("No resources found.")
		return nil
	}

	start := eventLen - limit
	if start < 0 {
		start = 0
	}
	events := eventList.Items[start:]

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(writer, "NAME\tTYPE\tDATE")

	for i, ev := range events {
		if i >= limit {
			break
		}
		date := ev.Spec.Time.Format("2006-01-02 15:04:05")
		fmt.Fprintf(writer, "%s\t%s\t%s\n", ev.Name, ev.Spec.Type, date)
	}
	writer.Flush()

	return nil
}
