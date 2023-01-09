// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package core

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/common/hexutil"
	"github.com/dominant-strategies/go-quai/common/math"
	"github.com/dominant-strategies/go-quai/params"
)

var _ = (*genesisSpecMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (g Genesis) MarshalJSON() ([]byte, error) {
	type Genesis struct {
		Config     *params.ChainConfig                         `json:"config"`
		Nonce      math.HexOrDecimal64                         `json:"nonce"`
		Timestamp  math.HexOrDecimal64                         `json:"timestamp"`
		ExtraData  hexutil.Bytes                               `json:"extraData"`
		GasLimit   []math.HexOrDecimal64                         `json:"gasLimit"   gencodec:"required"`
		Difficulty []*math.HexOrDecimal256                       `json:"difficulty" gencodec:"required"`
		Mixhash    common.Hash                                 `json:"mixHash"`
		Coinbase   []common.Address                              `json:"coinbase"`
		Alloc      map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Number     []math.HexOrDecimal64                         `json:"number"`
		GasUsed    []math.HexOrDecimal64                         `json:"gasUsed"`
		ParentHash []common.Hash                                 `json:"parentHash"`
		BaseFee    []*math.HexOrDecimal256                       `json:"baseFeePerGas"`
	}
	var enc Genesis
	enc.Config = g.Config
	enc.Nonce = math.HexOrDecimal64(g.Nonce)
	enc.Timestamp = math.HexOrDecimal64(g.Timestamp)
	enc.ExtraData = g.ExtraData
	enc.Mixhash = g.Mixhash
	enc.GasLimit = make([]math.HexOrDecimal64, common.HierarchyDepth)
	enc.Difficulty = make([]*math.HexOrDecimal256, common.HierarchyDepth)
	enc.Coinbase = make([]common.Address, common.HierarchyDepth)
	enc.Number = make([]math.HexOrDecimal64, common.HierarchyDepth)
	enc.GasUsed = make([]math.HexOrDecimal64, common.HierarchyDepth)
	enc.ParentHash = make([]common.Hash, common.HierarchyDepth)
	enc.BaseFee = make([]*math.HexOrDecimal256, common.HierarchyDepth)
	if g.Alloc != nil {
		enc.Alloc = make(map[common.UnprefixedAddress]GenesisAccount, len(g.Alloc))
		for k, v := range g.Alloc {
			internal, err := k.InternalAddress()
			if err != nil {
				return nil, err
			}
			enc.Alloc[common.UnprefixedAddress(*internal)] = v
		}
	}
	for i := 0; i < common.HierarchyDepth; i++ {
		enc.GasLimit[i] = math.HexOrDecimal64(g.GasLimit[i])
		enc.Difficulty[i] = (*math.HexOrDecimal256)(g.Difficulty[i])
		enc.Coinbase[i] = g.Coinbase[i]
		enc.Number[i] = math.HexOrDecimal64(g.Number[i])
		enc.GasUsed[i] = math.HexOrDecimal64(g.GasUsed[i])
		enc.ParentHash[i] = g.ParentHash[i]
		enc.BaseFee[i] = (*math.HexOrDecimal256)(g.BaseFee[i])
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (g *Genesis) UnmarshalJSON(input []byte) error {
	type Genesis struct {
		Config     *params.ChainConfig                         `json:"config"`
		Nonce      *math.HexOrDecimal64                        `json:"nonce"`
		Timestamp  *math.HexOrDecimal64                        `json:"timestamp"`
		ExtraData  *hexutil.Bytes                              `json:"extraData"`
		GasLimit   []*math.HexOrDecimal64                        `json:"gasLimit"   gencodec:"required"`
		Difficulty []*math.HexOrDecimal256                       `json:"difficulty" gencodec:"required"`
		Mixhash    *common.Hash                                `json:"mixHash"`
		Coinbase   []*common.Address                             `json:"coinbase"`
		Alloc      map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Number     []*math.HexOrDecimal64                        `json:"number"`
		GasUsed    []*math.HexOrDecimal64                        `json:"gasUsed"`
		ParentHash []*common.Hash                                `json:"parentHash"`
		BaseFee    []*math.HexOrDecimal256                       `json:"baseFeePerGas"`
	}
	var dec Genesis
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Config != nil {
		g.Config = dec.Config
	}
	if dec.Nonce != nil {
		g.Nonce = uint64(*dec.Nonce)
	}
	if dec.Timestamp != nil {
		g.Timestamp = uint64(*dec.Timestamp)
	}
	if dec.ExtraData != nil {
		g.ExtraData = *dec.ExtraData
	}
	if dec.Mixhash != nil {
		g.Mixhash = *dec.Mixhash
	}
	if dec.Alloc == nil {
		return errors.New("missing required field 'alloc' for Genesis")
	}
	g.Alloc = make(GenesisAlloc, len(dec.Alloc))
	for k, v := range dec.Alloc {
		internal := common.InternalAddress(k)
		g.Alloc[common.NewAddressFromData(&internal)] = v
	}

	for i := 0; i < common.HierarchyDepth; i++ {
	if dec.GasLimit[i] == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	g.GasLimit[i] = uint64(*dec.GasLimit[i])
	if dec.Difficulty[i] == nil {
		return errors.New("missing required field 'difficulty' for Genesis")
	}
	g.Difficulty[i] = (*big.Int)(dec.Difficulty[i])
	if dec.Coinbase[i] != nil {
		g.Coinbase[i] = *dec.Coinbase[i]
	}
	if dec.Number[i] != nil {
		g.Number[i] = uint64(*dec.Number[i])
	}
	if dec.GasUsed[i] != nil {
		g.GasUsed[i] = uint64(*dec.GasUsed[i])
	}
	if dec.ParentHash[i] != nil {
		g.ParentHash[i] = *dec.ParentHash[i]
	}
	if dec.BaseFee[i] != nil {
		g.BaseFee[i] = (*big.Int)(dec.BaseFee[i])
	}
}
	return nil
}
