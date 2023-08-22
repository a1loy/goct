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
	"regexp"
	"time"

	scanner "github.com/google/certificate-transparency-go/scanner"
)

type MatchByRegexpCert struct {
	Name            string
	ReportClient    report.ReportClient
	StoreClient     store.StoreClient
	Config          config.CheckConfig
	CtLogClients    *[]ctlog.CtLogClient
	Results         *[]models.DetectMsg
	Regexs          *[]*regexp.Regexp
	IssuedNotBefore time.Time
	IsDaemon        bool
	CtScannerOpts   *scanner.ScannerOptions
}

func (c *MatchByRegexpCert) Run(ctx context.Context) {
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

func (c *MatchByRegexpCert) Init(cfg config.Config) {
	c.ReportClient.Init(cfg)
	if c.StoreClient.IsReady() {
		initErr := c.StoreClient.Init()
		if initErr != nil {
			panic(initErr)
		}
	}
	matcher, _ := matcher.NewCustomCertMatcherByRegex(c.Regexs)
	c.IssuedNotBefore, c.CtScannerOpts = buildDetectFields(c.GetConfig(), matcher, (*c.CtLogClients)[0])
}

func (c MatchByRegexpCert) GetName() string {
	return c.Name
}

func NewMatchByRegexpCert(cfg config.Config, checkCfg config.CheckConfig) Check {
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
	for _, regexStr := range checkCfg.Regex {
		regexObj, compileErr := regexp.Compile(regexStr)
		if compileErr == nil && len(regexStr) > 0 {
			regexs = append(regexs, regexObj)
		} else {
			logger.Infof("unable to build regexp for %s", regexStr)
		}
	}

	storeClient, err := store.NewSqliteStoreClient(cfg)
	if err != nil {
		logger.Infof("unable to init store client: %v", err)
		// panic(err)
	}
	return &MatchByRegexpCert{Name: checkCfg.Name, ReportClient: report.NewTelegramClient(cfg),
		StoreClient: storeClient,
		Config:      checkCfg, CtLogClients: &ctLogClients, Regexs: &regexs, Results: &msgs, IsDaemon: cfg.IsDaemon}
}

func (c *MatchByRegexpCert) GetConfig() config.CheckConfig {
	return c.Config
}
