package ctlog

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"goct/internal/config"
	"goct/internal/logger"
	"goct/internal/models"

	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
)

// calcIndexFetchRetries bounds the per-entry fetch retries during the boundary
// search, so a flaky/rate-limited log doesn't make the search loop forever.
const calcIndexFetchRetries = 3

// entryAddTime fetches a single log entry and returns the time it was added to
// the log (its SCT timestamp), retrying a few times on transient errors. The
// timestamp lives on every entry type, so no precert check is needed.
func entryAddTime(ctLogClient CtLogClient, index int64) (time.Time, error) {
	var lastErr error
	for attempt := 0; attempt < calcIndexFetchRetries; attempt++ {
		resp, err := ctLogClient.GetEntries(index, index)
		switch {
		case err != nil:
			lastErr = err
		case len(resp.Entries) == 0:
			lastErr = fmt.Errorf("empty entries response for index %d", index)
		default:
			leaf, loadErr := ct.LogEntryFromLeaf(index, &resp.Entries[0])
			if loadErr != nil {
				return time.Time{}, loadErr
			}
			return ct.TimestampToTime(leaf.Leaf.TimestampedEntry.Timestamp), nil
		}
		time.Sleep(time.Second * time.Duration(attempt+1))
	}
	return time.Time{}, lastErr
}

// CalcIndex returns the first log index whose entry was added at or after
// issuedAt — i.e. where a scan should start to cover everything newer than the
// cutoff. Entries are appended in log order, so add-time is effectively
// monotonic in index and we can binary-search the boundary in O(log treeSize)
// single-entry fetches, which scales to full-size CT logs.
//
// On an unrecoverable fetch error it fails safe by returning treeSize (an empty
// scan window) rather than guessing a too-early index; the next rescan retries.
func CalcIndex(ctLogClient CtLogClient, issuedAt time.Time) int64 {
	treeSize := ctLogClient.GetTreeSize()
	if treeSize <= 0 {
		return 0
	}
	logger.Debugf("tree size = %d, looking for first entry added at/after %s", treeSize, issuedAt)

	lo, hi := int64(0), treeSize
	for lo < hi {
		mid := lo + (hi-lo)/2
		addTime, err := entryAddTime(ctLogClient, mid)
		if err != nil {
			logger.Errorf("CalcIndex: giving up at entry %d, scanning nothing this pass: %v", mid, err)
			return treeSize
		}
		if addTime.Before(issuedAt) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	logger.Debugf("CalcIndex: starting scan at index %d", lo)
	return lo
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
