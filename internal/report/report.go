package report

import (
	"fmt"
	"goct/internal/config"
	"goct/internal/logger"
	"goct/internal/models"
	"strings"
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
				reportErr := reportClient.Report(formatReportMsg(event))
				if reportErr != nil {
					logger.Errorf("unable to report %s", reportErr)
				}
				filter[event.Hash] = true
			}
		}
	}
}

func formatReportMsg(msg models.DetectMsg) string {
	name := strings.Replace(msg.Name, "_", " ", -1)
	s := fmt.Sprintf("``` \n[Check]```%s ``` \n[CN] %s\n```", name, msg.CN)
	return s
}
