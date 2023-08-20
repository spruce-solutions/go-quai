// Copyright 2014 The go-ethereum Authors
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

package types

import (
	"bytes"
	"container/heap"
	"errors"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/crypto"
	"github.com/dominant-strategies/go-quai/rlp"
)

var (
	ErrInvalidSig         = errors.New("invalid transaction v, r, s values")
	ErrExpectedProtection = errors.New("transaction signature is not protected")
	ErrTxTypeNotSupported = errors.New("transaction type not supported")
	ErrGasFeeCapTooLow    = errors.New("fee cap less than base fee")
	errEmptyTypedTx       = errors.New("empty typed transaction bytes")
)

// Transaction types.
const (
	InternalTxType = iota
	ExternalTxType
	InternalToExternalTxType
)

// Transaction is a Quai transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash       atomic.Value
	size       atomic.Value
	from       atomic.Value
	toChain    atomic.Value
	fromChain  atomic.Value
	confirmCtx atomic.Value // Context at which the ETX may be confirmed
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// TxData is the underlying data of a transaction.
//
// This is implemented by InternalTx, ExternalTx and InternalToExternal.
type TxData interface {
	txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	value() *big.Int
	nonce() uint64
	to() *common.Address
	etxData() []byte
	etxAccessList() AccessList

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := tx.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	return rlp.Encode(w, tx.inner)
}

// MarshalBinary returns the canonical encoding of the transaction.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == rlp.String {
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		inner, err := tx.decodeTyped(b)
		if err == nil {
			tx.setDecoded(inner, len(b))
		}
		return err
	} else {
		return ErrTxTypeNotSupported
	}
}

// UnmarshalBinary decodes the canonical encoding of transactions.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	inner, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(inner, len(b))
	return nil
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(b []byte) (TxData, error) {
	if len(b) == 0 {
		return nil, errEmptyTypedTx
	}
	switch b[0] {
	case InternalTxType:
		var inner InternalTx
		err := rlp.DecodeBytes(b[1:], &inner)
		return &inner, err
	case ExternalTxType:
		var inner ExternalTx
		err := rlp.DecodeBytes(b[1:], &inner)
		return &inner, err
	case InternalToExternalTxType:
		var inner InternalToExternalTx
		err := rlp.DecodeBytes(b[1:], &inner)
		return &inner, err
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size int) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(common.StorageSize(size))
	}
}

func sanityCheckSignature(v *big.Int, r *big.Int, s *big.Int) error {
	if !crypto.ValidateSignatureValues(byte(v.Uint64()), r, s) {
		return ErrInvalidSig
	}
	return nil
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId returns the chain ID of the transaction. The return value will always be
// non-nil.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Value returns the ether amount of the transaction.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// ETXData returns the input data of the external transaction.
func (tx *Transaction) ETXData() []byte { return tx.inner.etxData() }

// ETXAccessList returns the access list of the transaction.
func (tx *Transaction) ETXAccessList() AccessList { return tx.inner.etxAccessList() }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

func (tx *Transaction) ETXSender() common.Address { return tx.inner.(*ExternalTx).Sender }

func (tx *Transaction) IsInternalToExternalTx() (inner *InternalToExternalTx, ok bool) {
	inner, ok = tx.inner.(*InternalToExternalTx)
	return
}

func (tx *Transaction) From() *common.Address {
	sc := tx.from.Load()
	if sc != nil {
		sigCache := sc.(sigCache)
		return &sigCache.from
	} else {
		return nil
	}
}

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *common.Address {
	// Copy the pointed-to address.
	ito := tx.inner.to()
	if ito == nil {
		return nil
	}
	cpy := *ito
	return &cpy
}

// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).SetUint64(0)
	total.Add(total, tx.Value())
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// Hash returns the transaction hash.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	h := prefixedRlpHash(tx.Type(), tx.inner)
	tx.hash.Store(h)
	return h
}

