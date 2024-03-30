package models

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const (
	DetectMsgMarkdownTemplate = "[Check] ``` {{.Name}} ```\r\n" +
		"[CN] ``` {{.CN}} ```\r\n" +
		"[Hash] ``` {{.Hash}} ```\r\n" +
		"[Issued by] ``` {{.IssuerName}} ```\r\n" +
		"[Issued at] ```{{.IssuanceDate}} ```\r\n" +
		"[AltNames] ```{{.DNSNames}} ``` \r\n" +
		"[Entry] ``` {{.Entry}} ```\r\n"
)

type DetectMsg struct {
	Name         string `json:"name"`
	Entry        string `json:"entry"`
	CN           string `json:"cn,omitempty"`
	Hash         string `json:"hash,omitempty"`
	Raw          string `json:"raw,omitempty"`
	IssuanceDate time.Time
	IssuerName   string `json:"issuer,omitempty"`
	DNSNames     string `json:"dnsnames,omitempty"`
}

func (m *DetectMsg) String() string {
	name := strings.Replace(m.Name, "_", " ", -1)
	return fmt.Sprintf("``` \n[Check]```%s ``` \n[CN] %s\n```", name, m.CN)
}

func (m *DetectMsg) ToMarkdownString() string {
	t, err := template.New("OneMsg").Parse(DetectMsgMarkdownTemplate)
	if err != nil {
		panic(err)
	}
	var tpl bytes.Buffer
	err = t.Execute(&tpl, m)
	if err != nil {
		panic(err)
	}
	return tpl.String()
}
