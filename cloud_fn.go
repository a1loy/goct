package main

import (
	"flag"
	"fmt"
	"os"

	"goct/internal/cmd"
	"goct/internal/logger"
)

const (
	defaultConfigName = "config.yaml"
	defaultStatePath  = "state"
	defaultModeEnv    = "RUN_MODE"
)

func processFlags() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	var configPath = flag.String("config", cwd+"/"+defaultConfigName, "relative path to config")
	// var statePath = flag.String("state", cwd+"/"+defaultStatePath, "relative path to state")
	if os.Getenv(defaultModeEnv) == "CLOUD_FUNC" {
		// TODO: better solution for cloud funcs
		cpath := os.Getenv("CONFIG_PATH")
		configPath = &cpath
		// spath := os.Getenv("STATE_PATH")
		// statePath = &spath
	} else {
		flag.Parse()
	}
	return cwd + "/" + *configPath, nil
}

func CloudFuncHandler() error {
	logger.Infof("running mode: %s", os.Getenv(defaultModeEnv))
	cfgPath, err := processFlags()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Unable to process flags %s", err.Error())
		os.Exit(1)
	}
	cmd.Do(cfgPath)
	return nil
}

func main() {
	logger.Infof("running mode: %s", os.Getenv(defaultModeEnv))
	cfgPath, err := processFlags()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Unable to process flags %s", err.Error())
		os.Exit(1)
	}
	cmd.Do(cfgPath)
}
