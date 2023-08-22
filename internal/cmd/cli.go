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
	cliCmd      = &cobra.Command{
		Use:   "cli",
		Short: "Run goct in cli mode",
		Run: func(cmd *cobra.Command, args []string) {
			RunAsCli(logURI, lookupDepth)
		},
	}
)

func init() {
	cliCmd.Flags().StringVar(&logURI, "logUri", "", "CT Log URI")
	cliCmd.Flags().IntVar(&lookupDepth, "lookupDepth", 10, "Log lookup depth")
}

func RunAsCli(ctLogURI string, limit int) {
	logClient, err := ctlog.NewCtLogClient(ctLogURI)
	if err != nil {
		panic("unable to init ct log")
	}
	treeSize := logClient.GetTreeSize()
	logger.Debugf("tree size = %d", treeSize)
	var pos int64
	var delta int64 = 100
	if int64(limit) < delta {
		delta = int64(limit)
	}
	processedCerts := 0

	for pos = treeSize - 1; pos > treeSize-int64(limit); pos -= delta {
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
			// chain := leaf.Chain
			cn := ""
			if leaf.Precert == nil {
				logger.Debugf("unable to read cert at index %d\n", index)
				continue
			}
			cn = leaf.Precert.TBSCertificate.Subject.CommonName
			if loadErr != nil {
				logger.Infof("unable to print leaf with cn %s due to %s \n", cn, loadErr.Error())
				// return
			}
			addTime := ct.TimestampToTime(leaf.Leaf.TimestampedEntry.Timestamp)
			fmt.Fprintf(os.Stdout, "Found cert for %s at index %d issued at %s\n", cn,
				int(left)+len(resp.Entries)-index-1, addTime.String())
		}
	}
}
