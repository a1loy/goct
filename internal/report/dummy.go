package report

import (
	"fmt"
	"os"

	"goct/internal/config"
)

type DummyReportingClient struct {
	Name        string
	URL         string
	MsgTemplate string
}

func NewDummyClient(cfg config.Config) *DummyReportingClient {
	return &DummyReportingClient{
		Name:        "dummy",
		URL:         "https://report.me/msg",
		MsgTemplate: "some_template",
	}
}

func (c *DummyReportingClient) Init(cfg config.Config) {
}

func (c *DummyReportingClient) Report(msg string) error {
	fmt.Fprintf(os.Stdout, "client %s reports %s\n", c.Name, msg)

	return nil
}
