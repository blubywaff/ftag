package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config_Gremlin struct {
	Url string
}

type Config struct {
	Gremlin Config_Gremlin
	UrlBase string
}

var Global Config

func Load() {
	var (
		cleanupFlag    = flag.Bool("clean", false, "If the database should be cleaned on startup.")
		configPathFlag = flag.String("config", "ftag.config.json", "The location of the config file.")
	)

	flag.Parse()

	if *cleanupFlag {
		log.Fatal("Feature Not Supported")
	}

	// Parse Config
	bts, err := os.ReadFile(*configPathFlag)
	if err != nil {
		log.Fatal("failed to read config:", err)
	}
	json.Unmarshal(bts, &Global)
}
