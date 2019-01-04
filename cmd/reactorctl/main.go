package main

import (
	"errors"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis"
)

var (
	namespace string
	c         client.Client
)

func preRun(cmd *cobra.Command, args []string) error {
	var err error

	namespace, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}
	if namespace == "" {
		return errors.New("namespace must not be empty")
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

	return nil
}

func main() {
	var cmd = &cobra.Command{
		Use:               "reactorctl",
		Short:             "reactorctl controls the Event Reactor resource",
		PersistentPreRunE: preRun,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringP("namespace", "n", "default", "The namespace")

	cmd.AddCommand(NewEventsCommand())
	cmd.AddCommand(NewPipelinesCommand())
	cmd.AddCommand(NewActionsCommand())

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
