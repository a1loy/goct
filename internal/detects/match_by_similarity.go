package detects

import (
	"context"
	"goct/internal/config"
	"goct/internal/ctlog"
	"goct/internal/logger"
	"goct/internal/matcher"
	"goct/internal/models"
	"goct/internal/report"
	"goct/internal/store"
	"time"

	scanner "github.com/google/certificate-transparency-go/scanner"
)

type MatchBySimilarityCert struct {
	Name         string
	ReportClient report.ReportClient
	StoreClient  store.StoreClient
	Config       config.CheckConfig
	CtLogClients *[]ctlog.CtLogClient
	Results      *[]models.DetectMsg
	// Regexs          *[]*regexp.Regexp
	SimilarityPatterns []string
	SimilarityCfg      config.SimilarityCheckCfg
	IssuedNotBefore    time.Time
	IsDaemon           bool
	CtScannerOpts      *scanner.ScannerOptions
}

func (c *MatchBySimilarityCert) Run(ctx context.Context) {
	logClient, _ := ctlog.NewLogClient((*c.CtLogClients)[0].LogURI)
	logger.Debugf("scanner opts indexes start %d end %d\n", c.CtScannerOpts.StartIndex, c.CtScannerOpts.EndIndex)
	reportEvents := c.ReportClient != nil
	storeEvents := false
	if c.StoreClient != nil {
		storeEvents = c.StoreClient.IsReady()
	}
	var eventChannels []chan models.DetectMsg
	var signalChannels []chan struct{}
	setupChannels(reportEvents, storeEvents, &eventChannels, &signalChannels, c.ReportClient, c.StoreClient)
	runScan(ctx, c.Name, logClient, c.CtScannerOpts, eventChannels, signalChannels, c.IsDaemon, c.Config.RescanInterval)
	for _, ch := range signalChannels {
		close(ch)
	}
}

func (c *MatchBySimilarityCert) Init(cfg config.Config) {
	c.ReportClient.Init(cfg)
	if c.StoreClient.IsReady() {
		initErr := c.StoreClient.Init()
		if initErr != nil {
			panic(initErr)
		}
	}
	matcher, err := matcher.NewCustomCertMatcherBySimilarity(c.SimilarityPatterns, c.SimilarityCfg)
	if err != nil {
		panic(err)
	}
	c.IssuedNotBefore, c.CtScannerOpts = buildDetectFields(c.GetConfig(), matcher, (*c.CtLogClients)[0])
}

func (c MatchBySimilarityCert) GetName() string {
	return c.Name
}

func NewMatchBySimilarityCert(cfg config.Config, checkCfg config.CheckConfig) Check {
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
	if checkCfg.WorkersCount == 1 {
		checkCfg.WorkersCount = cfg.WorkersCount
	}
	checkCfg.RescanInterval = cfg.RescanInterval
	msgs := make([]models.DetectMsg, 0)

	storeClient, err := store.NewSqliteStoreClient(cfg)
	if err != nil {
		logger.Infof("unable to init store client: %v", err)
		// panic(err)
	}
	return &MatchBySimilarityCert{Name: checkCfg.Name, ReportClient: report.NewTelegramClient(cfg),
		StoreClient: storeClient,
		Config:      checkCfg, CtLogClients: &ctLogClients,
		SimilarityPatterns: checkCfg.Patterns, SimilarityCfg: checkCfg.SimilarityCfg,
		Results: &msgs, IsDaemon: cfg.IsDaemon}
}

func (c *MatchBySimilarityCert) GetConfig() config.CheckConfig {
	return c.Config
}
