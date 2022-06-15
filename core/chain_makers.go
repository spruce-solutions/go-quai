// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/spruce-solutions/go-quai/common"
	"github.com/spruce-solutions/go-quai/consensus"
	"github.com/spruce-solutions/go-quai/consensus/blake3"
	"github.com/spruce-solutions/go-quai/consensus/misc"
	"github.com/spruce-solutions/go-quai/core/state"
	"github.com/spruce-solutions/go-quai/core/types"
	"github.com/spruce-solutions/go-quai/core/vm"
	"github.com/spruce-solutions/go-quai/ethdb"
	"github.com/spruce-solutions/go-quai/params"
)

// BlockGen creates blocks for testing.
// See GenerateChain for a detailed explanation.
type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header

	config *params.ChainConfig
	engine consensus.Engine
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (b *BlockGen) SetCoinbase(addr common.Address) {
	if b.gasPool != nil {
		if len(b.txs) > 0 {
			panic("coinbase must be set before adding transactions")
		}
		panic("coinbase can only be set once")
	}
	b.header.Coinbase[types.QuaiNetworkContext] = addr
	b.gasPool = new(GasPool).AddGas(b.header.GasLimit[types.QuaiNetworkContext])
}

// SetExtra sets the extra data field of the generated block.
func (b *BlockGen) SetExtra(data []byte) {
	b.header.Extra[types.QuaiNetworkContext] = data
}

// SetNonce sets the nonce field of the generated block.
func (b *BlockGen) SetNonce(nonce types.BlockNonce) {
	b.header.Nonce = nonce
}

// SetDifficulty sets the difficulty field of the generated block. This method is
// useful for Clique tests where the difficulty does not depend on time. For the
// blake3 tests, please use OffsetTime, which implicitly recalculates the diff.
func (b *BlockGen) SetDifficulty(diff *big.Int) {
	b.header.Difficulty[types.QuaiNetworkContext] = diff
}

// AddTx adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTx panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. Notably, contract code relying on the BLOCKHASH instruction
// will panic during execution.
func (b *BlockGen) AddTx(tx *types.Transaction) {
	b.AddTxWithChain(nil, tx)
}

