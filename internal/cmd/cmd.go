package cmd

import (
	"context"
	"os"
	"sync"

	"goct/internal/config"
	"goct/internal/detects"
	"goct/internal/logger"
)

const (
	verboseEnvKey = "VERBOSE"
)

func Do(configPath string) {
	logger.Infof("Running with cfg %s", configPath)
	cfg, err := config.NewConfigFile(configPath)
	if err != nil {
		panic(err)
	}
	if cfg.IsVerbose() {
		logger.Infof("Runnig in verbose mode")
		verboseVar := os.Getenv(verboseEnvKey)
		if verboseVar != "false" {
			os.Setenv(verboseEnvKey, "true")
		}
	}
	rules := detects.InitDetectsFromConfig(cfg)
	var wg sync.WaitGroup
	ctx := context.TODO()
	for _, r := range rules {
		logger.Infof("starting gorutine for %s", r.GetName())
		wg.Add(1)
		go func(conf *config.Config, wg *sync.WaitGroup, r detects.Check) {
			defer wg.Done()
			r.Init(*cfg)
			r.Run(ctx)
		}(cfg, &wg, r)
	}
	wg.Wait()
}
