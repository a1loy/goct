package matcher

import (
	"goct/internal/config"
	"goct/internal/util"

	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
	ctx509 "github.com/google/certificate-transparency-go/x509"
)

type CustomCertMatcherByValidity struct {
	// Regexs      *[]*regexp.Regexp
	CAWhitelist map[string]bool
}

func NewCustomCertMatcherByValidity(checkCfg config.CheckConfig) (scanner.Matcher, error) {
	whitelist := make(map[string]bool, 0)

	return CustomCertMatcherByValidity{
		// Regexs: regexs,
		CAWhitelist: whitelist,
	}, nil
}

func (m CustomCertMatcherByValidity) CertificateMatches(c *ctx509.Certificate) bool {
	_, err := util.IsValidCert(c.RawTBSCertificate)

	return err != nil && !m.CAWhitelist[c.Issuer.CommonName]
}

func (m CustomCertMatcherByValidity) PrecertificateMatches(p *ct.Precertificate) bool {
	return false
}
