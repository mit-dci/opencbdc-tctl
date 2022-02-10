package http

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
)

func writeSelfSignedCert() error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{"US"},
			OrganizationalUnit: []string{"Dev"},
			Province:           []string{"MA"},
			Locality:           []string{"Boston"},
			Organization:       []string{"OpenCBDC Test Controller"},
			CommonName:         "opencbdc-tct.dev.local",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 3650),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&priv.PublicKey,
		priv,
	)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %v", err)
	}

	cert, err := os.OpenFile(
		filepath.Join(common.DataDir(), "certs/server.crt"),
		os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open certificate file: %v", err)
	}
	err = pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("failed to encode certificate: %v", err)
	}
	cert.Close()

	key, err := os.OpenFile(
		filepath.Join(common.DataDir(), "certs/server.key"),
		os.O_WRONLY|os.O_CREATE,
		0600,
	)
	if err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	err = pem.Encode(
		key,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to encode certificate: %v", err)
	}

	return nil
}
