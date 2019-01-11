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

func NewPipelinesListCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "list [flags]",
		Short: "Print the list of pipeline",
		Long:  "Print the list of pipeline.",
		RunE:  pipelinesListRun,
	}

	flags := cmd.Flags()
	flags.StringP("type", "t", "", "Filter pipeline by event type")

	return cmd
}

func pipelinesListRun(cmd *cobra.Command, args []string) error {
	selector := map[string]string{}

	flags := cmd.Flags()

	eventType, err := flags.GetString("type")
	if err != nil {
		return err
	}

	if eventType != "" {
		selector[v1alpha1.KeyEventType] = eventType
	}

	opts := client.MatchingLabels(selector)
	opts.Namespace = namespace

	pipelineList := &v1alpha1.PipelineList{}
	err = c.List(context.TODO(), opts, pipelineList)
	if err != nil {
		return err
	}

	if len(pipelineList.Items) == 0 {
		fmt.Println("No resources found.")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(writer, "NAME\tTRIGGER")

	for _, p := range pipelineList.Items {
		trigger := ""
		if p.Spec.Trigger.Pipeline != nil {
			trigger = "pipeline"
		} else {
			trigger = fmt.Sprintf("event:%s", p.Spec.Trigger.Event.Type)
		}

		fmt.Fprintf(writer, "%s\t%s\n", p.Name, trigger)
	}
	writer.Flush()

	return nil
}
