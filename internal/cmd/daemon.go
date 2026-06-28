package cmd

import (
	"context"
	"goct/internal/healthcheck"
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
	cfg := loadConfig(configPath)

	// The daemon section wins over the command-line flags; fall back to the
	// flags for whatever it didn't set. Populating the section is also what
	// marks this run as daemon mode (see config.Config.IsDaemon).
	if cfg.Daemon.RescanInterval == nil {
		cfg.Daemon.RescanInterval = &rescanInterval
	}
	if cfg.Daemon.Foreground == nil {
		cfg.Daemon.Foreground = &foreground
	}

	ctx := context.Background()
	if cfg.EnableHealthCheck() {
		healthcheckCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		go healthcheck.RunHealthCheck(healthcheckCtx, healthcheck.Host, healthcheck.Port, healthcheck.Endpoint)
		defer stop()
	}
	// Must be set before runChecks builds the checks, since each check copies
	// RescanInterval out of cfg in its constructor.
	cfg.RescanInterval = *cfg.Daemon.RescanInterval

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	runChecks(runCtx, cfg)
}
