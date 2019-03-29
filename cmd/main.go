package main

import (
	"fmt"
	"github.com/xitonix/configor"
	"log"
)

type Connection struct {
	Endpoint string `json:"endpoint" required:"true"`
}

type config struct {
	*Connection
	Port int  `json:"port" required:"true"`
	I    *int `json:"integer" required:"true""`
}

func main() {
	i := 1000
	cfg := config{
		Port: 100,
		I:    &i,
	}
	err := configor.New(&configor.Config{
		ENVPrefix:            "APP",
		Debug:                false,
		Verbose:              false,
		ErrorOnUnmatchedKeys: true,
	}).Load(&cfg)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v", cfg.Endpoint)
}
