package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/caarlos0/env"
	"github.com/dxas90/cole/configuration"
	"github.com/dxas90/cole/notifier"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/dxas90/cole/dmtimer"
)

const (
	version = "v0.2.6"
)

var (
	ns   = notifier.NotificationSet{}
	conf = configuration.Conf{}
)

func init() {
	// Log as text. Color with tty attached
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
}

func main() {

	versionPtr := flag.Bool("v", false, "Version")
	configFile := flag.String("c", "", "Path to Configuration File")

	// Once all flags are declared, call `flag.Parse()`
	// to execute the command-line parsing.
	flag.Parse()
	if *versionPtr {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Println("Starting application...")

	// if no config file parameter was passed try env vars.
	if *configFile == "" {
		log.Info("Using ENV Vars for configuration")
		conf = configuration.Conf{}
		if err := env.Parse(&conf); err != nil {
			log.Fatal("Unable to parse envs: ", err)
		}
	} else {
		// read from config file
		log.Info("Reading from config file")
		conf = configuration.ReadConfig(*configFile)
	}

	// DEBUG
	// fmt.Printf("%+v", conf)

	// init first timer at launch of service
	// TODO:
	// figure out a way to start another timer after this alert fires.
	// we want this to continue to go off as long as the dead man
	// switch is not being tripped.

	// init notificaiton set
	ns = notifier.NotificationSet{
		Config: conf,
		Timers: dmtimer.DmTimers{},
	}

	// HTTP Handlers
	http.Handle("/healthz", use(healthCheckHandler))
	http.HandleFunc("/ping/", use(ping, withLogging, withTracing))
	http.HandleFunc("/id", use(genID, withLogging, withTracing))
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, version)
	})
	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
