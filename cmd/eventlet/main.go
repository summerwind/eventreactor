package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/eventreactor/pkg/apis"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

var (
	namespace string
	addr      string
	port      int
	certFile  string
	keyFile   string

	eventPeriod  time.Duration
	eventCount   int
	actionPeriod time.Duration
	actionCount  int

	c   client.Client
	log logr.Logger
)

var allowedContentType = []string{
	"text/",
	"application/json",
	"application/xml",
	"application/x-www-form-urlencoded",
	"application/cloudevents+json",
}

type CloudEvent struct {
	SpecVersion string          `json:"specversion"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	ID          string          `json:"id"`
	Time        string          `json:"time"`
	SchemaURL   string          `json:"schemeurl"`
	ContentType string          `json:"contenttype"`
	Data        json.RawMessage `json:"data"`
}

// logError writes a meesage to stderr.
func logError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func parseBinaryContent(r *http.Request, event *v1alpha1.Event) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	event.Spec = v1alpha1.EventSpec{
		Type:        r.Header.Get("CE-Type"),
		Source:      r.Header.Get("CE-Source"),
		ID:          r.Header.Get("CE-ID"),
		SchemaURL:   r.Header.Get("CE-SchemaURL"),
		ContentType: r.Header.Get("Content-Type"),
		Data:        string(b),
	}

	if r.Header.Get("CE-Time") != "" {
		t, err := time.Parse(time.RFC3339, r.Header.Get("CE-Time"))
		if err == nil {
			et := metav1.NewTime(t)
			event.Spec.Time = &et
		}
	}

	return nil
}

func parseStructuredContent(r *http.Request, event *v1alpha1.Event) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	ce := &CloudEvent{}
	err = json.Unmarshal(b, ce)
	if err != nil {
		return err
	}
	fmt.Println(ce)

	event.Spec = v1alpha1.EventSpec{
		Type:        ce.Type,
		Source:      ce.Source,
		ID:          ce.ID,
		SchemaURL:   ce.SchemaURL,
		ContentType: ce.ContentType,
		Data:        string(ce.Data),
	}

	if ce.Time != "" {
		t, err := time.Parse(time.RFC3339, ce.Time)
		if err == nil {
			et := metav1.NewTime(t)
			event.Spec.Time = &et
		}
	}

	return nil
}

// eventHandler processes requests to submit an Event
func eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		logError("Header 'Content-Type' missing")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	allowed := false
	for _, prefix := range allowedContentType {
		if strings.HasPrefix(contentType, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		logError(fmt.Sprintf("Invalid content type: %s", contentType))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	event := &v1alpha1.Event{}

	if contentType == "application/cloudevents+json" {
		err := parseStructuredContent(r, event)
		if err != nil {
			logError(fmt.Sprintf("Unable to parse structured content: %v", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else {
		err := parseBinaryContent(r, event)
		if err != nil {
			logError(fmt.Sprintf("Unable to parse binary content: %v", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	event.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      v1alpha1.NewID(),
		Labels: map[string]string{
			v1alpha1.KeyEventType: event.Spec.Type,
		},
	}

	err := c.Create(context.TODO(), event)
	if err != nil {
		logError(fmt.Sprintf("Unable to create event: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Info("Event injected", "name", event.Name, "namespace", namespace, "remote_addr", r.RemoteAddr)
}

func cleanupEvents() error {
	eventList := &v1alpha1.EventList{}
	opts := []client.ListOptionFunc{
		client.InNamespace(namespace),
	}

	err := c.List(context.TODO(), eventList, opts...)
	if err != nil {
		return err
	}

	keepStart := 0
	eventLen := len(actionList.Items)
	if eventLen >= eventCount {
		keepStart = eventLen - eventCount
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
			log.Info("Event deleted", "name", event.Name, "namespace", namespace)
		}
	}

	return nil
}

func cleanupActions() error {
	actionList := &v1alpha1.ActionList{}
	opts := []client.ListOptionFunc{
		client.InNamespace(namespace),
	}

	err := c.List(context.TODO(), actionList, opts...)
	if err != nil {
		return err
	}

	keepStart := 0
	actionLen := len(actionList.Items)
	if actionLen >= actionCount {
		keepStart = actionLen - actionCount
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
			log.Info("Action deleted", "name", action.Name, "namespace", namespace)
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
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

	gcTicker := time.NewTicker(time.Duration(10) * time.Second)
	defer gcTicker.Stop()

	go func() {
		for _ = range gcTicker.C {
			err = cleanupEvents()
			if err != nil {
				logError(fmt.Sprintf("Failed to cleanup events: %v", err))
			}

			err = cleanupActions()
			if err != nil {
				logError(fmt.Sprintf("Failed to cleanup actions: %v", err))
			}
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
	logf.SetLogger(logf.ZapLogger(false))
	log = logf.Log.WithName("eventlet")

	var cmd = &cobra.Command{
		Use:   "eventlet",
		Short: "Namespace Agent for Event Reactor",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "The namespace to create Event resources")
	cmd.Flags().StringVar(&addr, "bind-address", "0.0.0.0", "The IP address on which to listen")
	cmd.Flags().IntVar(&port, "port", 14380, "The port on which to listen")
	cmd.Flags().StringVar(&certFile, "tls-cert-file", "", "File containing the default x509 Certificate for HTTPS")
	cmd.Flags().StringVar(&keyFile, "tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file")
	cmd.Flags().DurationVar(&eventPeriod, "event-retention-period", time.Duration(0), "Retention period of events")
	cmd.Flags().IntVar(&eventCount, "event-retention-count", 100, "The maximum number of events to keep")
	cmd.Flags().DurationVar(&actionPeriod, "action-retention-period", time.Duration(0), "Retention period of actions")
	cmd.Flags().IntVar(&actionCount, "action-retention-count", 100, "The maximum number of actions to keep")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
