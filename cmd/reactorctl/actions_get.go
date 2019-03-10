package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Action struct {
	Name     string
	Status   string
	Reason   string
	Date     metav1.Time
	Event    string
	Pipeline string
	Steps    []ActionStep
}

type ActionStep struct {
	Name       string
	Reason     string
	ExitCode   int32
	StartedAt  metav1.Time
	FinishedAt metav1.Time
	Log        string
}

func NewActionsGetCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get <action>",
		Short: "Print the information of action",
		Long:  "Print the information of action.",
		RunE:  actionsGetRun,
	}

	return cmd
}

func actionsGetRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("Action name required")
	}

	key := types.NamespacedName{
		Name:      args[0],
		Namespace: namespace,
	}

	action := &v1alpha1.Action{}
	err := c.Get(context.TODO(), key, action)
	if err != nil {
		return err
	}

	cond := action.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	status := "Pending"

	if cond != nil {
		switch cond.Status {
		case corev1.ConditionTrue:
			status = "Succeeded"
			cs := action.CompletionStatus()
			if cs == v1alpha1.CompletionStatusNeutral {
				status = fmt.Sprintf("Succeeded (Neutral)")
			}
		case corev1.ConditionFalse:
			status = "Failed"
			reason := action.FailedReason()
			if reason != "" {
				status = fmt.Sprintf("Failed (%s)", reason)
			}
		case corev1.ConditionUnknown:
			status = "Running"
		}
	}

	a := &Action{
		Name:     action.Name,
		Status:   status,
		Date:     action.ObjectMeta.CreationTimestamp,
		Event:    action.Spec.Event.Name,
		Pipeline: action.Spec.Pipeline.Name,
		Steps:    []ActionStep{},
	}

	for i, s := range action.Status.BuildStatus.StepsCompleted {
		state := action.Status.StepStates[i].Terminated
		if state == nil {
			continue
		}

		as := ActionStep{
			Name:       strings.Replace(s, "build-step-", "", -1),
			Reason:     state.Reason,
			ExitCode:   state.ExitCode,
			StartedAt:  state.StartedAt,
			FinishedAt: state.FinishedAt,
			Log:        action.Status.StepLogs[i],
		}

		a.Steps = append(a.Steps, as)
	}

	out, err := render(actionTemplate, a)
	if err != nil {
		return err
	}

	fmt.Print(out)

	return nil
}

const actionTemplate = `
Name:     {{ .Name }}
Status:   {{ .Status }}
Date:     {{ .Date }}
Event:    {{ .Event }}
Pipeline: {{ .Pipeline }}

{{ range $i, $s := .Steps -}}
[ {{ $s.Name }} ]
Started At: {{ $s.StartedAt }}
Finished At: {{ $s.FinishedAt }}
Exit Code: {{ $s.ExitCode }}
-------------------
{{ $s.Log }}
{{ end -}}
`