// FromChain returns the chain location this transaction originated from
func (tx *Transaction) FromChain() common.Location {
	if loc := tx.fromChain.Load(); loc != nil {
		return loc.(common.Location)
	}
	var loc common.Location
	switch tx.Type() {
	case ExternalTxType:
		// External transactions do not have a signature, but instead store the
		// sender explicitly. Use that sender to get the location.
		loc = *tx.inner.(*ExternalTx).Sender.Location()
	default:
		// All other TX types are signed, and should use the signature to determine
		// the sender location
		signer := NewSigner(tx.ChainId())
		from, err := Sender(signer, tx)
		if err != nil {
			panic("failed to get transaction sender!")
		}
		loc = *from.Location()
	}
	tx.fromChain.Store(loc)
	return loc
}

// ConfirmationCtx indicates the chain context at which this ETX becomes
// confirmed and referencable to the destination chain
func (tx *Transaction) ConfirmationCtx() int {
	if ctx := tx.confirmCtx.Load(); ctx != nil {
		return ctx.(int)
	}

	ctx := tx.To().Location().CommonDom(tx.FromChain()).Context()
	tx.confirmCtx.Store(ctx)
	return ctx
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previously cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.inner)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := tx.inner.copy()
	cpy.setSignatureValues(signer.ChainID(), v, r, s)
	return &Transaction{inner: cpy, time: tx.time}, nil
}

// Transactions implements DerivableList for transactions.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// EncodeIndex encodes the i'th transaction to w. Note that this does not check for errors
// because we assume that *Transaction will only ever contain valid txs that were either
// constructed by decoding or via public API in this package.
func (s Transactions) EncodeIndex(i int, w *bytes.Buffer) {
	tx := s[i]
	tx.encodeTyped(w)
}

// FilterByLocation returns the subset of transactions with a 'to' address which
// belongs the given chain location
func (s Transactions) FilterToLocation(l common.Location) Transactions {
	filteredList := Transactions{}
	for _, tx := range s {
		toChain := *tx.To().Location()
		if l.Equal(toChain) {
			filteredList = append(filteredList, tx)
		}
	}
	return filteredList
}

// FilterToSlice returns the subset of transactions with a 'to' address which
// belongs to the given slice location, at or above the given minimum context
func (s Transactions) FilterToSlice(slice common.Location, minCtx int) Transactions {
	filteredList := Transactions{}
	for _, tx := range s {
		toChain := tx.To().Location()
		if toChain.InSameSliceAs(slice) {
			filteredList = append(filteredList, tx)
		}
	}
	return filteredList
}

// FilterConfirmationCtx returns the subset of transactions who can be confirmed
// at the given context
func (s Transactions) FilterConfirmationCtx(ctx int) Transactions {
	filteredList := Transactions{}
	for _, tx := range s {
		if tx.ConfirmationCtx() == ctx {
			filteredList = append(filteredList, tx)
		}
	}
	return filteredList
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxWithMinerFee wraps a transaction with its gas price or effective miner gasTipCap
type TxWithMinerFee struct {
	tx       *Transaction
	minerFee *big.Int
}

// TxByPriceAndTime implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPriceAndTime []*TxWithMinerFee

func (s TxByPriceAndTime) Len() int { return len(s) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	// If the prices are equal, use the time the transaction was first seen for
	// deterministic sorting
	cmp := s[i].minerFee.Cmp(s[j].minerFee)
	if cmp == 0 {
		return s[i].tx.time.Before(s[j].tx.time)
	}
	return cmp > 0
}
func (s TxByPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *TxByPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*TxWithMinerFee))
}

