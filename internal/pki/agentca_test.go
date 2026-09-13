package pki

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"
)

func TestIssueAgentCertificate(t *testing.T) {
	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	certPEM, keyPEM, fingerprint, expiry, err := m.Issue("agent-1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint == "" || expiry.Before(time.Now()) {
		t.Fatal("missing certificate metadata")
	}
	pair, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "agent-1" {
		t.Fatalf("wrong identity %s", cert.Subject.CommonName)
	}
	caPEM, err := os.ReadFile(m.CertPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(caPEM)
	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	if _, err = cert.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		t.Fatal(err)
	}
}
