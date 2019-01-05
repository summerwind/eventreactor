package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		selector[v1alpha1.LabelPipelineName] = pipelineName
	}
	if eventName != "" {
		selector[v1alpha1.LabelEventName] = eventName
	}

	opts := client.MatchingLabels(selector)
	opts.Namespace = namespace
	opts.Raw = &metav1.ListOptions{Limit: int64(limit)}

	actionList := &v1alpha1.ActionList{}
	err = c.List(context.TODO(), opts, actionList)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(writer, "NAME\tSTATUS\tDATE")

	for i, a := range actionList.Items {
		if i >= limit {
			break
		}

		cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
		status := "Pending"
		switch cond.Status {
		case corev1.ConditionTrue:
			status = "Succeeded"
		case corev1.ConditionFalse:
			status = "Failed"
		case corev1.ConditionUnknown:
			status = "Running"
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\n", a.Name, status, a.ObjectMeta.CreationTimestamp)
	}
	writer.Flush()

	return nil
}
