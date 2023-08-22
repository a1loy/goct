package detects

import (
	"goct/internal/config"
	"goct/internal/ctlog"
	"goct/internal/logger"
	"goct/internal/models"
	"goct/internal/report"
	"goct/internal/store"
	"regexp"
)

func NewRecentlyIssuedCert(cfg config.Config, checkCfg config.CheckConfig) Check {
	ctLogClients := make([]ctlog.CtLogClient, 0)

	if len(checkCfg.Logs) != 0 {
		for _, ctLogURI := range checkCfg.Logs {
			ctLogClient, err := ctlog.NewCtLogClient(ctLogURI)
			if err != nil {
				logger.Infof("unable to init ct log for %s", ctLogURI)
				continue
			}
			ctLogClients = append(ctLogClients, *ctLogClient)
		}
	}
	checkCfg.RescanInterval = cfg.RescanInterval
	msgs := make([]models.DetectMsg, 0)
	regexs := make([]*regexp.Regexp, 0)
	regex := regexp.MustCompile(".*")
	regexs = append(regexs, regex)

	storeClient, err := store.NewSqliteStoreClient(cfg)
	if err != nil {
		logger.Infof("unable to init store client %v", err)
		// panic(err)
	}

	return &MatchByRegexpCert{Name: checkCfg.Name, ReportClient: report.NewTelegramClient(cfg),
		StoreClient: storeClient,
		Config:      checkCfg, CtLogClients: &ctLogClients, Regexs: &regexs, Results: &msgs, IsDaemon: cfg.IsDaemon}
}
