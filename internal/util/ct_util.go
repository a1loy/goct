package util

import (
	"crypto/x509"

	ct "github.com/google/certificate-transparency-go"
	ctx509 "github.com/google/certificate-transparency-go/x509"
)

// TODO: rename to ReturnErrorIfInvalid ?
func IsValidLeaf(entry *ct.LogEntry) (*x509.Certificate, error) {
	var cert *x509.Certificate
	if entry.X509Cert == nil {
		_, parseErr := ctx509.ParseCertificate(entry.Precert.Submitted.Data)
		if parseErr != nil {
			return cert, parseErr
		}
	}
	return x509.ParseCertificate(entry.Precert.Submitted.Data)
}

// TODO: rename to ReturnErrorIfInvalid ?
func IsValidCert(data []byte) (*x509.Certificate, error) {
	_, err := ctx509.ParseCertificate(data)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(data)
}
