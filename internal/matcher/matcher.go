package matcher

import (
	"regexp"

	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
	ctx509 "github.com/google/certificate-transparency-go/x509"
)

type CustomCertMatcherByRegex struct {
	Regexs      *[]*regexp.Regexp
	CAWhitelist map[string]bool
}

func NewCustomCertMatcherByRegex(regexs *[]*regexp.Regexp) (scanner.Matcher, error) {
	whitelist := make(map[string]bool, 0)

	return CustomCertMatcherByRegex{
		Regexs: regexs, CAWhitelist: whitelist,
	}, nil
}

func (m CustomCertMatcherByRegex) CertificateMatches(cert *ctx509.Certificate) bool {
	for _, regex := range *m.Regexs {
		if regex.FindStringIndex(cert.Subject.CommonName) != nil {
			return !m.CAWhitelist[cert.Issuer.CommonName]
		}
		for _, alt := range cert.DNSNames {
			if regex.FindStringIndex(alt) != nil {
				return !m.CAWhitelist[cert.Issuer.CommonName]
			}
		}
	}
	return false
}

func (m CustomCertMatcherByRegex) PrecertificateMatches(precert *ct.Precertificate) bool {
	for _, regex := range *m.Regexs {
		if regex.FindStringIndex(precert.TBSCertificate.Subject.CommonName) != nil {
			return !m.CAWhitelist[precert.TBSCertificate.Issuer.CommonName]
		}
		for _, alt := range precert.TBSCertificate.DNSNames {
			if regex.FindStringIndex(alt) != nil {
				return !m.CAWhitelist[precert.TBSCertificate.Issuer.CommonName]
			}
		}
	}
	return false
}
