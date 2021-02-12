package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Config holds the application configuration
type Config struct {
	enableCron        bool
	enableWebhook     bool
	kubeconfig        string
	locoAPIKeys       map[string]string
	tlsCertFile       string
	tlsPrivateKeyFile string
	cronPeriod        time.Duration
}

// App represents the main application object
type App struct {
	config       *Config
	translations TranslationsProvider
	refresher    *Refresher
}

func newApp(c *Config) *App {
	return &App{config: c}
}

// Init initializes the application
func (app *App) Init() error {
	// create the kubernetes client config
	config, err := app.configFromKubeConfig()
	if err != nil {
		return err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// create the Refresher service
	app.refresher = NewRefresher(clientset)

	// create the Loco clients
	locoClients := CreateLocoClients(app.config.locoAPIKeys)

	// create the Loco translations provider
	app.translations = NewLocoProvider(locoClients)

	return nil
}

// Run the application
func (app *App) Run() {
	// create Prometheus recorder
	recorder := metrics.NewPrometheus(prometheus.DefaultRegisterer)

	// compute translation hashes
	hashes := app.translations.Fetch()

	if app.config.enableCron {
		cron := NewCron(app.config.cronPeriod)
		go cron.Run(func() {
			// make sure we have the latest translations
			app.translations.Fetch()
			// update kubernetes deployments
			app.refresher.Refresh(hashes)
		})
	}

	if app.config.enableWebhook {
		// We need the handler func with all the translations sync logic
		deployHandler := Appsv1ResourceHandler("etsglobal.org", hashes)

		// Create the mutating webhook
		mw := NewMutatingWebhook(deployHandler, recorder)
		// Start the mutating webhook server in a separate goroutine
		go func() {
			err := mw.ListenAndServeTLS(":8443", app.config.tlsCertFile, app.config.tlsPrivateKeyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error serving webhook: %s", err)
				os.Exit(1)
			}
		}()
	}

	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		defer wg.Done()

		// HTTP server
		mux := http.NewServeMux()
		mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		mux.Handle("/metrics", http.Handler(promhttp.Handler()))

		fmt.Println("Listening on port 8080")
		err := http.ListenAndServe(":8080", mux)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error serving HTTP requests: %s", err)
		}
	}()

	// When everythins is up and running, we can finally run the refresh
	app.refresher.Refresh(hashes)

	wg.Wait()
}

func (app App) configFromKubeConfig() (*rest.Config, error) {
	_, err := os.Stat(app.config.kubeconfig)
	if os.IsNotExist(err) {
		// kubeconfig file doesn't exist, try in-cluster config
		return rest.InClusterConfig()
	}

	// use the current context in kubeconfig
	return clientcmd.BuildConfigFromFlags("", app.config.kubeconfig)
}
