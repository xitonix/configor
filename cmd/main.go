package main

import (
	"fmt"
	"github.com/xitonix/configor"
	"log"
)

type connection struct {
	Endpoint string `json:"endpoint" required:"true"`
}

type config struct {
	Connection connection `json:"database"`
	Port       int        `json:"port" required:"true"`
}

func main() {

	cfg := config{}
	err := configor.New(&configor.Config{
		ENVPrefix:            "APP",
		Debug:                false,
		Verbose:              false,
		ErrorOnUnmatchedKeys: true,
	}).Load(&cfg)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v", cfg)
}
