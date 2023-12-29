package node

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	backlog "node/backlog"
	client "node/client"
	"os"

	"github.com/google/uuid"
)

type NodeStatus string

const (
	NodeAlive       NodeStatus = "alive"       // When the program starts
	NodeHibernating NodeStatus = "hibernating" // When te program ends
	NodeLiquidated  NodeStatus = "liquidated"  // When the node is destroyed
)

/*
The node represents some server in the network that's running meander. This type is the entrypoint
to perform any action in the server, such as create a client or a transaction.

A backlog is directly associated within a node. Please go to `backlog.go` to see more about this.

The node also has several clients that registered through it. The actions that can be performed
must have a client as their owner, as like the client must have a node as its owner.

The node data are registered in the backlog in a document identified by the node address hash.
Therefore, a node can be created from the host public address (for the first time) or can be retrieved
from the backlog by the same address hash.

With this, the node should be understand as a abstraction of the Backlog, since it just creates a
handling layer to connect the clients and the other server resources around the Elastic Search database.
*/
type Node struct {
	*backlog.Backlog `json:"-"`
	Mirror           string     `json:"syncer"`  // The host address from some peer that serves as mirror
	Host             string     `json:"host"`    // The host address from the current node server
	Version          string     `json:"version"` // Identifier of the source code that's running on the current node server
	Status           NodeStatus `json:"status"`  // The status of the meander
}

const nodeVersion string = "2023-12-26"

var BasePath = os.Getenv("BASE_PATH")

// Creates a new node struct since the local host
func NewLocalNode(syncer string) *Node {
	host, err := getLocalAddress()

	if err != nil {
		log.Fatalf("Failed to find the host: %v", err)
	}

	backlog := backlog.NewBacklog()
	node := Node{
		Backlog: backlog,
		Mirror:  syncer,
		Host:    host,
		Version: nodeVersion,
		Status:  NodeAlive,
	}

	return &node
}

// Creates a new node struct since the node stored in local elastic
func GetLocalNode() *Node {
	host, err := getLocalAddress()
	if err != nil {
		log.Fatalf("Failed to find the host: %v", err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(host))
	hash := hex.EncodeToString(hasher.Sum(nil))

	backlog := backlog.NewBacklog()
	nodeData, err := backlog.GetDocument("node", hash)
	if err != nil {
		log.Fatalf("Failed to get the node elastic document: %v", err)
	}

	node := Node{
		Backlog: backlog,
		Mirror:  nodeData["syncer"].(string),
		Host:    nodeData["host"].(string),
		Status:  NodeStatus(nodeData["status"].(string)),
		Version: nodeData["version"].(string),
	}

	return &node
}

// (Over)Writes the node state in local elastic using the current in-memory node state
func (n Node) SyncWithElastic(nodeIndex string) error {
	hasher := sha256.New()
	hasher.Write([]byte(n.Host))
	hash := hex.EncodeToString(hasher.Sum(nil))

	nodeBytes, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal the current node: %v", err)
	}
	var node map[string]interface{}
	if err := json.Unmarshal(nodeBytes, &node); err != nil {
		return fmt.Errorf("failed to unmarshal the current node into map: %v", err)
	}

	err = n.Backlog.IndexDocument(nodeIndex, hash, node)
	if err != nil {
		return fmt.Errorf("failed to overwrite the node document: %v", err)
	}

	return nil
}

// Sends node start signal to local elastic
func (n *Node) Attach() {
	n.Status = NodeAlive
	n.SyncWithElastic("peers")
	n.SyncWithElastic("node")
}

// Sends node end signal to local elastic
func (n *Node) Dettach() {
	n.Status = NodeHibernating
	n.SyncWithElastic("peers")
	n.SyncWithElastic("node")
}

// Sends node destroying signal to local elastic
func (n *Node) Liquidate() {
	n.Status = NodeLiquidated
	n.SyncWithElastic("peers")
	n.SyncWithElastic("node")
}

// Creates a new client in the node
func (n Node) NewLocalClient(alias, address, secret, password string) *Client {
	nodeHasher := sha256.New()
	nodeHasher.Write([]byte(n.Host))
	nodeHash := hex.EncodeToString(nodeHasher.Sum(nil))

	addrHasher := sha256.New()
	addrHasher.Write([]byte(address))
	addrHash := hex.EncodeToString(addrHasher.Sum(nil))

	pwdHasher := sha256.New()
	addrHasher.Write([]byte(password))
	pwdHash := hex.EncodeToString(pwdHasher.Sum(nil))

	uuid, _ := uuid.NewUUID()
	accountId := generateAccountId()

	client := Client{
		Node:        &n,
		UID:         uuid.String(),
		AccountId:   accountId,
		Alias:       alias,
		NodeAddress: nodeHash,
		Address:     addrHash,
		Secret:      secret,
		Password:    pwdHash,
	}

	if _, err := os.Stat(fmt.Sprintf("%s/%s", BasePath, uuid.String())); os.IsNotExist(err) {
		os.Mkdir(fmt.Sprintf("%s/%s", BasePath, uuid.String()), 0755)
	}

	client.GenerateCrypto()
	client.ClientId = client.Identity()
	client.PublicKey = string(client.ImpersonatePublicKey())
	client.PrivateKey = string(client.ImpersonatePrivateKey())
	cache := client.CreateCache()

	clientBytes, _ := json.Marshal(client)

	var clientData map[string]interface{}
	if err := json.Unmarshal(clientBytes, &clientData); err != nil {
		log.Fatalf("Failed to unmarshal client data: %v", err)
	}

	cacheBytes, _ := json.Marshal(cache)
	var cacheData map[string]interface{}
	if err := json.Unmarshal(cacheBytes, &cacheData); err != nil {
		log.Fatalf("Failed to unmarshal client data: %v", err)
	}

	n.Backlog.IndexDocument("clients", uuid.String(), clientData)
	n.Backlog.IndexDocument("cache", uuid.String(), cacheData)

	return &client
}

// Manually builds a client in the node with existing informations
func (n Node) RememberClient(uid, accountId, alias, address, secret, password string) (*Client, client.Cache) {
	nodeHasher := sha256.New()
	nodeHasher.Write([]byte(n.Host))
	nodeHash := hex.EncodeToString(nodeHasher.Sum(nil))

	addrHasher := sha256.New()
	addrHasher.Write(([]byte(address)))
	addrHash := hex.EncodeToString(addrHasher.Sum(nil))

	client := Client{
		UID:         uid,
		AccountId:   accountId,
		Alias:       alias,
		NodeAddress: nodeHash,
		Address:     addrHash,
		Secret:      secret,
		Password:    password,
	}

	client.RetrieveCrypto()
	client.ClientId = client.Identity()
	client.PublicKey = string(client.ImpersonatePublicKey())
	client.PrivateKey = string(client.ImpersonatePrivateKey())
	cache := client.CreateCache()

	cacheBytes, _ := json.Marshal(cache)
	var cacheData map[string]interface{}
	if err := json.Unmarshal(cacheBytes, &cacheData); err != nil {
		log.Fatalf("Failed to unmarshal client data: %v", err)
	}

	n.Backlog.IndexDocument("cache", uid, cacheData)

	return &client, cache
}
