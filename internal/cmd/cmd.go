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

// loadConfig reads the config file and applies verbose-mode side effects.
func loadConfig(configPath string) *config.Config {
	logger.Infof("Running with cfg %s", configPath)
	cfg, err := config.NewConfigFile(configPath)
	if err != nil {
		panic(err)
	}
	if cfg.IsVerbose() {
		logger.Infof("Runnig in verbose mode")
		if os.Getenv(verboseEnvKey) != "false" {
			os.Setenv(verboseEnvKey, "true")
		}
	}
	return cfg
}

// runChecks builds a check per config entry and runs each in its own goroutine,
// returning once ctx is cancelled or every check has finished.
func runChecks(ctx context.Context, cfg *config.Config) {
	rules := detects.InitDetectsFromConfig(cfg)
	var wg sync.WaitGroup
	for _, r := range rules {
		logger.Infof("starting gorutine for %s", r.GetName())
		wg.Add(1)
		go func(r detects.Check) {
			defer func() {
				logger.Infof("goroutine for %s finished", r.GetName())
				wg.Done()
			}()
			r.Init(*cfg)
			r.Run(ctx)
		}(r)
	}
	wg.Wait()
}

func Do(configPath string) {
	cfg := loadConfig(configPath)
	runChecks(context.TODO(), cfg)
}
