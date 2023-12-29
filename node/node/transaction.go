package node

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

/*
The ForeignClient represents a Client with limited informations.

The node has complete informations only about the clients that registered through it.
Clientes from other node must be shared with this node and the other nodes in the
network and it's shared over the gRPC server.

In reason of this, the node must register its own clients as foreign clients to be
accessible for all the nodes in the network.

A Client can be easily converted to a ForeignClient with the method `MakeForeign`
*/
type ForeignClient struct {
	*Node
	ClientId    string `json:"client_id"`
	NodeAddress string `json:"node"`
	Address     string `json:"address"`
}

// (Over)Writes the foreign client state in backlog using the current in-memory state
func (c ForeignClient) SyncWithBacklog() error {
	clientBytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal the client: %v", err)
	}

	var client map[string]interface{}
	if err := json.Unmarshal(clientBytes, &client); err != nil {
		return fmt.Errorf("failed to unmarshal the client into map: %v", err)
	}

	err = c.IndexDocument("clients", c.ClientId, client)
	if err != nil {
		return fmt.Errorf("failed to overwrite the client document: %v", err)
	}

	return nil
}

/*
A transaction is a operation between two clients in the same node or not. One client
occupies the place of sender (or signer), who performs the transaction,
and the other occupies the place of recipient.

To sign a transaction, you must to do it in the node of the sender/signer, once the transaction
is signed by the private key and the key pair is only available at the node that owns the client
credentials.

Every transaction has a value, a timestamp and a signature. The transaction is
signed only when it's included in a valid block. Please, go to `blockchain` to see
more about blocks.

The transaction can be converted into a byte array, a marshalling of the following information:
sender client id, recipient client id, value and timestamp. The signature is not included
in the marshalling process.
*/
type Transaction struct {
	TransactionId string         // A unique and universal id that references the transaction anywhere
	Sender        *Client        // The client who performed the transaction
	Recipient     *ForeignClient // The target client of the transaction (it belongs to the local node or to an external node)
	Value         float64        // The value is the current content of the transaction. It could be changed to a message or another content type
	Timestamp     int64          // The timestamp that records when the transaction was performed
	Signature     *string        // A pointer to the signature made by the sender client when the transaction have been accepted
}

// (Over)Writes the transaction state in backlog using the current in-memory state
func (t Transaction) SyncWithBacklog() error {
	transBytes, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal the client: %v", err)
	}

	var transaction map[string]interface{}
	if err := json.Unmarshal(transBytes, &transaction); err != nil {
		return fmt.Errorf("failed to unmarshal the client into map: %v", err)
	}

	err = t.Sender.IndexDocument("transactions", t.TransactionId, transaction)
	if err != nil {
		return fmt.Errorf("failed to overwrite the client document: %v", err)
	}

	return nil
}

// Converts the transaction  information to a encryptable byte array
func (t Transaction) ToBytes() []byte {
	transaction := map[string]interface{}{
		"sender":    t.Sender.ClientId,
		"recipient": t.Recipient.ClientId,
		"value":     t.Value,
		"timestamp": t.Timestamp,
	}

	transBytes, _ := json.Marshal(transaction)
	return transBytes
}

// Signs the transaction and updates the transaction record in backlog with the new signature
func (t *Transaction) SignTransaction() error {
	signature := t.Sender.CreateSignature(t)
	t.Signature = &signature

	err := t.SyncWithBacklog()
	if err != nil {
		return err
	}

	return nil
}

// Creates a new transaction from the client as its sender
func (c Client) NewTransaction(rcp string, value float64) *Transaction {
	transactionId, _ := uuid.NewUUID()
	sender := &c
	recipient, err := c.Node.RetrieveForeignClient(rcp)
	timestamp := time.Now().Unix()

	if err != nil {
		return nil
	}

	transaction := Transaction{
		TransactionId: transactionId.String(),
		Sender:        sender,
		Recipient:     recipient,
		Value:         value,
		Timestamp:     timestamp,
		Signature:     nil,
	}

	return &transaction
}
