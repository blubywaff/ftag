package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Configurable interface {
    Config() *interface{}
}
