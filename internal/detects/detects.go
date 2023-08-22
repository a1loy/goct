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
)

const (
	DefaultLookupDelta int64 = 100
	DefaultLookupDepth int64 = 36
)

type NewDetectFuncMap map[string]func(cfg config.Config, checkCfg config.CheckConfig) Check

var callbacks = NewDetectFuncMap{
	recently_issued_cert_name: NewRecentlyIssuedCert,
	invalid_cert:              NewInvalidCertDetect,
	match_by_regexp:           NewMatchByRegexpCert,
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
	signalChannels []chan struct{}, isDaemon bool, rescanInterval int) {
	for {
		scn := scanner.NewScanner(logClient, *opts)
		processed := 0
		err := scn.Scan(ctx,
			func(e *ct.RawLogEntry) {
				msg, err := ctlog.MsgFromLogEntry(e, detectName, "")
				if err != nil {
					logger.Errorf("unable to scan due to %v", err)
					return
				}
				processed = 0
				for _, ch := range eventChannels {
					ch <- msg
				}
			},
			func(e *ct.RawLogEntry) {
				msg, err := ctlog.MsgFromLogEntry(e, detectName, "")
				if err != nil {
					logger.Errorf("unable to scan due to %v", err)
					return
				}
				processed = 0
				for _, ch := range eventChannels {
					ch <- msg
				}
			})

		if err != nil {
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
			// unable to use scn.certsProcessed :(
			opts.StartIndex += int64(processed)
			sleepDuration := time.Duration(rescanInterval) * time.Second
			logger.Infof("iteration finished, taking a nap for %d", sleepDuration)
			time.Sleep(sleepDuration)
		}
	}
}

func setupChannels(reportEvents bool, storeEvents bool, eventChannels *[]chan models.DetectMsg, signalChannels *[]chan struct{},
	reportClient report.ReportClient, storeClient store.StoreClient) {
	if reportEvents {
		reportChan := make(chan models.DetectMsg)
		reportStopChan := make(chan struct{})
		*eventChannels = append(*eventChannels, reportChan)
		*signalChannels = append(*signalChannels, reportStopChan)
		go report.ReportEvent(reportChan, reportStopChan, reportClient)
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
	hoursBefore := DefaultLookupDelta
	if checkCfg.LookupDepth != 0 {
		hoursBefore = checkCfg.LookupDepth
	}
	issuedNotBefore := time.Now().Add(time.Hour * time.Duration(hoursBefore) * (-1))
	ctOpts, err := ctlog.InitScannerOpts(matcher, checkCfg, logClient)
	if err != nil {
		panic(err)
	}
	return issuedNotBefore, ctOpts
}
