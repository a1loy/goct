package report

import (
	"goct/internal/config"
	"goct/internal/logger"
	"goct/internal/models"
)

type ReportClient interface {
	Init(cfg config.Config)
	Report(msg string) error
}

func ReportEvent(dataChan chan models.DetectMsg, controlChan chan struct{}, reportClient ReportClient) {
	filter := make(map[string]bool)
	if reportClient == nil {
		return
	}
	for {
		select {
		case <-controlChan:
			return
		default:
			event := <-dataChan
			_, ok := filter[event.Hash]
			if !ok {
				reportErr := reportClient.Report(event.ToMarkdownString())
				if reportErr != nil {
					logger.Errorf("unable to report %s", reportErr)
				}
				filter[event.Hash] = true
			}
		}
	}
}