func (s *TxByPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs     map[common.AddressBytes]Transactions // Per account nonce-sorted list of transactions
	heads   TxByPriceAndTime                     // Next transaction for each unique account (price heap)
	signer  Signer                               // Signer for the set of transactions
	baseFee *big.Int                             // Current base fee
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.AddressBytes]Transactions, sort bool) *TransactionsByPriceAndNonce {
	// Initialize a price and received time based heap with the head transactions
	heads := make(TxByPriceAndTime, 0, len(txs))
	for from, accTxs := range txs {
		acc, _ := Sender(signer, accTxs[0])
		// Remove transaction if sender doesn't match from, or if wrapping fails.
		if acc.Bytes20() != from {
			delete(txs, from)
			continue
		}
		txs[from] = accTxs[1:]
	}
	if sort {
		heap.Init(&heads)
	}

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0].tx
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift(acc common.AddressBytes, sort bool) {
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		return
	}
	if sort {
		heap.Pop(&t.heads)
	} else if len(t.heads) > 1 {
		t.heads = t.heads[1:]
	} else {
		t.heads = make(TxByPriceAndTime, 0)
	}

}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to            *common.Address
	from          common.Address
	nonce         uint64
	amount        *big.Int
	data          []byte
	accessList    AccessList
	checkNonce    bool
	etxsender     common.Address // only used in ETX
	txtype        byte
	etxData       []byte
	etxAccessList AccessList
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, data []byte, accessList AccessList, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		data:       data,
		accessList: accessList,
		checkNonce: checkNonce,
	}
}

// AsMessage returns the transaction as a core.Message.
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.Nonce(),
		to:         tx.To(),
		amount:     tx.Value(),
		data:       tx.Data(),
		accessList: tx.AccessList(),
		checkNonce: true,
		txtype:     tx.Type(),
	}
	var err error
	if tx.Type() == ExternalTxType {
		msg.from = common.ZeroAddr
		msg.etxsender, err = Sender(s, tx)
		msg.checkNonce = false
	} else {
		msg.from, err = Sender(s, tx)
	}
	if internalToExternalTx, ok := tx.IsInternalToExternalTx(); ok {
		msg.etxData = internalToExternalTx.ETXData
		msg.etxAccessList = internalToExternalTx.ETXAccessList
	}
	return msg, err
}

// AsMessageWithSender returns the transaction as a core.Message.
func (tx *Transaction) AsMessageWithSender(s Signer, sender *common.InternalAddress) (Message, error) {
	msg := Message{
		nonce:      tx.Nonce(),
		to:         tx.To(),
		amount:     tx.Value(),
		data:       tx.Data(),
		accessList: tx.AccessList(),
		checkNonce: true,
		txtype:     tx.Type(),
	}
	var err error
	if tx.Type() == ExternalTxType {
		msg.from = common.ZeroAddr
		msg.etxsender, err = Sender(s, tx)
		msg.checkNonce = false
	} else {
		if sender != nil {
			msg.from = common.NewAddressFromData(sender)
		} else {
			msg.from, err = Sender(s, tx)
		}
	}
	if internalToExternalTx, ok := tx.IsInternalToExternalTx(); ok {
		msg.etxData = internalToExternalTx.ETXData
		msg.etxAccessList = internalToExternalTx.ETXAccessList
	}
	return msg, err
}

func (m Message) From() common.Address      { return m.from }
func (m Message) To() *common.Address       { return m.to }
func (m Message) Value() *big.Int           { return m.amount }
func (m Message) Nonce() uint64             { return m.nonce }
func (m Message) Data() []byte              { return m.data }
func (m Message) AccessList() AccessList    { return m.accessList }
func (m Message) CheckNonce() bool          { return m.checkNonce }
func (m Message) ETXSender() common.Address { return m.etxsender }
func (m Message) Type() byte                { return m.txtype }
func (m Message) ETXData() []byte           { return m.etxData }
func (m Message) ETXAccessList() AccessList { return m.etxAccessList }

// AccessList is an access list.
type AccessList []AccessTuple

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}

// StorageKeys returns the total number of storage keys in the access list.
func (al AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

// This function must only be used by tests
func GetInnerForTesting(tx *Transaction) TxData {
	return tx.inner
}
