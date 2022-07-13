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

var (
	big2e256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)) // 2^256
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
// ethash tests, please use OffsetTime, which implicitly recalculates the diff.
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
	if config == nil {
		config = params.TestChainConfig
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	chainreader := &fakeChainReader{config: config}
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{i: i, chain: blocks, parent: parent, statedb: statedb, config: config, engine: engine}
		b.header = makeHeader(chainreader, parent, statedb, b.engine)

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

func makeHeader(chain consensus.ChainReader, parent *types.Block, state *state.StateDB, engine consensus.Engine) *types.Header {
	var time uint64
	if parent.Time() == 0 {
		time = 10
	} else {
		time = parent.Time() + 10 // block time is fixed at 10 seconds
	}

	baseFee := misc.CalcBaseFee(chain.Config(), parent.Header(), chain.GetHeaderByNumber, chain.GetUnclesInChain, chain.GetGasUsedInChain)

	header := &types.Header{
		Coinbase:    []common.Address{common.Address{}, common.Address{}, common.Address{}},
		Number:      []*big.Int{big.NewInt(int64(1)), big.NewInt(int64(1)), big.NewInt(int64(1))},
		ParentHash:  []common.Hash{common.Hash{}, common.Hash{}, common.Hash{}},
		Root:        []common.Hash{common.Hash{}, common.Hash{}, common.Hash{}},
		Difficulty:  []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1)},
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       []types.Bloom{types.Bloom{}, types.Bloom{}, types.Bloom{}},
		Time:        time,
		BaseFee:     []*big.Int{baseFee, baseFee, baseFee},
		GasLimit:    []uint64{0, 0, 0},
		GasUsed:     []uint64{0, 0, 0},
		Extra:       [][]byte{[]byte(nil), []byte(nil), []byte(nil)},
	}
	header.GasLimit[types.QuaiNetworkContext] = parent.GasLimit()

	parentHeader := &types.Header{
		Number:     []*big.Int{big.NewInt(int64(1)), big.NewInt(int64(1)), big.NewInt(int64(1))},
		Difficulty: []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1)},
		UncleHash:  types.EmptyUncleHash,
		Time:       time - 10,
	}

	if chain.Config().IsLondon(header.Number[types.QuaiNetworkContext]) {
		if !chain.Config().IsLondon(parent.Number()) {
			parentGasLimit := parent.GasLimit() * params.ElasticityMultiplier
			header.GasLimit[types.QuaiNetworkContext] = CalcGasLimit(parentGasLimit, parent.GasUsed(), 0)
		}
	}

	parentHeader.Number[types.QuaiNetworkContext] = new(big.Int).Add(parent.Number(), common.Big1)
	parentHeader.Difficulty[types.QuaiNetworkContext] = parent.Difficulty()
	parentHeader.UncleHash[types.QuaiNetworkContext] = parent.UncleHash()

	header.Root[types.QuaiNetworkContext] = state.IntermediateRoot(chain.Config().IsEIP158(parent.Number()))
	header.ParentHash[types.QuaiNetworkContext] = parent.Hash()
	header.Coinbase[types.QuaiNetworkContext] = parent.Coinbase()
	header.Difficulty[types.QuaiNetworkContext] = engine.CalcDifficulty(chain, time, parentHeader, types.QuaiNetworkContext)
	header.Number[types.QuaiNetworkContext] = new(big.Int).Add(parent.Number(), common.Big1)

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