// AddTxWithChain adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTxWithChain panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. If contract code relies on the BLOCKHASH instruction,
// the block in chain will be returned.
func (b *BlockGen) AddTxWithChain(bc *BlockChain, tx *types.Transaction) {
	if b.gasPool == nil {
		b.SetCoinbase(common.Address{})
	}
	b.statedb.Prepare(tx.Hash(), len(b.txs))
	receipt, err := ApplyTransaction(b.config, bc, &b.header.Coinbase[types.QuaiNetworkContext], b.gasPool, b.statedb, b.header, tx, &b.header.GasUsed[types.QuaiNetworkContext], vm.Config{})
	if err != nil {
		panic(err)
	}
	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

// GetBalance returns the balance of the given address at the generated block.
func (b *BlockGen) GetBalance(addr common.Address) *big.Int {
	return b.statedb.GetBalance(addr)
}

// AddUncheckedTx forcefully adds a transaction to the block without any
// validation.
//
// AddUncheckedTx will cause consensus failures when used during real
// chain processing. This is best used in conjunction with raw block insertion.
func (b *BlockGen) AddUncheckedTx(tx *types.Transaction) {
	b.txs = append(b.txs, tx)
}

// Number returns the block number of the block being generated.
func (b *BlockGen) Number() *big.Int {
	return new(big.Int).Set(b.header.Number[types.QuaiNetworkContext])
}

// BaseFee returns the EIP-1559 base fee of the block being generated.
func (b *BlockGen) BaseFee() *big.Int {
	return new(big.Int).Set(b.header.BaseFee[types.QuaiNetworkContext])
}

// AddUncheckedReceipt forcefully adds a receipts to the block without a
// backing transaction.
//
// AddUncheckedReceipt will cause consensus failures when used during real
// chain processing. This is best used in conjunction with raw block insertion.
func (b *BlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
	b.receipts = append(b.receipts, receipt)
}

// TxNonce returns the next valid transaction nonce for the
// account at addr. It panics if the account does not exist.
func (b *BlockGen) TxNonce(addr common.Address) uint64 {
	if !b.statedb.Exist(addr) {
		panic("account does not exist")
	}
	return b.statedb.GetNonce(addr)
}

// AddUncle adds an uncle header to the generated block.
func (b *BlockGen) AddUncle(h *types.Header) {
	b.uncles = append(b.uncles, h)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
// For index -1, PrevBlock returns the parent block given to GenerateChain.
func (b *BlockGen) PrevBlock(index int) *types.Block {
	if index >= b.i {
		panic(fmt.Errorf("block index %d out of range (%d,%d)", index, -1, b.i))
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

// OffsetTime modifies the time instance of a block, implicitly changing its
// associated difficulty. It's useful to test scenarios where forking is not
// tied to chain length directly.
func (b *BlockGen) OffsetTime(seconds int64) {
	b.header.Time += uint64(seconds)
	if b.header.Time <= b.parent.Header().Time {
		panic("block time out of range")
	}
	chainreader := &fakeChainReader{config: b.config}
	b.header.Difficulty[types.QuaiNetworkContext] = b.engine.CalcDifficulty(chainreader, b.header.Time, b.parent.Header(), types.QuaiNetworkContext)
}

// GenerateChain creates a chain of n blocks. The first block's
// parent will be the provided parent. db is used to store
// intermediate states and should contain the parent's state trie.
//
// The generator function is called with a new block generator for
// every block. Any transactions and uncles added to the generator
// become part of the block. If gen is nil, the blocks will be empty
// and their coinbase will be the zero address.
//
// Blocks created by GenerateChain do not contain valid proof of work
// values. Inserting them into BlockChain requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(config *params.ChainConfig, parent *types.Block, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	if config.ChainID == nil { // for testing purposes
		config = params.TestChainConfig
	}

	if config == nil {
		config = params.TestChainConfig
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	chainreader := &fakeChainReader{config: config}
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{i: i, chain: blocks, parent: parent, statedb: statedb, config: config, engine: engine}
		// b.header = makeHeader(chainreader, parent, statedb, b.engine)

		// Execute any user modifications to the block
		if gen != nil {
			gen(i, b)
		}
		if b.engine != nil {
			// Finalize and seal the block
			block, _ := b.engine.FinalizeAndAssemble(chainreader, b.header, statedb, b.txs, b.uncles, b.receipts)

			// Write state changes to db
			root, err := statedb.Commit(config.IsEIP158(b.header.Number[types.QuaiNetworkContext]))
			if err != nil {
				panic(fmt.Sprintf("state write error: %v", err))
			}
			if err := statedb.Database().TrieDB().Commit(root, false, nil); err != nil {
				panic(fmt.Sprintf("trie write error: %v", err))
			}
			return block, b.receipts
		}
		return nil, nil
	}
	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db), nil)
		if err != nil {
			panic(err)
		}
		block, receipt := genblock(i, parent, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}

func makeHeader(config *params.ChainConfig, chain consensus.ChainReader, parent *types.Block, order int, state *state.StateDB, engine consensus.Engine) *types.Header {
	// return genesis block as itself for first block in chain
	if parent.Header().Number[0].Cmp(big.NewInt(0)) == 0 {
		return parent.Header()
	}

	// initialization of header
	baseFee := misc.CalcBaseFee(chain.Config(), parent.Header(), chain.GetHeaderByNumber, chain.GetUnclesInChain, chain.GetGasUsedInChain)
	header := &types.Header{
		ParentHash:        make([]common.Hash, 3),
		Number:            []*big.Int{new(big.Int).SetUint64(0), new(big.Int).SetUint64(0), new(big.Int).SetUint64(0)},
		Extra:             make([][]byte, 3),
		Time:              uint64(0),
		BaseFee:           []*big.Int{baseFee, baseFee, baseFee},
		GasLimit:          make([]uint64, 3),
		Coinbase:          make([]common.Address, 3),
		Difficulty:        make([]*big.Int, 3),
		NetworkDifficulty: make([]*big.Int, 3),
		Root:              make([]common.Hash, 3),
		TxHash:            make([]common.Hash, 3),
		ReceiptHash:       make([]common.Hash, 3),
		GasUsed:           make([]uint64, 3),
		Bloom:             make([]types.Bloom, 3),
		Location:          chain.Config().Location,
	}

	// same across orders
	header.Time = parent.Time() + uint64(10)
	gasLimit := CalcGasLimit(parent.GasLimit(), parent.GasUsed(), len(parent.Uncles()))
	header.GasLimit = []uint64{gasLimit, gasLimit, gasLimit}

	switch order {
	case 2: // Zone
		// fill ParentHash values
		header.ParentHash[0] = parent.Header().ParentHash[0]
		header.ParentHash[1] = parent.Header().ParentHash[1]
		header.ParentHash[2] = parent.Hash()
		// fill Number values
		header.Number[0] = parent.Header().Number[0]
		header.Number[1] = parent.Header().Number[1]
		header.Number[2].Add(parent.Header().Number[2], common.Big1)
		// fill Difficulty values
		header.Difficulty[0] = parent.Header().Difficulty[0]
		header.Difficulty[1] = parent.Header().Difficulty[1]
		header.Difficulty[2] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 2)
	case 1: // Region
		// fill ParentHash values
		header.ParentHash[0] = parent.Header().ParentHash[0]
		header.ParentHash[1] = parent.Hash()
		header.ParentHash[2] = header.ParentHash[1]
		// fill Number values
		header.Number[0] = parent.Header().Number[0]
		header.Number[1].Add(parent.Header().Number[1], common.Big1)
		header.Number[2].Add(parent.Header().Number[2], common.Big1)
		// fill Difficulty values
		header.Difficulty[0] = parent.Header().Difficulty[0]
		header.Difficulty[1] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 1)
		header.Difficulty[2] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 2)
	case 0: // Prime
		// fill ParentHash values
		header.ParentHash[0] = parent.Hash()
		header.ParentHash[1] = header.ParentHash[0] // take new header[0] value to reduce compute
		header.ParentHash[2] = header.ParentHash[0]
		// fill Number values
		header.Number[0].Add(parent.Header().Number[0], common.Big1)
		header.Number[1].Add(parent.Header().Number[1], common.Big1)
		header.Number[2].Add(parent.Header().Number[2], common.Big1)
		// fill Difficulty values
		header.Difficulty[0] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 0)
		header.Difficulty[1] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 1)
		header.Difficulty[2] = blake3.CalcDifficulty(config, header.Time, parent.Header(), 2)
	}

	return header
}

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.Header, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, engine, db, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

