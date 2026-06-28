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

// MsgDecorator transforms a DetectMsg before it is rendered into a report. It
// is handed a copy (not the message that goes to the store) and returns the
// value to render, so it can safely strip or rewrite fields.
type MsgDecorator func(models.DetectMsg) models.DetectMsg

// EntryDecorator returns a MsgDecorator that drops the raw Entry from the
// report unless the check enabled it via IncludeEntry. Because only the report
// path runs this, the stored copy keeps the full Entry.
func EntryDecorator(checkCfg config.CheckConfig) MsgDecorator {
	return func(msg models.DetectMsg) models.DetectMsg {
		if !checkCfg.IncludeEntry {
			msg.Entry = ""
		}
		return msg
	}
}

func ReportEvent(dataChan chan models.DetectMsg, controlChan chan struct{}, reportClient ReportClient, decorate MsgDecorator) {
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
			if decorate != nil {
				event = decorate(event)
			}
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
