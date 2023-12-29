package node

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	client "node/client"
	"time"
)

/*
A collection of credentials from a unique user that signed into a node

Whenever a new client is created, the created crypto is uploaded to the client path
and the credentials are synced in local elastic
*/
type Client struct {
	*client.CryptoResource `json:"-"`
	*Node                  `json:"-"`
	UID                    string `json:"uid"`        // A random hexadecimal id that's a internal reference along the node (no peers know the UID)
	Alias                  string `json:"alias"`      // The nickname chosen to connect the client
	AccountId              string `json:"account_id"` // A private random id composed by digits only
	NodeAddress            string `json:"node"`       // The hash of the host address from the node where the client has been registered
	Address                string `json:"address"`    // The hash of the host address from where the client was registered
	ClientId               string `json:"client_id"`  // The identification generated by the public key (external reference known by all the peers)
	PublicKey              string `json:"-"`          // RSA public key (result of ImpersonatePublicKey method)
	PrivateKey             string `json:"-"`          // RSA private key used to assign the client transactions (result of ImpersonatePrivateKey method)
	Secret                 string `json:"-"`          // The password that protects the private key in the node filesystem
	Password               string `json:"password"`   // The hex hash from the password chosen together with the alias to connect the client
}

func (c Client) CreateCache() client.Cache {
	cka := client.GenerateComputedKeyA(c.AccountId)

	hasher := sha256.Sum256([]byte(c.Password))
	hash := int(binary.BigEndian.Uint64(hasher[:8]))

	ckp := client.GenerateComputedKeyP(hash)

	cache := client.Cache{
		ComputedKeyA: cka,
		ComputedKeyP: ckp,
		Timestamp:    time.Now().Unix(),
		Alias:        c.Alias,
		Password:     c.Password,
		PublicKey:    c.ImpersonatePublicKey(),
	}

	return cache
}

// (Over)Writes the client state in local elastic using the current in-memory state
func (c Client) SyncWithElastic(nodeIndex string) error {
	nodeBytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal the current node: %v", err)
	}

	var client map[string]interface{}
	if err := json.Unmarshal(nodeBytes, &client); err != nil {
		return fmt.Errorf("failed to unmarshal the current node into map: %v", err)
	}

	err = c.Backlog.IndexDocument(nodeIndex, c.ClientId, client)
	if err != nil {
		return fmt.Errorf("failed to overwrite the node document: %v", err)
	}

	return nil
}

// Retrieve the existing RSA key pair for the client and keep in-memory
func (c *Client) RetrieveCrypto() {
	private, err := client.DownloadPrivateKey(c.Secret, c.UID)

	if err != nil {
		log.Fatalf("failed to download private key: %v", err)
	}

	public, err := client.DownloadPublicKey(c.UID)

	if err != nil {
		log.Fatalf("failed to download public key: %v", err)
	}

	crypto := client.CryptoResource{
		PrivateKey: private,
		PublicKey:  public,
	}

	c.CryptoResource = &crypto
}

// Generate a new RSA key pair for the client and upload it
func (c *Client) GenerateCrypto() {
	crypto, err := client.NewCryptoResource()

	if err != nil {
		log.Fatalf("failed to create a new crypto resource: %v", err)
	}

	c.CryptoResource = crypto

	err = c.UploadPrivateKey(c.Secret, c.UID)
	if err != nil {
		log.Fatalf("failed to upload private key: %v", err)
	}

	err = c.UploadPublicKey(c.UID)
	if err != nil {
		log.Fatalf("failed to upload public key: %v", err)
	}
}