// makeBlockChain creates a deterministic chain of blocks rooted at parent.
func makeBlockChain(parent *types.Block, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(params.TestChainConfig, parent, engine, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}

// struct for notation
type blockGenSpec struct {
	numbers    [3]int    // prime idx, region idx, zone idx
	parentTags [3]string // (optionally) Override the parents to point to tagged blocks. Empty strings are ignored.
	tag        string    // (optionally) Give this block a named tag. Empty strings are ignored.
}

// Generate blocks to form a network of chains
func GenerateNetworkBlocks(graph [3][3][]*blockGenSpec) ([]*types.Block, error) {
	return nil, errors.New("Not implemented")
}

type fakeChainReader struct {
	config *params.ChainConfig
}

// Config returns the chain configuration.
func (cr *fakeChainReader) Config() *params.ChainConfig {
	return cr.config
}

func (cr *fakeChainReader) CurrentHeader() *types.Header                            { return nil }
func (cr *fakeChainReader) GetHeaderByNumber(number uint64) *types.Header           { return nil }
func (cr *fakeChainReader) GetHeaderByHash(hash common.Hash) *types.Header          { return nil }
func (cr *fakeChainReader) GetHeader(hash common.Hash, number uint64) *types.Header { return nil }
func (cr *fakeChainReader) GetBlock(hash common.Hash, number uint64) *types.Block   { return nil }
func (cr *fakeChainReader) GetExternalBlock(hash common.Hash, number uint64, location []byte, context uint64) (*types.ExternalBlock, error) {
	return nil, nil
}
func (cr *fakeChainReader) QueueAndRetrieveExtBlocks(blocks []*types.ExternalBlock, header *types.Header) []*types.ExternalBlock {
	return nil
}

func (cr *fakeChainReader) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	return nil
}
func (cr *fakeChainReader) GetGasUsedInChain(block *types.Block, length int) int64 { return 0 }

func (cr *fakeChainReader) GetExternalBlocks(header *types.Header) ([]*types.ExternalBlock, error) {
	return nil, nil
}
func (cr *fakeChainReader) GetLinkExternalBlocks(header *types.Header) ([]*types.ExternalBlock, error) {
	return nil, nil
}

