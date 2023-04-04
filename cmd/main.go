package main

import (
	"flag"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"miser"
	"miser/agent"
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

	// Init Miser
	Miser, err := agent.NewMiser(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Start syncing
	Miser.Logger.Info("Miser started running...")
	err = Miser.Sync()
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
