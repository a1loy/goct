package detects

import (
	"context"
	"time"

	"goct/internal/config"
	"goct/internal/ctlog"
	"goct/internal/logger"
	"goct/internal/matcher"
	"goct/internal/models"
	"goct/internal/report"
	"goct/internal/store"

	scanner "github.com/google/certificate-transparency-go/scanner"
)

type InvalidCertDetect struct {
	Name            string
	ReportClient    report.ReportClient
	StoreClient     store.StoreClient
	Config          config.CheckConfig
	CtLogClients    *[]ctlog.CtLogClient
	Results         *[]models.DetectMsg
	IssuedNotBefore time.Time
	IsDaemon        bool
	CtScannerOpts   *scanner.ScannerOptions
}

func (c *InvalidCertDetect) Init(cfg config.Config) {
	c.ReportClient.Init(cfg)
	if c.StoreClient.IsReady() {
		initErr := c.StoreClient.Init()
		if initErr != nil {
			panic(initErr)
		}
	}
	matcher, _ := matcher.NewCustomCertMatcherByValidity(c.Config)
	c.IssuedNotBefore, c.CtScannerOpts = buildDetectFields(c.GetConfig(), matcher, (*c.CtLogClients)[0])
}

func (c *InvalidCertDetect) Run(ctx context.Context) {
	logClient, _ := ctlog.NewLogClient((*c.CtLogClients)[0].LogURI)
	logger.Debugf("scanner opts indexes start %d end %d\n", c.CtScannerOpts.StartIndex, c.CtScannerOpts.EndIndex)
	reportEvents := c.ReportClient != nil
	storeEvents := true
	if c.StoreClient == nil {
		storeEvents = false
	}
	storeEvents = c.StoreClient.IsReady()

	var eventChannels []chan models.DetectMsg
	var signalChannels []chan struct{}
	setupChannels(reportEvents, storeEvents, &eventChannels, &signalChannels, c.ReportClient, c.StoreClient)
	// ctx := context.TODO()
	runScan(ctx, c.Name, logClient, c.CtScannerOpts, eventChannels, signalChannels, c.IsDaemon, c.Config.RescanInterval)
	logger.Debugf("iteration finised")
	for _, ch := range signalChannels {
		close(ch)
	}
}

func (c *InvalidCertDetect) GetName() string {
	return c.Name
}

func NewInvalidCertDetect(cfg config.Config, checkCfg config.CheckConfig) Check {
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
	storeClient, err := store.NewSqliteStoreClient(cfg)
	if err != nil {
		logger.Infof("unable to init store client %v", err)
		// panic(err)
	}

	return &InvalidCertDetect{Name: checkCfg.Name,
		ReportClient: report.NewTelegramClient(cfg),
		StoreClient:  storeClient,
		Config:       checkCfg, CtLogClients: &ctLogClients,
		Results: &msgs, IsDaemon: cfg.IsDaemon,
	}
}

func (c *InvalidCertDetect) GetConfig() config.CheckConfig {
	return c.Config
}
