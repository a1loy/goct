package detects

import (
	"context"
	"goct/internal/config"
	"goct/internal/ctlog"
	"goct/internal/logger"
	"goct/internal/models"
	"goct/internal/report"
	"goct/internal/store"
	"time"

	ct "github.com/google/certificate-transparency-go"
	client "github.com/google/certificate-transparency-go/client"
	scanner "github.com/google/certificate-transparency-go/scanner"
)

const (
	recently_issued_cert_name = "recently_issued_cert"
	invalid_cert              = "invalid_cert"
	match_by_regexp           = "match_by_regexp"
	match_by_similarity       = "match_by_similarity"
)

const (
	DefaultLookupDelta int64 = 100
	DefaultLookupDepth int64 = 60
)

type NewDetectFuncMap map[string]func(cfg config.Config, checkCfg config.CheckConfig) Check

var callbacks = NewDetectFuncMap{
	recently_issued_cert_name: NewRecentlyIssuedCert,
	invalid_cert:              NewInvalidCertDetect,
	match_by_regexp:           NewMatchByRegexpCert,
	match_by_similarity:       NewMatchBySimilarityCert,
}

type Check interface {
	Init(cfg config.Config)
	Run(ctx context.Context)
	GetName() string
	GetConfig() config.CheckConfig
}

func InitDetectsFromConfig(cfg *config.Config) map[string]Check {
	rules := make(map[string]Check, 0)
	for _, customCfg := range cfg.Checks {
		newDetectCallback, ok := callbacks[customCfg.Name]
		if ok {
			rules[customCfg.Name] = newDetectCallback(*cfg, customCfg)
		}
	}
	return rules
}

func runScan(ctx context.Context, detectName string, logClient *client.LogClient, opts *scanner.ScannerOptions, eventChannels []chan models.DetectMsg,
	signalChannels []chan struct{}, isDaemon bool, checkCfg config.CheckConfig) {
	onMatch := func(e *ct.RawLogEntry) {
		msg, err := ctlog.MsgFromLogEntry(e, detectName, "")
		if err != nil {
			logger.Debugf("unable to scan due to %v", err)
			return
		}
		for _, ch := range eventChannels {
			ch <- msg
		}
	}

	for {
		scn := scanner.NewScanner(logClient, *opts)
		if err := scn.Scan(ctx, onMatch, onMatch); err != nil {
			for _, ch := range signalChannels {
				close(ch)
			}
			panic(err)
		}
		if !isDaemon {
			break
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Resume where this pass ended; extend to the log's current head.
		sth, err := logClient.GetSTH(ctx)
		if err != nil {
			logger.Errorf("unable to refresh STH, re-scanning same window: %v", err)
		} else {
			opts.StartIndex = opts.EndIndex
			opts.EndIndex = int64(sth.TreeSize)
		}
		if opts.EndIndex <= opts.StartIndex {
			logger.Infof("no new entries since index %d", opts.StartIndex)
		}

		sleepDuration := time.Duration(checkCfg.RescanInterval) * time.Second
		logger.Infof("iteration finished, taking a nap for %s", sleepDuration)
		time.Sleep(sleepDuration)
	}
}

func setupChannels(reportEvents bool, storeEvents bool, eventChannels *[]chan models.DetectMsg, signalChannels *[]chan struct{},
	reportClient report.ReportClient, storeClient store.StoreClient, decorate report.MsgDecorator) {
	if reportEvents {
		reportChan := make(chan models.DetectMsg)
		reportStopChan := make(chan struct{})
		*eventChannels = append(*eventChannels, reportChan)
		*signalChannels = append(*signalChannels, reportStopChan)
		go report.ReportEvent(reportChan, reportStopChan, reportClient, decorate)
	}
	if storeEvents {
		storeChan := make(chan models.DetectMsg)
		storeStopChan := make(chan struct{})
		*eventChannels = append(*eventChannels, storeChan)
		*signalChannels = append(*signalChannels, storeStopChan)
		go store.StoreEvent(storeChan, storeStopChan, storeClient)
	}
}

func buildDetectFields(checkCfg config.CheckConfig, matcher scanner.Matcher, logClient ctlog.CtLogClient) (time.Time, *scanner.ScannerOptions) {
	minutesBefore := DefaultLookupDelta
	if checkCfg.LookupDepth != 0 {
		minutesBefore = checkCfg.LookupDepth
	}
	issuedNotBefore := time.Now().Add(time.Minute * time.Duration(minutesBefore) * (-1))
	ctOpts, err := ctlog.InitScannerOpts(matcher, checkCfg, logClient, issuedNotBefore)
	if err != nil {
		panic(err)
	}
	logger.Infof("running with scanner opts: BatchSize: %d, NumWorkers: %d, scanning from %d to %d",
		ctOpts.BatchSize, ctOpts.NumWorkers, ctOpts.StartIndex, ctOpts.EndIndex)
	return issuedNotBefore, ctOpts
}
