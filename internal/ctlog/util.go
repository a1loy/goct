package ctlog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"goct/internal/config"
	"goct/internal/logger"
	"goct/internal/models"

	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
)

func calcPostTasks(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, treeSize int64, delta int64, tasksChan chan []int64, foundChan chan bool) {
	defer func() {
		wg.Done()
		logger.Debugf("exiting from calcPostTasks routine")
		close(tasksChan)
		cancel()
	}()
	for pos := treeSize - 1; pos > 0; pos -= delta {
		left := pos - delta
		if left < 0 {
			left = 0
		}
		select {
		case tasksChan <- []int64{left, pos}:
			logger.Debugf("posting %d %d\n", left, pos)
		case <-foundChan:
			return
		default:
			logger.Debugf("unable to post, waiting...\n")
			time.Sleep(time.Second * 1)
		}
	}
}

func calcProcessTasks(ctx context.Context, wg *sync.WaitGroup, issuedAt time.Time, tasksChan chan []int64, foundChan chan bool, resChan chan int64,
	ctLogClient CtLogClient) {
	defer func() {
		logger.Debugf("exiting from calcProcessTasks routine")
	}()
	for {
		select {
		case pair, chanOpen := <-tasksChan:
			if !chanOpen {
				return
			}
			left, pos := pair[0], pair[1]
			logger.Infof("received task to retrieve %d %d", left, pos)
			resp, err := ctLogClient.GetEntries(left, pos)
			if err != nil {
				logger.Errorf("unable to get entries %d %d due to %s", left, pos, err)
			}
			for index := range resp.Entries {
				// processedCerts++
				reverseIndex := len(resp.Entries) - index - 1
				entry := resp.Entries[reverseIndex]
				leaf, loadErr := ct.LogEntryFromLeaf(0, &entry)
				if loadErr != nil {
					logger.Infof("unable to print leaf with cn %s due to %s \n",
						leaf.Precert.TBSCertificate.Subject.CommonName, loadErr.Error())
					continue
				}
				if leaf.Precert == nil {
					logger.Debugf("unable to read leaf cert at index %d\n", reverseIndex)
					continue
				}
				addTime := ct.TimestampToTime(leaf.Leaf.TimestampedEntry.Timestamp)
				if addTime.Before(issuedAt) {
					logger.Debugf("found cert issued at %s", addTime)
					// logger.Debugf("left = %d pos = %d index = %d = certs_processed = %d", left, pos, reverseIndex, processedCerts)
					logger.Debugf("end pos %d", int(left)+reverseIndex)
					resChan <- left + int64(reverseIndex) + 1
					foundChan <- true
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func CalcIndex(ctLogClient CtLogClient, issuedAt time.Time, workersCount int) int64 {
	treeSize := ctLogClient.GetTreeSize()
	logger.Debugf("tree size = %d", treeSize)
	logger.Debugf("looking for certs issued after %s", issuedAt.String())
	var delta int64 = LogLookupBatchSize
	bufSize := 32
	ctx, ctxWithCancel := context.WithCancel(context.Background())
	pairsChan := make(chan []int64, bufSize)
	foundChan := make(chan bool)
	resChan := make(chan int64, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go calcPostTasks(ctx, ctxWithCancel, &wg, treeSize, delta, pairsChan, foundChan)

	for i := 0; i < workersCount; i++ {
		logger.Debugf("starting calcProcessTasks\n")
		go calcProcessTasks(ctx, &wg, issuedAt, pairsChan, foundChan, resChan, ctLogClient)
	}
	wg.Wait()
	return <-resChan
}

func InitScannerOpts(matcher scanner.Matcher, cfg config.CheckConfig,
	logClient CtLogClient, issuedNotBefore time.Time) (*scanner.ScannerOptions, error) {
	numWorkers := runtime.NumCPU()
	if numWorkers < cfg.WorkersCount {
		numWorkers = cfg.WorkersCount
	}
	opts := scanner.DefaultScannerOptions()
	opts.Matcher = matcher
	// TODO: support configuration of BatchSize, NumWorkers, ParallelFetch, etc
	// opts.BatchSize = cfg.BatchSize
	opts.BatchSize = 1000
	opts.NumWorkers = numWorkers
	opts.ParallelFetch = numWorkers
	// opts.TickTime = time.Duration(cfg.TickTime) * time.Second
	opts.StartIndex = CalcIndex(logClient, issuedNotBefore, numWorkers)
	opts.EndIndex = logClient.GetTreeSize()
	// opts.Tickers = []scanner.Ticker{scanner.LogTicker{}}
	// opts.Quiet = false
	return opts, nil
}

func MsgFromLogEntry(e *ct.RawLogEntry, detectName string, raw string) (models.DetectMsg, error) {
	entry, _ := e.ToLogEntry()
	precert := entry.Precert
	if precert == nil {
		return models.DetectMsg{}, errors.New("precert not found")
	}
	hash := sha256.Sum256(precert.Submitted.Data)
	cn := precert.TBSCertificate.Subject.CommonName
	issuer := precert.TBSCertificate.Issuer.CommonName
	dnsNames := precert.TBSCertificate.DNSNames
	issuedAt := precert.TBSCertificate.NotBefore
	return models.DetectMsg{Name: detectName,
		Entry:        base64.StdEncoding.EncodeToString(precert.Submitted.Data),
		Hash:         fmt.Sprintf("%x", hash),
		CN:           cn,
		IssuerName:   issuer,
		DNSNames:     strings.Join(dnsNames, ","),
		IssuanceDate: issuedAt,
		Raw:          raw}, nil
}
