package store

import (
	"goct/internal/logger"
	"goct/internal/models"
)

type StoreClient interface {
	Init() error
	Store(msg models.DetectMsg) error
	IsReady() bool
}

func StoreEvent(dataChan chan models.DetectMsg, controlChan chan struct{}, storeClient StoreClient) {
	if storeClient == nil {
		return
	}
	if !storeClient.IsReady() {
		return
	}
	filter := make(map[string]bool)
	for {
		select {
		case <-controlChan:
			return
		default:
			event := <-dataChan
			_, ok := filter[event.Hash]
			if !ok {
				reportErr := storeClient.Store(event)
				if reportErr != nil {
					logger.Errorf("unable to store %s", reportErr)
				}
				filter[event.Hash] = true
			}
		}
	}
}
