package blockchain

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func parseHexAddress(address string) (common.Address, error) {
	normalizedAddress := strings.TrimSpace(address)
	if !common.IsHexAddress(normalizedAddress) {
		return common.Address{}, fmt.Errorf("invalid address: %s", address)
	}

	return common.HexToAddress(normalizedAddress), nil
}
