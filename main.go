package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/util/homedir"
)

func main() {
	var (
		kubeconfig    *string
		refreshperiod *string
		enableCron    *bool
		enableWebhook *bool
	)

	// Kubeconfig flag
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// Boolean flags
	enableCron = flag.Bool("cron", true, "Enable periodic translations refreshes in the backgound")
	enableWebhook = flag.Bool("webhook", false, "Enable mutation webhook endpoint")

	// Refresh duration flag
	refreshperiod = flag.String("period", "2m", "Duration between translations refreshes")
	duration, err := time.ParseDuration(*refreshperiod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "refresh-period flag cannot be parsed: it is not a valid time.Duration expression")
	}

	flag.Parse()

	config := &Config{
		enableCron:    *enableCron,
		enableWebhook: *enableWebhook,
		kubeconfig:    *kubeconfig,
		locoAPIKeys: map[string]string{
			"catalog": os.Getenv("LOCO_API_KEY_CATALOG"),
			//"documents": os.Getenv("LOCO_API_KEY_DOCUMENTS"),
			"emails": os.Getenv("LOCO_API_KEY_EMAILS"),
		},
		tlsCertFile:       os.Getenv("TLS_CERT_FILE"),
		tlsPrivateKeyFile: os.Getenv("TLS_PRIVATE_KEY_FILE"),
		cronPeriod:        duration,
	}

	app := newApp(config)
	err = app.Init()
	if err != nil {
		panic(err.Error())
	}

	app.Run()
}
