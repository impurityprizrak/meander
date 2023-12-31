package pb

import (
	"context"
	"fmt"
	"net"
	backlog "node/backlog"
	client "node/client"
	node "node/node"
	"unicode"

	"google.golang.org/grpc/peer"
)

type MeanderServer struct {
	UnimplementedMeanderClientIOServer
}

func (s *MeanderServer) CreateClient(ctx context.Context, p *ClientPayload) (*Client, error) {
	if p.Alias == "" || p.Password == "" || p.Secret == "" {
		return nil, fmt.Errorf("create client request requires: alias, password, secret")
	}

	peer, ok := peer.FromContext(ctx)
	if !ok {
		err := fmt.Errorf("failed to get the peer from context")
		return nil, err
	}

	clientIP, _, err := net.SplitHostPort(peer.Addr.String())
	if err != nil {
		err := fmt.Errorf("failed to get host address from peer: %v", err)
		return nil, err
	}

	node := node.GetLocalNode()
	results, err := node.Backlog.FindDocument("local_clients", "alias", p.Alias)

	if err != nil {
		err := fmt.Errorf("failed to verify the existent document: %v", err)
		return nil, err
	}

	if len(results) > 0 {
		err := fmt.Errorf("invalid alias: the alias was found in this node")
		return nil, err
	}

	if isValid := func() bool {
		var hasMin, hasMaj, hasNum bool
		length := 0

		for _, char := range p.Password {
			switch {
			case unicode.IsLower(char):
				hasMin = true
			case unicode.IsUpper(char):
				hasMaj = true
			case unicode.IsDigit(char):
				hasNum = true
			}

			length++
		}

		return length >= 10 && hasMin && hasMaj && hasNum
	}(); !isValid {
		err := fmt.Errorf("invalid password: password must have at least 10 chars with major and minor letters and numbers")
		return nil, err
	}

	localClient := node.NewLocalClient(p.Alias, clientIP, p.Secret, p.Password)

	client := Client{
		Alias:   localClient.Alias,
		Node:    localClient.NodeAddress,
		Address: localClient.Address,
		UserId:  localClient.UID,
	}

	return &client, nil
}

func (s *MeanderServer) ConnectClient(ctx context.Context, p *ClientPayload) (*Connection, error) {
	node := node.GetLocalNode()
	results, err := node.Backlog.FindDocument("local_clients", "alias", p.Alias)

	if err != nil {
		err := fmt.Errorf("failed to verify the existent document: %v", err)
		return nil, err
	} else if len(results) == 0 {
		err := fmt.Errorf("not found: the alias was not found inside the server")
		return nil, err
	}

	client := results
	uid := client["_id"]

	localClient, cache := node.RetrieveClient(uid.(string), p.Secret)
	token, err := cache.Token()

	if err != nil {
		err := fmt.Errorf("could not generate token: %v", err)
		return nil, err
	}

	connection := Connection{
		UserId: localClient.UID,
		Token:  token,
	}

	return &connection, nil
}

func (s *MeanderServer) ValidateToken(ctx context.Context, p *ConnectionPayload) (*Commit, error) {
	uid := p.UserId
	secret := p.Secret
	privateKey, err := client.DownloadPrivateKey(secret, uid)

	if err != nil {
		return nil, fmt.Errorf("failed to download private key: %v", err)
	}

	publicKey, err := client.DownloadPublicKey(uid)

	if err != nil {
		return nil, fmt.Errorf("failed to download public key: %v", err)
	}

	crypto := client.CryptoResource{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	payload, err := crypto.DecryptToken(p.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the token: %v", err)
	}

	backlog := backlog.NewBacklog()
	cache, err := backlog.GetDocument("cache", uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache document: %v", err)
	}

	matchA := compareDigest(
		[]byte(cache["computed_key_a"].(string)),
		[]byte(payload["computed_key_a"].(string)),
	)

	if !matchA {
		err := "The computed key A doesn't match"
		return &Commit{
			Status: 1,
			Error:  &err,
		}, nil
	}

	matchP := compareDigest(
		[]byte(cache["computed_key_p"].(string)),
		[]byte(payload["computed_key_p"].(string)),
	)

	if !matchP {
		err := "The computed key P doesn't match"
		return &Commit{
			Status: 1,
			Error:  &err,
		}, nil
	}

	return &Commit{}, nil

}

// func (s *MeanderServer) RegisterClient(ctx context.Context, c *Client) (*Commit, error) {
// 	commit := Commit{}

// 	client := node.Client{
// 		Alias:       c.Alias,
// 		NodeAddress: c.Node,
// 		Address:     c.Address,
// 		ClientId:    c.ClientId,
// 		PublicKey:   c.PublicKey,
// 	}

// 	err := client.SyncWithElastic("clients")

// 	if err != nil {
// 		errStr := err.Error()
// 		commit.Status = 1
// 		commit.Error = &errStr
// 	} else {
// 		commit.Status = 0
// 	}

// 	return &commit, nil
// }

// func (s *MeanderServer) RegisterNode(ctx context.Context, n *Node) (*Commit, error) {
// 	commit := Commit{}

// 	node := node.Node{
// 		Mirror:  n.Syncer,
// 		Host:    n.Host,
// 		Version: n.Version,
// 		Status:  node.NodeStatus(n.Status),
// 	}

// 	err := node.SyncWithElastic("peers")

// 	if err != nil {
// 		errStr := err.Error()
// 		commit.Status = 1
// 		commit.Error = &errStr
// 	} else {
// 		commit.Status = 0
// 	}

// 	return &commit, nil
// }
