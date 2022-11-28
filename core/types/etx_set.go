package types

import (
	"fmt"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/log"
)

const (
	EtxExpirationAge = 8640 // With 10s blocks, ETX expire after ~24hrs
)

// The EtxSet maps an ETX hash to the ETX and block number in which it became available.
// If no entry exists for a given ETX hash, then that ETX is not available.
type EtxSet map[common.Hash]*EtxSetEntry

type EtxSetEntry struct {
	Height uint64
	ETX    *Transaction
}

func NewEtxSet() EtxSet {
	return make(EtxSet)
}

// updateInboundEtxs updates the set of inbound ETXs available to be mined into
// a block in this location. This method adds any new ETXs to the set and
// removes expired ETXs.
func (set EtxSet) Update(newInboundEtxs Transactions, currentHeight uint64) {
	if numNewEtxs := len(newInboundEtxs); numNewEtxs > 0 {
		fmt.Printf("_____INBOUND::::| added %n ETXs to ETX set\n", numNewEtxs)
	}
	// Add new ETX entries to the inbound set
	for _, etx := range newInboundEtxs {
		if etx.ToChain().Equal(common.NodeLocation) {
			set[etx.Hash()] = &EtxSetEntry{currentHeight, etx}
		} else {
			log.Error("skipping ETX belonging to other destination", "etxHash: ", etx.Hash(), "etxToChain: ", etx.ToChain())
		}
	}

	// Remove expired ETXs
	for txHash, entry := range set {
		availableAtBlock := entry.Height
		etxExpirationHeight := availableAtBlock + EtxExpirationAge
		if currentHeight > etxExpirationHeight {
			delete(set, txHash)
		}
	}
}
