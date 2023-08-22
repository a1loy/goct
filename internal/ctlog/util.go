package ctlog

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"goct/internal/config"
	"goct/internal/logger"
	"goct/internal/models"

	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
)

func CalcIndex(ctLogClient CtLogClient, issuedAt time.Time) int64 {
	treeSize := ctLogClient.GetTreeSize()
	logger.Debugf("tree size = %d", treeSize)
	logger.Debugf("looking for certs issued after %s", issuedAt.String())
	var pos int64
	var delta int64 = 100
	processedCerts := 0
	var endPos int64 = 0
	for pos = treeSize - 1; pos > 0; pos -= delta {
		left := pos - delta
		if left < 0 {
			left = 0
		}
		resp, err := ctLogClient.GetEntries(left, pos)
		if err != nil {
			return 0
		}
		for index := range resp.Entries {
			processedCerts++
			reverseIndex := len(resp.Entries) - index - 1
			entry := resp.Entries[reverseIndex]
			leaf, loadErr := ct.LogEntryFromLeaf(0, &entry)
			if leaf.Precert == nil {
				logger.Debugf("unable to read cert at index %d\n", reverseIndex)

				continue
			}
			if loadErr != nil {
				logger.Infof("unable to print leaf with cn %s due to %s \n",
					leaf.Precert.TBSCertificate.Subject.CommonName, loadErr.Error())
			}
			addTime := ct.TimestampToTime(leaf.Leaf.TimestampedEntry.Timestamp)
			if addTime.Before(issuedAt) {
				logger.Debugf("found cert issued at %s", addTime)
				logger.Debugf("left = %d pos = %d index = %d = certs_processed = %d", left, pos, reverseIndex, processedCerts)
				logger.Debugf("end pos %d", int(left)+reverseIndex)
				return left + int64(reverseIndex) + 1
			}
		}
	}
	return endPos
}

func InitScannerOpts(matcher scanner.Matcher, cfg config.CheckConfig,
	logClient CtLogClient) (*scanner.ScannerOptions, error) {
	issuedNotBefore := time.Now().Add(time.Hour * time.Duration(cfg.LookupDepth) * (-1))
	opts := scanner.DefaultScannerOptions()
	opts.Matcher = matcher
	// TODO: support configuration of BatchSize, NumWorkers, ParallelFetch, etc
	// opts.BatchSize = cfg.BatchSize
	// opts.NumWorkers = cfg.NumWorkers
	// opts.ParallelFetch = cfg.ParallelFetch
	// opts.TickTime = time.Duration(cfg.TickTime) * time.Second
	opts.StartIndex = CalcIndex(logClient, issuedNotBefore)
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
	return models.DetectMsg{Name: detectName,
		Entry: base64.StdEncoding.EncodeToString(precert.Submitted.Data),
		Hash:  fmt.Sprintf("%x", hash),
		// Entry: string(leaf.Precert.Submitted.Data),
		CN: cn, Raw: raw}, nil
}
