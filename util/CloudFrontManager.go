package util

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/aws/aws-sdk-go/service/cloudfront/sign"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type CloudFrontManager struct {
	BaseUrl        string
	PrivateKeyFile string
	KeyId          string
	Expiration     int
	urlSigner      *sign.URLSigner
	key            *rsa.PrivateKey
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

	manager.key = privKey

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

func (manager *CloudFrontManager) WriteCookies(w http.ResponseWriter, domain string) {
	expiration := time.Now().Add(time.Duration(manager.Expiration) * time.Hour)

	p := sign.NewCannedPolicy("*", expiration)

	b64Signature, b64Policy, err := p.Sign(manager.key)
	if err != nil {
		log.Fatalf("Failed to sign policy, err: %s\n", err.Error())
	}

	http.SetCookie(w, &http.Cookie{HttpOnly: true, Domain: domain, Name: "CloudFront-Policy", Value: string(b64Policy)})
	http.SetCookie(w, &http.Cookie{HttpOnly: true, Domain: domain, Name: "CloudFront-Signature", Value: string(b64Signature)})
	http.SetCookie(w, &http.Cookie{HttpOnly: true, Domain: domain, Name: "CloudFront-Key-Pair-Id", Value: manager.KeyId})
}
