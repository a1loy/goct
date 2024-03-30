package ctlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	ct "github.com/google/certificate-transparency-go"
	client "github.com/google/certificate-transparency-go/client"
	"github.com/google/certificate-transparency-go/jsonclient"
)

const (
	LogLookupBatchSize = 100
)

type CtLogClient struct {
	LogURI  string
	client  client.LogClient
	context context.Context
}

func NewLogClient(logURI string) (*client.LogClient, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConnsPerHost:   10,
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	opts := jsonclient.Options{}
	return client.New(logURI, httpClient, opts)
}

func NewCtLogClient(logURI string) (*CtLogClient, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConnsPerHost:   10,
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	opts := jsonclient.Options{}
	logClient, err := client.New(logURI, httpClient, opts)
	if err != nil {
		return &CtLogClient{}, err
	}
	ctx := context.TODO()
	return &CtLogClient{LogURI: logURI, client: *logClient, context: ctx}, nil
}

func (c CtLogClient) PrintState() error {
	return c.writeState(os.Stdout)
}

func (c CtLogClient) GetState() (string, error) {
	b := bytes.NewBufferString("")
	err := c.writeState(b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (c CtLogClient) writeState(w io.Writer) error {
	sth, err := c.client.GetSTH(c.context)
	if err != nil {
		return err
	}
	when := ct.TimestampToTime(sth.Timestamp)
	// treeSize := int64(sth.TreeSize)
	fmt.Fprintf(w, "%v (timestamp %d): Got STH for %v log (size=%d) at %v, hash %x\n",
		when, sth.Timestamp, sth.Version, sth.TreeSize, c.client.BaseURI(), sth.SHA256RootHash)
	fmt.Fprintf(w, "%v", signatureToString(&sth.TreeHeadSignature))
	return nil
}

func (c CtLogClient) GetTreeSize() int64 {
	sth, err := c.client.GetSTH(c.context)
	if err != nil {
		return -1
	}
	// when := ct.TimestampToTime(sth.Timestamp)
	treeSize := int64(sth.TreeSize)
	return treeSize
}

func (c CtLogClient) GetEntries(start, end int64) (*ct.GetEntriesResponse, error) {
	return c.client.GetRawEntries(c.context, start, end)
}

func signatureToString(signed *ct.DigitallySigned) string {
	return fmt.Sprintf("Signature: Hash=%v Sign=%v Value=%x",
		signed.Algorithm.Hash, signed.Algorithm.Signature, signed.Signature)
}
