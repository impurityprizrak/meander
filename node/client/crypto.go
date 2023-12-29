package node

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

var BasePath string = os.Getenv("BASE_PATH")

/*
Every client has a pair of private and public key to assign the transactions.

The Crypto Resource can be independently generated and attached to a Client.
Whenever you create a new CryptoResource, a new RSA private and public key
are generated, without associations with the Client until it's did.

The method of CryptoResource creation and attachment to a Client is equal to the
method of creation and attachment of the Backlog to a Node. This resource can be
generated and used at any point of the code, including to regenerate a Client pair of keys.
*/
type CryptoResource struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type Signable interface {
	ToBytes() []byte
}

func NewCryptoResource() (*CryptoResource, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)

	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	publicKey := &privateKey.PublicKey

	crypto := CryptoResource{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	return &crypto, nil
}

// This is a identifier based on the public key. It's used to represent the client in its transactions
func (c CryptoResource) Identity() string {
	derPkix, err := x509.MarshalPKIXPublicKey(c.PublicKey)
	if err != nil {
		log.Fatalf("failed do generate the crypto identity: %v", err)
		return ""
	}

	hexString := hex.EncodeToString(derPkix)
	return hexString
}

// Assigns the client transactions using the private key. The signature grants that the transaction was included in a valid block.
func (c CryptoResource) CreateSignature(t Signable) string {
	hasher := sha256.New()
	hasher.Write(t.ToBytes())
	hashed := hasher.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, c.PrivateKey, crypto.SHA256, hashed)

	if err != nil {
		log.Fatalf("Failed to create signature: %v\n", err)
	}

	return string(signature)
}

// Converts the private key to a byte array and, eventually, a string
func (c CryptoResource) ImpersonatePrivateKey() []byte {
	pemPrivate := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(c.PrivateKey),
		},
	)

	return pemPrivate
}

// Converts the public key to a byte array and, eventually, a string
func (c CryptoResource) ImpersonatePublicKey() []byte {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(c.PublicKey)
	if err != nil {
		log.Fatalf("Failed to convert public key: %v\n", err)
	}

	pemPublic := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyBytes,
		},
	)

	return pemPublic
}

// Writes the byte array from private key to an I/O stream
func (c CryptoResource) UploadPrivateKey(secret string, uid string) error {
	privBytes, err := x509.MarshalPKCS8PrivateKey(c.PrivateKey)
	if err != nil {
		return err
	}

	block, err := x509.EncryptPEMBlock(
		rand.Reader,
		"ENCRYPTED PRIVATE KEY",
		privBytes,
		[]byte(secret),
		x509.PEMCipherAES256,
	)
	if err != nil {
		return err
	}

	file, err := os.Create(fmt.Sprintf("%s/%s/private.pem", os.Getenv("BASE_PATH"), uid))
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

// Writes the byte array from public key to an I/O stream
func (c CryptoResource) UploadPublicKey(uid string) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(c.PublicKey)
	if err != nil {
		return err
	}

	file, err := os.Create(fmt.Sprintf("%s/%s/public.pem", os.Getenv("BASE_PATH"), uid))
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
}

// Converts the byte array from a I/O stream to a private key
func DownloadPrivateKey(secret string, uid string) (*rsa.PrivateKey, error) {
	file, err := os.ReadFile(fmt.Sprintf("%s/%s/private.pem", os.Getenv("BASE_PATH"), uid))

	if err != nil {
		return nil, fmt.Errorf("failed to read file private.pem: %v", err)
	}

	block, _ := pem.Decode(file)

	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM bytes")
	}

	decryptedBytes, err := x509.DecryptPEMBlock(block, []byte(secret))

	if err != nil {
		return nil, fmt.Errorf("failed to decrypt pem block: %v", err)
	}

	priv, err := x509.ParsePKCS8PrivateKey(decryptedBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze RSA private key: %v", err)
	}

	privateKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unknown private key type")
	}

	return privateKey, nil
}

// Converts the byte array from a I/O stream to a public key
func DownloadPublicKey(uid string) (*rsa.PublicKey, error) {
	file, err := os.ReadFile(fmt.Sprintf("%s/%s/public.pem", os.Getenv("BASE_PATH"), uid))
	if err != nil {
		return nil, fmt.Errorf("failed to read file public.pem: %v", err)
	}

	block, _ := pem.Decode(file)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM bytes")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze RSA public key: %v", err)
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unknown public key type")
	}

	return publicKey, nil
}
