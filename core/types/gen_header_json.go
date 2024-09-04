// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/common/hexutil"
)

var _ = (*headerMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (h Header) MarshalJSON() ([]byte, error) {
	var enc struct {
		ParentHash   					[]common.Hash   		`json:"parentHash"         			gencodec:"required"`
		UncleHash    					common.Hash    			`json:"sha3Uncles"         			gencodec:"required"`
		EVMRoot      					common.Hash   			`json:"evmRoot"            			gencodec:"required"`
		UTXORoot		 				common.Hash	 			`json:"utxoRoot"           			gencodec:"required"`
		TxHash       					common.Hash   			`json:"transactionsRoot"   			gencodec:"required"`
		ReceiptHash  					common.Hash   			`json:"receiptsRoot"       			gencodec:"required"`
		EtxHash      					common.Hash   			`json:"extTransactionsRoot"			gencodec:"required"`
		EtxSetRoot    					common.Hash    			`json:"etxSetRoot"          		gencodec:"required"`
		EtxRollupHash					common.Hash   			`json:"extRollupRoot"      			gencodec:"required"`
		ManifestHash 					[]common.Hash  			`json:"manifestHash"       			gencodec:"required"`
		PrimeTerminus         			common.Hash           	`json:"primeTerminus"            	gencodec:"required"`
		InterlinkRootHash     			common.Hash           	`json:"interlinkRootHash"        	gencodec:"required"`
		ParentEntropy					[]*hexutil.Big 			`json:"parentEntropy"      			gencodec:"required"`
		ParentDeltaS 					[]*hexutil.Big 			`json:"parentDeltaS"       			gencodec:"required"`
		ParentUncledSubDeltaS  		 	[]*hexutil.Big 			`json:"parentUncledSubDeltaS"    	gencodec:"required"`
		EfficiencyScore 			 	hexutil.Uint64  		`json:"efficiencyScore"    			gencodec:"required"`
		ThresholdCount				 	hexutil.Uint64  		`json:"thresholdCount"    			gencodec:"required"`
		ExpansionNumber				 	hexutil.Uint64  		`json:"expansionNumber"    			gencodec:"required"`
		EtxEligibleSlices			 	common.Hash  		  	`json:"etxEligibleSlices" 			gencodec:"required"`
		UncledS							*hexutil.Big   			`json:"uncledS"            			gencodec:"required"`
		Number      					[]*hexutil.Big 			`json:"number"             			gencodec:"required"`
		GasLimit    					hexutil.Uint64		 	`json:"gasLimit"           			gencodec:"required"`
		GasUsed     					hexutil.Uint64		 	`json:"gasUsed"            			gencodec:"required"`
		BaseFee     					*hexutil.Big   		 	`json:"baseFeePerGas"      			gencodec:"required"`
		Extra       					hexutil.Bytes  		 	`json:"extraData"          			gencodec:"required"`
	}
	// Initialize the enc struct
	enc.ParentEntropy = make([]*hexutil.Big, common.HierarchyDepth)
	enc.ParentDeltaS = make([]*hexutil.Big, common.HierarchyDepth)
	enc.ParentUncledSubDeltaS = make([]*hexutil.Big, common.HierarchyDepth)
	enc.ParentHash = make([]common.Hash, common.HierarchyDepth-1)
	enc.Number = make([]*hexutil.Big, common.HierarchyDepth-1)

	copy(enc.ManifestHash, h.ManifestHashArray())
	for i := 0; i < common.HierarchyDepth; i++ {
		enc.ParentEntropy[i] = (*hexutil.Big)(h.ParentEntropy(i))
		enc.ParentDeltaS[i] = (*hexutil.Big)(h.ParentDeltaS(i))
		enc.ParentUncledSubDeltaS[i] = (*hexutil.Big)(h.ParentUncledSubDeltaS(i))
	}
	for i :=0 ; i< common.HierarchyDepth-1; i++ {
		enc.ParentHash[i] = h.ParentHash(i)
		enc.Number[i] = (*hexutil.Big)(h.Number(i))
	}
	enc.UncleHash = h.UncleHash()
	enc.EVMRoot = h.EVMRoot()
	enc.UTXORoot = h.UTXORoot()
	enc.TxHash = h.TxHash()
	enc.EtxHash = h.EtxHash()
	enc.EtxSetRoot = h.EtxSetRoot()
	enc.EtxRollupHash = h.EtxRollupHash()
	enc.ReceiptHash = h.ReceiptHash()
	enc.PrimeTerminus = h.PrimeTerminus()
	enc.InterlinkRootHash = h.InterlinkRootHash()
	enc.UncledS = (*hexutil.Big)(h.UncledS())
	enc.GasLimit = hexutil.Uint64(h.GasLimit())
	enc.GasUsed = hexutil.Uint64(h.GasUsed())
	enc.EfficiencyScore = hexutil.Uint64(h.EfficiencyScore())
	enc.ThresholdCount = hexutil.Uint64(h.ThresholdCount())
	enc.ExpansionNumber = hexutil.Uint64(h.ExpansionNumber())
	enc.EtxEligibleSlices = h.EtxEligibleSlices()
	enc.BaseFee = (*hexutil.Big)(h.BaseFee())
	enc.Extra = hexutil.Bytes(h.Extra())
	raw, err := json.Marshal(&enc)
	return raw, err
}

// UnmarshalJSON unmarshals from JSON.
func (h *Header) UnmarshalJSON(input []byte) error {
	var dec struct {
		ParentHash   					[]common.Hash   		`json:"parentHash"         			gencodec:"required"`
		UncleHash    					*common.Hash    		`json:"sha3Uncles"         			gencodec:"required"`
		EVMRoot      					*common.Hash   			`json:"evmRoot"            			gencodec:"required"`
		UTXORoot		 				*common.Hash	 		`json:"utxoRoot"           			gencodec:"required"`
		TxHash       					*common.Hash   			`json:"transactionsRoot"   			gencodec:"required"`
		ReceiptHash  					*common.Hash   			`json:"receiptsRoot"       			gencodec:"required"`
		EtxHash      					*common.Hash   			`json:"extTransactionsRoot"			gencodec:"required"`
		EtxSetRoot    					*common.Hash    		`json:"etxSetRoot"          		gencodec:"required"`
		EtxRollupHash					*common.Hash   			`json:"extRollupRoot"      			gencodec:"required"`
		ManifestHash 					[]common.Hash  			`json:"manifestHash"       			gencodec:"required"`
		PrimeTerminus         			*common.Hash           	`json:"primeTerminus"            	gencodec:"required"`
		InterlinkRootHash     			*common.Hash           	`json:"interlinkRootHash"        	gencodec:"required"`
		ParentEntropy					[]*hexutil.Big 			`json:"parentEntropy"      			gencodec:"required"`
		ParentDeltaS 					[]*hexutil.Big 			`json:"parentDeltaS"       			gencodec:"required"`
		ParentUncledSubDeltaS  		 	[]*hexutil.Big 			`json:"parentUncledSubDeltaS"    	gencodec:"required"`
		EfficiencyScore 			 	*hexutil.Uint64  		`json:"efficiencyScore"    			gencodec:"required"`
		ThresholdCount				 	*hexutil.Uint64  		`json:"thresholdCount"    			gencodec:"required"`
		ExpansionNumber				 	*hexutil.Uint64  		`json:"expansionNumber"    			gencodec:"required"`
		EtxEligibleSlices			 	*common.Hash  		  	`json:"etxEligibleSlices" 			gencodec:"required"`
		UncledS							*hexutil.Big   			`json:"uncledS"            			gencodec:"required"`
		Number      					[]*hexutil.Big 			`json:"number"             			gencodec:"required"`
		GasLimit    					*hexutil.Uint64		 	`json:"gasLimit"           			gencodec:"required"`
		GasUsed     					*hexutil.Uint64		 	`json:"gasUsed"            			gencodec:"required"`
		BaseFee     					*hexutil.Big   		 	`json:"baseFeePerGas"      			gencodec:"required"`
		Extra       					hexutil.Bytes  		 	`json:"extraData"          			gencodec:"required"`
	}
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for Header")
	}
	if dec.UncleHash == nil {
		return errors.New("missing required field 'sha3Uncles' for Header")
	}
	if dec.EVMRoot == nil {
		return errors.New("missing required field 'evmRoot' for Header")
	}
	if dec.UTXORoot == nil {
		return errors.New("missing required field 'utxoRoot' for Header")
	}
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	if dec.EtxHash == nil {
		return errors.New("missing required field 'extTransactionsRoot' for Header")
	}
	if dec.EtxSetRoot == nil {
		return errors.New("missing required field 'etxSetRoot' for Header")
	}
	if dec.EtxRollupHash == nil {
		return errors.New("missing required field 'extRollupRoot' for Header")
	}
	if dec.ManifestHash == nil {
		return errors.New("missing required field 'manifestHash' for Header")
	}
	if dec.ReceiptHash == nil {
		return errors.New("missing required field 'receiptsRoot' for Header")
	}
	if dec.PrimeTerminus == nil {
		return errors.New("missing required field 'primeTerminus' for Header")
	}
	if dec.InterlinkRootHash == nil {
		return errors.New("missing required field 'interlinkRootHash' for Header")
	}
	if dec.ParentEntropy == nil {
		return errors.New("missing required field 'parentEntropy' for Header")
	}
	if dec.ParentDeltaS == nil {
		return errors.New("missing required field 'parentDeltaS' for Header")
	}
	if dec.ParentUncledSubDeltaS == nil {
		return errors.New("missing required field 'parentUncledSubDeltaS' for Header")
	}
	if dec.UncledS == nil {
		return errors.New("missing required field 'uncledS' for Header")
	}
	if dec.Number == nil {
		return errors.New("missing required field 'number' for Header")
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	if dec.EfficiencyScore == nil {
		return errors.New("missing required field 'efficiencyScore' for Header")
	}
	if dec.ThresholdCount == nil {
		return errors.New("missing required field 'thresholdCount' for Header")
	}
	if dec.ExpansionNumber == nil {
		return errors.New("missing required field 'expansionNumber' for Header")
	}
	if dec.BaseFee == nil {
		return errors.New("missing required field 'baseFee' for Header")
	}
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	// Initialize the header
	h.parentHash = make([]common.Hash, common.HierarchyDepth-1)
	h.manifestHash = make([]common.Hash, common.HierarchyDepth)
	h.parentEntropy = make([]*big.Int, common.HierarchyDepth)
	h.parentDeltaS = make([]*big.Int, common.HierarchyDepth)
	h.parentUncledSubDeltaS = make([]*big.Int, common.HierarchyDepth)
	h.number = make([]*big.Int, common.HierarchyDepth-1)

	for i := 0; i < common.HierarchyDepth; i++ {
		h.SetManifestHash(dec.ManifestHash[i], i)
		if dec.ParentEntropy[i] == nil {
			return errors.New("missing required field 'parentEntropy' for Header")
		}
		h.SetParentEntropy((*big.Int)(dec.ParentEntropy[i]), i)
		if dec.ParentDeltaS[i] == nil {
			return errors.New("missing required field 'parentDeltaS' for Header")
		}
		h.SetParentDeltaS((*big.Int)(dec.ParentDeltaS[i]), i)
		if  dec.ParentUncledSubDeltaS[i] == nil {
			return errors.New("missing required field 'parentUncledDeltaS' for Header")
		}
		h.SetParentUncledSubDeltaS((*big.Int)(dec.ParentUncledSubDeltaS[i]), i)
	}

	for i := 0; i < common.HierarchyDepth-1; i++ {
		h.SetParentHash(dec.ParentHash[i], i)
		if dec.Number[i] == nil {
			return errors.New("missing required field 'number' for Header")
		}
		h.SetNumber((*big.Int)(dec.Number[i]), i)
	}

	h.SetUncleHash(*dec.UncleHash)
	h.SetEVMRoot(*dec.EVMRoot)
	h.SetUTXORoot(*dec.UTXORoot)
	h.SetTxHash(*dec.TxHash)
	h.SetReceiptHash(*dec.ReceiptHash)
	h.SetEtxHash(*dec.EtxHash)
	h.SetEtxSetRoot(*dec.EtxSetRoot)
	h.SetEtxRollupHash(*dec.EtxRollupHash)
	h.SetPrimeTerminus(*dec.PrimeTerminus)
	h.SetInterlinkRootHash(*dec.InterlinkRootHash)
	h.SetUncledS((*big.Int)(dec.UncledS))
	h.SetGasLimit(uint64(*dec.GasLimit))
	h.SetGasUsed(uint64(*dec.GasUsed))
	h.SetEfficiencyScore(uint16(*dec.EfficiencyScore))
	h.SetThresholdCount(uint16(*dec.ThresholdCount))
	h.SetExpansionNumber(uint8(*dec.ExpansionNumber))
	h.SetEtxEligibleSlices(*dec.EtxEligibleSlices)
	h.SetBaseFee((*big.Int)(dec.BaseFee))
	h.SetExtra(dec.Extra)
	return nil
}

func (t Termini) MarshalJSON() ([]byte, error) {
	var enc struct {
		DomTermini []common.Hash `json:"domTermini" gencodec:"required"`
		SubTermini []common.Hash `json:"subTermini"  gencodec:"required"`
	}
	copy(enc.SubTermini, t.SubTermini())
	copy(enc.DomTermini, t.DomTermini())
	raw, err := json.Marshal(&enc)
	return raw, err
}

func (t *Termini) UnmarshalJSON(input []byte) error {
	var dec struct {
		DomTermini []common.Hash `json:"domTermini" gencodec:"required"`
		SubTermini []common.Hash `json:"subTermini"  gencodec:"required"`
	}
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.DomTermini == nil {
		return errors.New("missing required field 'domTerminus' for Termini")
	}
	if dec.SubTermini == nil {
		return errors.New("missing required field 'subTermini' for Termini")
	}
	t.SetDomTermini(dec.DomTermini)
	t.SetSubTermini(dec.SubTermini)
	return nil
}

func (wh *WorkObjectHeader) MarshalJSON() ([]byte, error) {
	var enc struct {
		HeaderHash common.Hash    `json:"headerHash" gencoden:"required"`
		ParentHash common.Hash    `json:"parentHash" gencoden:"required"`
		Number     *hexutil.Big   `json:"number" gencoden:"required"`
		Difficulty *hexutil.Big   `json:"difficulty" gencoden:"required"`
		PrimeTerminusNumber *hexutil.Big `json:"primeTerminusNumber" gencoden:"required"`
		TxHash     common.Hash    `json:"txHash" gencoden:"required"`
		Location   hexutil.Bytes  `json:"location" gencoden:"required"`
		MixHash    common.Hash    `json:"mixHash" gencoden:"required"`
		Time       hexutil.Uint64 `json:"timestamp" gencoden:"required"`
		Nonce      BlockNonce     `json:"nonce" gencoden:"required"`
		Coinbase   common.Address `json:"coinbase" gencoden:"required"`
	}

	enc.HeaderHash = wh.HeaderHash()
	enc.Difficulty = (*hexutil.Big)(wh.Difficulty())
	enc.PrimeTerminusNumber = (*hexutil.Big)(wh.PrimeTerminusNumber())
	enc.Number = (*hexutil.Big)(wh.Number())
	enc.TxHash = wh.TxHash()
	enc.Location = hexutil.Bytes(wh.Location())
	enc.MixHash = wh.MixHash()
	enc.Time = hexutil.Uint64(wh.Time())
	enc.Nonce = wh.Nonce()
	enc.Coinbase = wh.Coinbase()

	raw, err := json.Marshal(&enc)
	return raw, err
}

func (wh *WorkObjectHeader) UnmarshalJSON(input []byte) error {
	var dec struct {
		HeaderHash common.Hash     `json:"headerHash" gencoden:"required"`
		ParentHash common.Hash     `json:"parentHash" gencoden:"required"`
		Number     *hexutil.Big    `json:"number" gencoden:"required"`
		Difficulty *hexutil.Big    `json:"difficulty" gencoden:"required"`
		PrimeTerminusNumber *hexutil.Big `json:"primeTerminusNumber" gencoden:"required"`
		TxHash     common.Hash     `json:"txHash" gencoden:"required"`
		Location   hexutil.Bytes   `json:"location" gencoden:"required"`
		MixHash	   common.Hash     `json:"mixHash" gencoden:"required"`
		Time       hexutil.Uint64  `json:"timestamp" gencoden:"required"`
		Nonce      BlockNonce      `json:"nonce" gencoden:"required"`
		Coinbase   common.Address  `json:"coinbase" gencoden:"required"`
	}

	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	wh.SetHeaderHash(dec.HeaderHash)
	wh.SetParentHash(dec.ParentHash)
	wh.SetNumber((*big.Int)(dec.Number))
	wh.SetDifficulty((*big.Int)(dec.Difficulty))
	wh.SetPrimeTerminusNumber((*big.Int)(dec.PrimeTerminusNumber))
	wh.SetTxHash(dec.TxHash)
	if len(dec.Location) > 0 {
		wh.location = make([]byte, len(dec.Location))
		copy(wh.location, dec.Location)
	}
	wh.SetMixHash(dec.MixHash)
	wh.SetTime(uint64(dec.Time))
	wh.SetNonce(dec.Nonce)
	wh.SetCoinbase(dec.Coinbase)
	return nil
}

func (wb *WorkObjectBody) MarshalJSON() ([]byte, error) {
	var enc struct {
		Header 				*Header 			`json:"header" gencoden:"required"`
		Transactions 		Transactions 		`json:"transactions" gencoden:"required"`
		ExtTransactions 	Transactions 		`json:"extTransactions" gencoden:"required"`
		Uncles 				[]*WorkObjectHeader	`json:"uncles" gencoden:"required"`
		Manifest 			BlockManifest 		`json:"manifest" gencoden:"required"`
		InterlinkHashes 	common.Hashes 		`json:"interlinkHashes" gencoden:"required"`
	}

	enc.Header = wb.Header()
	enc.Transactions = wb.Transactions()
	enc.ExtTransactions = wb.ExtTransactions()
	enc.Uncles = wb.Uncles()
	enc.Manifest = wb.Manifest()
	enc.InterlinkHashes = wb.InterlinkHashes()

	raw, err := json.Marshal(&enc)
	return raw, err
}

func (wb *WorkObjectBody) UnmarshalJSON(input []byte) error {
	var dec struct {
		Header 				*Header 				`json:"header" gencoden:"required"`
		Transactions 		Transactions 			`json:"transactions" gencoden:"required"`
		ExtTransactions 	Transactions 			`json:"extTransactions" gencoden:"required"`
		Uncles 				[]*WorkObjectHeader 	`json:"uncles" gencoden:"required"`
		Manifest 			BlockManifest 			`json:"manifest" gencoden:"required"`
		InterlinkHashes 	common.Hashes 			`json:"interlinkHashes" gencoden:"required"`
	}

	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	wb.SetHeader(dec.Header)
	wb.SetTransactions(dec.Transactions)
	wb.SetExtTransactions(dec.ExtTransactions)
	wb.SetUncles(dec.Uncles)
	wb.SetManifest(dec.Manifest)
	wb.SetInterlinkHashes(dec.InterlinkHashes)
	return nil
}

func (wo *WorkObject) MarshalJSON() ([]byte, error) {
	var enc struct {
		WoHeader *WorkObjectHeader `json:"woHeader" gencoden:"required"`
		WoBody   *WorkObjectBody   `json:"woBody" gencoden:"required"`
		Tx       *Transaction	   `json:"tx" gencoden:"required"`
	}

	enc.WoHeader = wo.WorkObjectHeader()
	enc.WoBody = wo.Body()
	enc.Tx = wo.Tx()

	raw, err := json.Marshal(&enc)
	return raw, err
}

func (wo *WorkObject) UnmarshalJSON(input []byte) error {
	var dec struct {
		WoHeader *WorkObjectHeader `json:"woHeader" gencoden:"required"`
		WoBody   *WorkObjectBody   `json:"woBody" gencoden:"required"`
		Tx 	     *Transaction      `json:"tx" gencoden:"required"`
	}

	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	wo.SetWorkObjectHeader(dec.WoHeader)
	wo.SetBody(dec.WoBody)
	wo.SetTx(dec.Tx)
	return nil
}