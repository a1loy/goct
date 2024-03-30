package matcher

import (
	"strings"

	"goct/internal/config"

	metrics "github.com/adrg/strutil/metrics"
	ct "github.com/google/certificate-transparency-go"
	scanner "github.com/google/certificate-transparency-go/scanner"
	ctx509 "github.com/google/certificate-transparency-go/x509"
)

type CustomCertMatcherBySimilarity struct {
	Patterns        []string
	CAWhitelist     map[string]bool
	IsSimilarDomain SimilarityFunc
}

type SimilarityFunc func(string, string) bool

func buildSimilarityFunc(cfg config.SimilarityCheckCfg) SimilarityFunc {
	return func(domain, pattern string) bool {
		metric := metrics.NewSmithWatermanGotoh()
		return metric.Compare(domain, pattern) > cfg.Distance || strings.HasSuffix(domain, pattern)
	}
}

func NewCustomCertMatcherBySimilarity(patterns []string, cfg config.SimilarityCheckCfg) (scanner.Matcher, error) {
	whitelist := make(map[string]bool, 0)
	return CustomCertMatcherBySimilarity{
		Patterns:        patterns,
		CAWhitelist:     whitelist,
		IsSimilarDomain: buildSimilarityFunc(cfg),
	}, nil
}

func (m CustomCertMatcherBySimilarity) CertificateMatches(c *ctx509.Certificate) bool {
	for _, p := range m.Patterns {
		if m.IsSimilarDomain(c.Subject.CommonName, p) {
			return !m.CAWhitelist[c.Issuer.CommonName]
		}
		for _, alt := range c.DNSNames {
			if m.IsSimilarDomain(alt, p) {
				return !m.CAWhitelist[c.Issuer.CommonName]
			}
		}
	}
	return false
}

func (m CustomCertMatcherBySimilarity) PrecertificateMatches(precert *ct.Precertificate) bool {
	for _, p := range m.Patterns {
		if m.IsSimilarDomain(precert.TBSCertificate.Subject.CommonName, p) {
			return !m.CAWhitelist[precert.TBSCertificate.Issuer.CommonName]
		}
		for _, alt := range precert.TBSCertificate.DNSNames {
			if m.IsSimilarDomain(alt, p) {
				return !m.CAWhitelist[precert.TBSCertificate.Issuer.CommonName]
			}
		}
	}
	return false
}
