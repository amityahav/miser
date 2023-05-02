package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"miser"
	"miser/agent"
	"net/http"
)

func main() {
	var path string

	flag.StringVar(&path, "config", "", "path to configuration file")

	// Parse command-line flags
	flag.Parse()

	if path == "" {
		path = "config.yaml"
	}

	// Load configuration file
	cfg, err := readConfig(path)
	if err != nil {
		log.Fatal(err)
	}

	// Init miser
	m, err := agent.NewMiser(cfg)
	if err != nil {
		log.Fatal(err)
	}

	reg := m.GetPromRegistry()
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		log.Fatal(http.ListenAndServe("0.0.0.0:8766", nil))
	}()

	// Start syncing
	m.Logger.Info("Miser started running...")
	err = m.Sync()
	if err != nil {
		log.Fatal(err)
	}

}

func readConfig(path string) (*miser.Config, error) {
	var c miser.Config

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
