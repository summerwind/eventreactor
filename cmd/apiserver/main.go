package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/oklog/ulid"
	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

var (
	VERSION = "0.0.1"
	COMMIT  = "HEAD"
)

var (
	namespace string
	c         client.Client
	entropy   *rand.Rand
)

// logError writes a meesage to stderr.
func logError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// newEventName returns a event name based on ULID.
func newEventName() (string, error) {
	t := ulid.MaxTime() - ulid.Now()

	id, err := ulid.New(t, entropy)
	if err != nil {
		return "", err
	}

	return strings.ToLower(id.String()), nil
}

// eventHandler processes requests to submit an Event
func eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name, err := newEventName()
	if err != nil {
		log.Print(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	eventType := r.Header.Get("CE-EventType")

	ev := &v1alpha1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			Labels: map[string]string{
				"eventreactor.summerwind.github.io/event-type": eventType,
			},
		},
		Spec: v1alpha1.EventSpec{
			CloudEventsVersion: r.Header.Get("CE-CloudEventsVersion"),
			EventID:            r.Header.Get("CE-EventID"),
			EventType:          eventType,
			EventTypeVersion:   r.Header.Get("CE-EventTypeVersion"),
			Source:             r.Header.Get("CE-Source"),
			ContentType:        r.Header.Get("ContentType"),
			Data:               string(b),
		},
	}

	if r.Header.Get("CE-EventTime") != "" {
		t, err := time.Parse(time.RFC3339, r.Header.Get("CE-EventTime"))
		if err == nil {
			ev.Spec.EventTime = metav1.NewTime(t)
		}
	}

	err = c.Create(context.TODO(), ev)
	if err != nil {
		log.Print(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("remote_addr:%s name:%s event_type:%s", r.RemoteAddr, name, eventType)
}

func run(cmd *cobra.Command, args []string) error {
	v, err := cmd.Flags().GetBool("version")
	if err != nil {
		return err
	}

	if v {
		fmt.Printf("%s (%s)\n", VERSION, COMMIT)
		return nil
	}

	namespace, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}
	addr, err := cmd.Flags().GetString("bind-address")
	if err != nil {
		return err
	}
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return err
	}
	certFile, err := cmd.Flags().GetString("tls-cert-file")
	if err != nil {
		return err
	}
	keyFile, err := cmd.Flags().GetString("tls-private-key-file")
	if err != nil {
		return err
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

	t := time.Now()
	entropy = rand.New(rand.NewSource(t.UnixNano()))

	//http.HandleFunc("/api/v1alpha1/events", eventHandler)
	//http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1alpha1/events", eventHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", addr, port),
		Handler: mux,
	}

	go func() {
		if certFile != "" && keyFile != "" {
			server.ListenAndServeTLS(certFile, keyFile)
		} else {
			server.ListenAndServe()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var cmd = &cobra.Command{
		Use:   "apiserver",
		Short: "Event Reactor API server.",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().StringP("namespace", "n", "default", "The namespace to manage resources")
	cmd.Flags().String("bind-address", "0.0.0.0", "The IP address on which to listen")
	cmd.Flags().Int("port", 14380, "The port on which to listen")
	cmd.Flags().String("tls-cert-file", "", "File containing the default x509 Certificate for HTTPS")
	cmd.Flags().String("tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file")
	cmd.Flags().BoolP("version", "v", false, "Display version information and exit")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}