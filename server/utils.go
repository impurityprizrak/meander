package pb

import (
	"crypto/subtle"
	"fmt"
	backlog "node/backlog"
	client "node/client"
)

func compareDigest(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func validateToken(uid, secret, token string) bool {
	privateKey, err := client.DownloadPrivateKey(secret, uid)

	if err != nil {
		fmt.Printf("failed to download private key: %v\n", err)
		return false
	}

	publicKey, err := client.DownloadPublicKey(uid)

	if err != nil {
		fmt.Printf("failed to download public key: %v\n", err)
		return false
	}

	crypto := client.CryptoResource{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	payload, err := crypto.DecryptToken(token)
	if err != nil {
		fmt.Printf("failed to decrypt the token: %v\n", err)
		return false
	}

	backlog := backlog.NewBacklog()
	cache, err := backlog.GetDocument("cache", uid)
	if err != nil {
		fmt.Printf("failed to get cache document: %v\n", err)
		return false
	}

	matchA := compareDigest(
		[]byte(cache["computed_key_a"].(string)),
		[]byte(payload["computed_key_a"].(string)),
	)

	matchP := compareDigest(
		[]byte(cache["computed_key_p"].(string)),
		[]byte(payload["computed_key_p"].(string)),
	)

	return matchA && matchP

}