func GenerateBlock(genesis *types.Block, parents []*types.Block, location []byte, tag string) (*types.Block, error) {
	// Must always have a zone parent.
	if parents[params.ZONE] == nil {
		return nil, fmt.Errorf("Missing zone parent for block %s", tag)
	}
	// If has prime parent, must also have region parent.
	if parents[params.PRIME] != nil && parents[params.REGION] == nil {
		return nil, fmt.Errorf("Prime block is missing region parent for block %s", tag)
	}
	// Compute any relative values
	time := []uint64{0, 0, 0}
	number := make([]*big.Int, 3, 3)
	parentHash := make([]common.Hash, 3, 3)
	gasLimit := []uint64{0, 0, 0}
	gasUsed := []uint64{0, 0, 0}
	baseFee := make([]*big.Int, 3, 3)
	for ctx, parent := range parents {
		if parent != nil {
			number[ctx] = new(big.Int).Add(parent.Number(ctx), big.NewInt(1))
			parentHash[ctx] = parent.Hash()
			gasLimit[ctx] = parent.GasLimit(ctx)
			gasUsed[ctx] = parent.GasUsed(ctx)
			baseFee[ctx] = parent.BaseFee(ctx)
			time[ctx] = parent.Header().Time + 10
		} else {
			number[ctx] = big.NewInt(0)
		}
	}
	// Determine the order we need to mine to
	desiredOrder := -1
	for ctx, parent := range parents {
		if parent != nil {
			desiredOrder = ctx
			break
		}
	}
	// Build the header
	header := types.Header{
		Coinbase:    []common.Address{common.Address{}, common.Address{}, common.Address{}},
		Number:      number,
		ParentHash:  parentHash,
		Root:        types.EmptyRootHash,
		Difficulty:  params.FakeDifficulty,
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       []types.Bloom{types.Bloom{}, types.Bloom{}, types.Bloom{}},
		Time:        time[desiredOrder],
		BaseFee:     baseFee,
		GasLimit:    gasLimit,
		GasUsed:     gasUsed,
		Extra:       [][]byte{[]byte(nil), []byte(nil), []byte(nil)},
		Location:    location,
	}
	// Mine the block
	// This is similar to the conventional mining loop, but modified to find a block at _exactly_
	// the specified order, instead of _at least_ the specified order.
	engine := blake3.NewFaker()
	// Loop until you find a block of the correct order
	nonce := 0
	for {
		// Set the new nonce and try again
		header.Nonce = types.EncodeNonce(uint64(nonce))
		actual_order, err := engine.GetDifficultyOrder(&header)
		if err != nil {
			return nil, fmt.Errorf("Failed to get block order: ", err)
		}
		if actual_order == desiredOrder {
			return types.NewBlockWithHeader(&header).WithBody([]*types.Transaction{}, []*types.Header{}), nil
		}
		nonce += 1
	}
}

// Generate blocks to form a network of chains
func GenerateNetworkBlocks(genesisBlock *types.Block, graph [3][3][]*types.BlockGenSpec) (map[string]types.Block, error) {
	// To generate a network, we need to generate individual chains and link the chains together (via coincidents & parent references)

	// Track the position of each generator in the various chains
	var position [3][3]int
	blocks := make(map[string]types.Block)
	blocks["gen"] = *genesisBlock

	// Loop infinitely until we've generated all blocks in all locations
	for {
		blocksGenerated := 0
		for r, regionSpecs := range graph {
			for z, zoneSpecs := range regionSpecs {
				if zoneSpecs == nil {
					// No blocks to generate in this zone
					continue
				}
				i := position[r][z]
				if i >= len(zoneSpecs) {
					// Reached end of this zone's blocks
					continue
				}
				if spec := zoneSpecs[i]; spec != nil {
					// Make sure this block tag is unique, and not already in use
					if _, exists := blocks[spec.Tag]; exists {
						return nil, fmt.Errorf("duplicate tag found: %s", spec.Tag)
					}
					// Look up the parent blocks
					parents := make([]*types.Block, len(spec.Parent), cap(spec.Parent))
					for ctx, tag := range spec.Parent {
						if tag != "" {
							parent, exists := blocks[tag]
							if exists {
								parents[ctx] = &parent
							} else {
								// If one or more of the parents is not available yet, stop generating blocks in this region.
								// The necessary parent must exist in one of the other regions, so we need to skip to the next
								// iteration of this loop to start generating blocks in other zones and regions. By the time
								// we come back to this zone, we hopefully will have generated the requisit parents.
								// As long as the requisit parents are tagged somewhere, this loop will eventually complete.
								continue
							}
						}
					}
					// Generate the block and add it to the map
					block, err := GenerateBlock(genesisBlock, parents, []byte{byte(r) + 1, byte(z) + 1}, spec.Tag)
					if err != nil {
						return nil, err
					}
					blocks[spec.Tag] = *block
					// Increment the position for this chain's generator
					position[r][z] += 1
					// Record that we've done some work this iteration
					blocksGenerated += 1
				}
			}
		}
		// If we ran through the entire network graph without generating a single block, then we've completed generating the graph
		if blocksGenerated == 0 {
			delete(blocks, "gen") // Do not include genesis block in final map
			return blocks, nil
		}
	}
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
func (cr *fakeChainReader) GetExternalBlock(hash common.Hash, location []byte, context uint64) (*types.ExternalBlock, error) {
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

// CheckContext checks to make sure the range of a context or order is valid
func (cr *fakeChainReader) CheckContext(context int) error {
	if context < 0 || context > len(params.FullerOntology) {
		return errors.New("the provided path is outside the allowable range")
	}
	return nil
}

// CheckLocationRange checks to make sure the range of r and z are valid
func (cr *fakeChainReader) CheckLocationRange(location []byte) error {
	if int(location[0]) < 1 || int(location[0]) > params.FullerOntology[0] {
		return errors.New("the provided location is outside the allowable region range")
	}
	if int(location[1]) < 1 || int(location[1]) > params.FullerOntology[1] {
		return errors.New("the provided location is outside the allowable zone range")
	}
	return nil
}