// GenerateNetwork creates a network of n blocks in c contexts. The first block's
// parent will be the provided parent. db is used to store intermediate states
// and should contain the parent's state trie.

// configs should contain the respective config settings being tested. This must
// include a valid slice of Prime, Region, and Zone mining locations. This is
// necessary for testing interchain linkages through coincident blocks as well
// as transactions flowing through them. The array of ints in c represent
// the intended order of contexts to be mined.

// The generator function is called with a new block generator for every
// block. Any transactions and uncles added to the generator become part of the
// block. If gen is nil, the blocks will be empty and their coinbase will
// be the zero address.

// Blocks created by GenerateNetwork do not contain valid proof of work values.
// Inserting them into BlockChain requires use of a TestPoW in order to test and
// verify the correct operation of the Proof-of-Work algorithm.
func GenerateNetwork(primeConfig *params.ChainConfig, regionConfig params.ChainConfig, zoneConfig params.ChainConfig, parent *types.Block, orders []int, startNumber []int, engine consensus.Engine, db ethdb.Database, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	// the first config must be the Prime
	if primeConfig != params.MainnetPrimeChainConfig {
		print("must have MainnetPrimeChainConfig as first config")
	}
	// check second config for Region
	if !(regionConfig.ChainID.Cmp(params.MainnetRegionChainConfigs[0].ChainID) == 0 || regionConfig.ChainID.Cmp(params.MainnetRegionChainConfigs[1].ChainID) == 0 || regionConfig.ChainID.Cmp(params.MainnetRegionChainConfigs[2].ChainID) == 0) {
		print("must have a MainnetRegionChainConfig as second config")
	}
	// check third config for Zone in Region
	if !(zoneConfig.ChainID.Cmp(params.MainnetZoneChainConfigs[regionConfig.Location[0]-1][0].ChainID) == 0 || zoneConfig.ChainID.Cmp(params.MainnetZoneChainConfigs[regionConfig.Location[0]-1][1].ChainID) == 0 || zoneConfig.ChainID.Cmp(params.MainnetZoneChainConfigs[regionConfig.Location[0]-1][2].ChainID) == 0) {
		print("must have a MainnetZoneConfig as third config")
	}

	n := len(orders)
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)

	// associate chainreaders and configs for correct block context placement
	chainreaders, configs := []fakeChainReader{}, []params.ChainConfig{}
	for _, context := range orders {
		switch context {
		case 0:
			chainreaders = append(chainreaders, fakeChainReader{config: primeConfig})
			configs = append(configs, *primeConfig)
		case 1:
			chainreaders = append(chainreaders, fakeChainReader{config: &regionConfig})
			configs = append(configs, regionConfig)
		case 2:
			chainreaders = append(chainreaders, fakeChainReader{config: &zoneConfig})
			configs = append(configs, zoneConfig)
		default:
			print("contexts must be 0, 1, or 2")
		}
	}

	genblock := func(i int, parent *types.Block, context int, statedb *state.StateDB, chainreaders []fakeChainReader, configs []params.ChainConfig) (*types.Block, types.Receipts) {
		// select appropriate fakeChainReader for block creation
		chainreader := chainreaders[i]
		config := configs[i]

		b := &BlockGen{i: i, chain: blocks, parent: parent, statedb: statedb, config: &config, engine: engine}
		b.header = makeHeader(&config, &chainreader, parent, context, statedb, b.engine)

		// Execute any user modifications to the block
		if gen != nil {
			gen(i, b)
		}

		if b.engine != nil {
			// Finalize and seal the block
			block, _ := b.engine.FinalizeAndAssemble(&chainreader, b.header, statedb, b.txs, b.uncles, b.receipts)

			// Write state changes to db
			root, err := statedb.Commit(config.IsEIP158(b.header.Number[types.QuaiNetworkContext]))
			if err != nil {
				panic(fmt.Sprintf("state write error: %v", err))
			}
			if err := statedb.Database().TrieDB().Commit(root, false, nil); err != nil {
				panic(fmt.Sprintf("trie write error: %v", err))
			}
			return block, b.receipts
		}
		return nil, nil
	}

	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db), nil)
		if err != nil {
			panic(err)
		}
		block, receipt := genblock(i, parent, orders[i], statedb, chainreaders, configs)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}
