package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	namespace    string
	eventPeriod  time.Duration
	eventCount   int
	actionPeriod time.Duration
	actionCount  int

	c   client.Client
	log logr.Logger
)

func cleanupEvent() error {
	eventList := &v1alpha1.EventList{}
	opts := &client.ListOptions{Namespace: namespace}

	err := c.List(context.TODO(), opts, eventList)
	if err != nil {
		return err
	}

	keepStart := len(eventList.Items)
	if keepStart >= eventCount {
		keepStart -= eventCount
	}

	for i, event := range eventList.Items {
		delete := false

		if eventCount > 0 && keepStart > i {
			delete = true
		}

		age := time.Since(event.ObjectMeta.CreationTimestamp.Time)
		if eventPeriod > 0 && age > eventPeriod {
			delete = true
		}

		if delete {
			err = c.Delete(context.TODO(), &event)
			if err != nil {
				return err
			}
			log.Info("Deleted event", "name", event.Name, "namespace", namespace)
		}
	}

	return nil
}

func cleanupAction() error {
	actionList := &v1alpha1.ActionList{}
	opts := &client.ListOptions{Namespace: namespace}

	err := c.List(context.TODO(), opts, actionList)
	if err != nil {
		return err
	}

	keepStart := len(actionList.Items)
	if keepStart >= actionCount {
		keepStart -= actionCount
	}

	for i, action := range actionList.Items {
		delete := false

		if actionCount > 0 && keepStart > i {
			delete = true
		}

		age := time.Since(action.ObjectMeta.CreationTimestamp.Time)
		if actionPeriod > 0 && age > actionPeriod {
			delete = true
		}

		if delete {
			err = c.Delete(context.TODO(), &action)
			if err != nil {
				return err
			}
			log.Info("Deleted action", "name", action.Name, "namespace", namespace)
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	if namespace == "" {
		return errors.New("Namespace is empty")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return err
	}

	sc := scheme.Scheme
	if err := apis.AddToScheme(sc); err != nil {
		return err
	}

	c, err = client.New(cfg, client.Options{Scheme: sc, Mapper: mapper})
	if err != nil {
		return err
	}

	err = cleanupEvent()
	if err != nil {
		return err
	}

	err = cleanupAction()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log = logf.Log.WithName("event-cleaner")

	var cmd = &cobra.Command{
		Use:   "event-cleaner",
		Short: "The garbage collector for Event Reactor",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "The namespace")
	cmd.Flags().DurationVar(&eventPeriod, "event-retention-period", time.Duration(0), "Retention period of the event")
	cmd.Flags().IntVar(&eventCount, "event-retention-count", 0, "The maximum number of the event to keep")
	cmd.Flags().DurationVar(&actionPeriod, "action-retention-period", time.Duration(0), "Retention period of the action")
	cmd.Flags().IntVar(&actionCount, "action-retention-count", 0, "The maximum number of the action to keep")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
