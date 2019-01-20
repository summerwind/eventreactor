package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func NewActionsLogsCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "logs [flags] <action>",
		Short: "Print the logs of action",
		Long:  "Print the logs of action.",
		RunE:  actionsLogsRun,
	}

	flags := cmd.Flags()
	flags.BoolP("follow", "f", false, "Specify if the logs should be streamed")

	return cmd
}

func actionsLogsRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("Action name required")
	}

	flags := cmd.Flags()

	follow, err := flags.GetBool("follow")
	if err != nil {
		return err
	}

	key := types.NamespacedName{
		Name:      args[0],
		Namespace: namespace,
	}

	action := &v1alpha1.Action{}
	err = c.Get(context.TODO(), key, action)
	if err != nil {
		return err
	}

	if action.IsCompleted() || !follow {
		for _, log := range action.Status.StepLogs {
			fmt.Println(log)
		}
		return nil
	}

	build := &buildv1alpha1.Build{}
	err = c.Get(context.TODO(), key, build)
	if err != nil {
		return err
	}

	if build.Status.Cluster == nil {
		return errors.New("Unsupported: the action was performed outside the cluster")
	}

	podKey := types.NamespacedName{
		Name:      build.Status.Cluster.PodName,
		Namespace: build.Status.Cluster.Namespace,
	}

	pod := &corev1.Pod{}
	err = c.Get(context.TODO(), podKey, pod)
	if err != nil {
		return err
	}

	skip := 1 + len(build.Spec.Sources)
	if build.Spec.Source != nil {
		skip = skip + 1
	}

	for i, c := range pod.Spec.InitContainers {
		if i < skip {
			continue
		}

		var (
			readCloser io.ReadCloser
			err        error
		)

		opts := &corev1.PodLogOptions{
			Follow:    true,
			Container: c.Name,
		}

		req := api.CoreV1().Pods(namespace).GetLogs(pod.Name, opts).Timeout(60 * time.Minute)
		for {
			readCloser, err = req.Stream()
			if err != nil {
				// Wait to start container
				if strings.Contains(err.Error(), "waiting to start") {
					time.Sleep(1 * time.Second)
					continue
				}
				return err
			}
			break
		}
		defer readCloser.Close()

		_, err = io.Copy(os.Stdout, readCloser)
		if err != nil && err != io.EOF {
			return err
		}

		fmt.Println("")
	}

	return nil
}
