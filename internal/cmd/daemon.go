package cmd

import (
	"context"
	"goct/internal/config"
	"goct/internal/detects"
	"goct/internal/healthcheck"
	"goct/internal/logger"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var (
	rescanInterval int
	foreground     bool
	daemonCmd      = &cobra.Command{
		Use:   "daemon",
		Short: "Run goct in daemon mode",
		Run: func(cmd *cobra.Command, args []string) {
			RunAsDaemon(configFile, rescanInterval, foreground)
		},
	}
)

func init() {
	daemonCmd.Flags().IntVar(&rescanInterval, "rescan", 60, "Rescan every amount of seconds")
	daemonCmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground")
}

func RunAsDaemon(configPath string, rescanInterval int, foreground bool) {
	// TODO: init config in more generic way
	logger.Infof("Running with cfg %s", configPath)
	cfg, err := config.NewConfigFile(configPath)
	if err != nil {
		panic(err)
	}
	if cfg.IsVerbose() {
		logger.Infof("Runnig in verbose mode")
		debugEnv := os.Getenv(verboseEnvKey)
		if debugEnv != "false" {
			os.Setenv(verboseEnvKey, "true")
		}
	}
	ctx := context.Background()
	if cfg.EnableHealthCheck() {
		healthcheckCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		go healthcheck.RunHealthCheck(healthcheckCtx, healthcheck.Host, healthcheck.Port, healthcheck.Endpoint)
		defer stop()
	}
	cfg.RescanInterval = rescanInterval
	cfg.IsDaemon = true
	rules := detects.InitDetectsFromConfig(cfg)

	hardcodedCheck := "match_by_regexp"

	customConfigs := make(map[string]config.CheckConfig)
	for _, customCfg := range cfg.Checks {
		_, ok := rules[customCfg.Name]
		if ok {
			customConfigs[customCfg.Name] = customCfg
		}
	}
	check := detects.NewMatchByRegexpCert(*cfg, customConfigs[hardcodedCheck])
	check.Init(*cfg)
	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	check.Run(runCtx)
	defer stop()
}
