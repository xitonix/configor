package configor

import (
	"fmt"
	"os"
	"regexp"
)

type Configor struct {
	*Config
	globalPrefix string
}

type Config struct {
	Environment string
	ENVPrefix   string
	Debug       bool
	Verbose     bool

	// In case of json files, this field will be used only when compiled with
	// go 1.10 or later.
	// This field will be ignored when compiled with go versions lower than 1.10.
	ErrorOnUnmatchedKeys bool
}

func (c *Config) getEnvPrefix() string {
	if prefix := os.Getenv("CONFIGOR_ENV_PREFIX"); prefix != "" {
		if prefix == "-" {
			return ""
		}
		return prefix
	}

	switch c.ENVPrefix {
	case "-":
		return ""
	case "":
		return "Configor"
	default:
		return c.ENVPrefix
	}
}

// New initialize a Configor
func New(config *Config) *Configor {
	if config == nil {
		config = &Config{}
	}

	if os.Getenv("CONFIGOR_DEBUG_MODE") != "" {
		config.Debug = true
	}

	if os.Getenv("CONFIGOR_VERBOSE_MODE") != "" {
		config.Verbose = true
	}

	c := &Configor{Config: config}
	c.globalPrefix = config.getEnvPrefix()
	return c
}

var testRegexp = regexp.MustCompile("_test|(\\.test$)")

// GetEnvironment get environment
func (c *Configor) GetEnvironment() string {
	if c.Environment == "" {
		if env := os.Getenv("CONFIGOR_ENV"); env != "" {
			return env
		}

		if testRegexp.MatchString(os.Args[0]) {
			return "test"
		}

		return "development"
	}
	return c.Environment
}

// GetErrorOnUnmatchedKeys returns a boolean indicating if an error should be
// thrown if there are keys in the config file that do not correspond to the
// config struct
func (c *Configor) GetErrorOnUnmatchedKeys() bool {
	return c.ErrorOnUnmatchedKeys
}

// Load will unmarshal configurations to struct from files that you provide
func (c *Configor) Load(config interface{}, files ...string) error {
	defer func() {
		if c.Config.Debug || c.Config.Verbose {
			fmt.Printf("Configuration:\n  %#v\n", config)
		}
	}()

	for _, file := range c.getConfigurationFiles(files...) {
		if c.Config.Debug || c.Config.Verbose {
			fmt.Printf("Loading configurations from file '%v'...\n", file)
		}
		if err := processFile(config, file, c.GetErrorOnUnmatchedKeys()); err != nil {
			return err
		}
	}

	if len(c.globalPrefix) > 0 {
		return c.processTags(config, c.globalPrefix)
	}
	return c.processTags(config)
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(config interface{}, files ...string) error {
	return New(nil).Load(config, files...)
}
