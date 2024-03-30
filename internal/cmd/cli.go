package cmd

import (
	"fmt"
	"os"

	"goct/internal/ctlog"
	"goct/internal/logger"

	ct "github.com/google/certificate-transparency-go"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cliCmd)
}

var (
	logURI      string
	lookupDepth int
	lookupCount int
	cliCmd      = &cobra.Command{
		Use:   "cli",
		Short: "Run goct in cli mode",
		Run: func(cmd *cobra.Command, args []string) {
			RunAsCli(logURI, lookupDepth, lookupCount)
		},
	}
)

func init() {
	cliCmd.Flags().StringVar(&logURI, "logUri", "", "CT Log URI")
	cliCmd.Flags().IntVar(&lookupDepth, "lookupDepth", 10, "Log lookup depth")
	cliCmd.Flags().IntVar(&lookupCount, "lookupCount", 10, "Number of certs to lookup")
}

func RunAsCli(ctLogURI string, limit, lookupCount int) {
	logClient, err := ctlog.NewCtLogClient(ctLogURI)
	if err != nil {
		panic("unable to init ct log")
	}
	treeSize := logClient.GetTreeSize()
	logger.Debugf("tree size = %d", treeSize)
	var pos int64 = treeSize - 1
	var delta int64 = ctlog.LogLookupBatchSize
	processedCerts := 0
	certsToProcess := lookupCount
	for certsToProcess > 0 && pos > 0 {
		logger.Debugf("certsToProcess = %d pos = %d", certsToProcess, pos)
		left := pos - delta
		if left < 0 {
			left = 0
		}
		logger.Infof("retreiving %d %d", left, pos)
		resp, err := logClient.GetEntries(left, pos)
		if err != nil {
			return
		}
		logger.Infof("received %d entries", len(resp.Entries))
		for index := range resp.Entries {
			entry := resp.Entries[len(resp.Entries)-index-1]
			leaf, loadErr := ct.LogEntryFromLeaf(0, &entry)
			processedCerts++
			if leaf.Precert == nil {
				continue
			}
			cn := leaf.Precert.TBSCertificate.Subject.CommonName
			if loadErr != nil {
				logger.Infof("unable to print leaf with cn %s due to %s \n", cn, loadErr.Error())
				continue
			}
			if certsToProcess > 0 {
				addTime := ct.TimestampToTime(leaf.Leaf.TimestampedEntry.Timestamp)
				fmt.Fprintf(os.Stdout, "Found cert for %s at index %d issued at %s\n", cn,
					int(left)+len(resp.Entries)-index-1, addTime.String())
				certsToProcess--
			} else {
				break
			}
		}
		pos -= delta
	}
}
