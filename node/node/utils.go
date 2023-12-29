package node

import (
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func getLocalAddress() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func generateAccountId() string {
	const size = 15
	rand.Seed(time.Now().UnixNano())

	baseID := make([]int, 20)
	for i := range baseID {
		baseID[i] = rand.Intn(41) + 5
	}

	var rawAccountIDBuilder strings.Builder
	for i := 0; i < len(baseID); i += 2 {
		pairSum := baseID[i]
		if i+1 < len(baseID) {
			pairSum += baseID[i+1]
		}
		rawAccountIDBuilder.WriteString(strconv.Itoa(pairSum))
	}
	rawAccountID := rawAccountIDBuilder.String()

	var accountID string
	switch {
	case len(rawAccountID) > size:
		accountID = rawAccountID[:size]
	case len(rawAccountID) < size:
		residual := size - len(rawAccountID)
		point0 := 0
		if residual != 1 {
			point0 = int(math.Pow10(residual - 1))
		}
		pointf := int(math.Pow10(residual)) - 1
		accountID = rawAccountID + strconv.Itoa(rand.Intn(pointf-point0+1)+point0)
	default:
		accountID = rawAccountID
	}

	return accountID
}
