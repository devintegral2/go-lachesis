package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/Fantom-foundation/go-lachesis/src/common"
)

// CreateCert create new cert & key for peer while initialize.
func CreateCert(key *common.PrivateKey, host, certPath string) ([]byte, error) {
	priv := (*ecdsa.PrivateKey)(key)

	notBefore := time.Now()

	// TODO: Setup it
	validFor := 365 * 24 * time.Hour

	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{""}, // TODO: Fill it later
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
		return nil, err
	}

	os.MkdirAll(certPath, os.ModePerm)

	certOut, err := os.Create(certPath + "cert_" + host + ".pem")
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}

	pemBlock := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}

	if err := pem.Encode(certOut, pemBlock); err != nil {
		log.Fatalf("failed to write data to cert.pem: %s", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("error closing cert.pem: %s", err)
	}
	log.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile(certPath+"key_"+host+".pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return nil, err
	}

	keyBlock, err := KeyToPemBlock(priv)
	if err != nil {
		log.Print("failed to process key -> pem block:", err)
		return nil, err
	}

	if err := pem.Encode(keyOut, keyBlock); err != nil {
		log.Fatalf("failed to write data to key.pem: %s", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("error closing key.pem: %s", err)
	}
	log.Print("wrote key.pem\n")

	return pem.EncodeToMemory(pemBlock), nil
}
