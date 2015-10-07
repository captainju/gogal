package util

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/aws/aws-sdk-go/service/cloudfront/sign"
	"io/ioutil"
	"log"
	"time"
)

type CloudFrontManager struct {
	BaseUrl        string
	PrivateKeyFile string
	KeyId          string
	Expiration     int
	urlSigner      *sign.URLSigner
}

func (manager *CloudFrontManager) Init() error {
	// Read the private key
	pemData, err := ioutil.ReadFile(manager.PrivateKeyFile)
	if err != nil {
		log.Fatalf("read key file: %s", err)
		return err
	}

	// Extract the PEM-encoded data block
	block, _ := pem.Decode(pemData)
	if block == nil {
		log.Fatalf("bad key data: %s", "not PEM-encoded")
		return err
	}
	if got, want := block.Type, "RSA PRIVATE KEY"; got != want {
		log.Fatalf("unknown key type %q, want %q", got, want)
		return err
	}

	// Decode the RSA private key
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("bad private key: %s", err)
		return err
	}

	manager.urlSigner = sign.NewURLSigner(manager.KeyId, privKey)
	return nil
}

func (manager *CloudFrontManager) SignUrl(url string) (signedUrl string) {
	expiration := time.Now().Add(time.Duration(manager.Expiration) * time.Hour)
	signedURL, err := manager.urlSigner.Sign(url, expiration)
	if err != nil {
		log.Fatalf("Failed to sign url, err: %s\n", err.Error())
	}
	return signedURL
}
