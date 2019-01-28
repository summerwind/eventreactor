package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func NewEventsPublishCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "publish [flags]",
		Short: "Publish an event",
		Long:  "Publish an event.",
		RunE:  eventsPublishRun,
	}

	flags := cmd.Flags()
	flags.StringP("type", "t", "", "Event type")
	flags.StringP("source", "s", "", "Event source")
	flags.StringP("data", "d", "{}", "Event data")
	flags.StringP("content-type", "c", "application/json", "Content type of event data")
	flags.String("copy", "", "Name of event to be copied")

	return cmd
}

func eventsPublishRun(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	t, err := flags.GetString("type")
	if err != nil {
		return err
	}
	s, err := flags.GetString("source")
	if err != nil {
		return err
	}
	d, err := flags.GetString("data")
	if err != nil {
		return err
	}
	ct, err := flags.GetString("content-type")
	if err != nil {
		return err
	}
	copy, err := flags.GetString("copy")
	if err != nil {
		return err
	}

	es := &v1alpha1.EventSpec{}
	if copy == "" {
		if t == "" {
			return fmt.Errorf("type must be specified")
		}
		if s == "" {
			return fmt.Errorf("source must be specified")
		}

		es.Type = t
		es.Source = s
		es.Data = d
		es.ContentType = ct
	} else {
		event := &v1alpha1.Event{}
		key := types.NamespacedName{
			Name:      copy,
			Namespace: namespace,
		}

		err := c.Get(context.TODO(), key, event)
		if err != nil {
			return err
		}

		es = event.Spec.DeepCopy()
	}

	now := metav1.Now()
	id := uuid.NewUUID()

	es.Time = &now
	es.ID = string(id)

	ev := &v1alpha1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      v1alpha1.NewID(),
			Labels: map[string]string{
				v1alpha1.KeyEventType: es.Type,
			},
		},
		Spec: *es,
	}

	err = c.Create(context.TODO(), ev)
	if err != nil {
		return err
	}

	fmt.Println(ev.Name)

	return nil
}
