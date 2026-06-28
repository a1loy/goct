package models

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const issuanceDateLayout = "2006-01-02 15:04:05 MST"

// detectMsgMarkdownTemplate renders a DetectMsg for Telegram MarkdownV2.
//
// Note the leading space inside every code span (e.g. "``` {{.CN}} ```"): the
// first token straight after an opening "```" is parsed as the code block's
// language tag and dropped, so without the space the first word of each value
// disappears (this is why the issuance date's day used to go missing). The
// date is rendered via IssuanceDateString so the template never emits
// time.Time's noisy default form. The [Entry] line is emitted only when Entry
// is non-empty, which lets a check suppress it by leaving Entry blank (see
// config IncludeEntry).
const detectMsgMarkdownTemplate = "[Check] ``` {{.Name}} ```\r\n" +
	"[CN] ``` {{.CN}} ```\r\n" +
	"[Hash] ``` {{.Hash}} ```\r\n" +
	"[Issued by] ``` {{.IssuerName}} ```\r\n" +
	"[Issued at] ``` {{.IssuanceDateString}} ```\r\n" +
	"[AltNames] ``` {{.DNSNames}} ```\r\n" +
	"{{if .Entry}}[Entry] ``` {{.Entry}} ```\r\n{{end}}"

var detectMsgTemplate = template.Must(template.New("OneMsg").Parse(detectMsgMarkdownTemplate))

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

// IssuanceDateString formats IssuanceDate for display in the report template.
func (m *DetectMsg) IssuanceDateString() string {
	return m.IssuanceDate.Format(issuanceDateLayout)
}

func (m *DetectMsg) ToMarkdownString() string {
	var tpl bytes.Buffer
	if err := detectMsgTemplate.Execute(&tpl, m); err != nil {
		panic(err)
	}
	return tpl.String()
}
