package vm

import (
	"math/big"
	"testing"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/crypto"
	"github.com/dominant-strategies/go-quai/params"
)

func TestHelloContract(t *testing.T) {
	var (
		env             = NewEVM(BlockContext{}, TxContext{}, nil, params.TestChainConfig, Config{})
		wasmInterpreter = NewWASMInterpreter(env, env.Config)
	)

	wasmBytes, err := wasmtime.Wat2Wasm(`
		(module
		  (func $logHelloWorld (import "" "logHelloWorld"))
		  (func (export "run") (call $logHelloWorld))
		)
		`)

	if err != nil {
		t.Errorf("error: %v", err)
	}

	// Track time taken and memory usage
	// defer trackTime(time.Now(), "wasm")

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(&dummyContractRef{}, &dummyContractRef{}, new(big.Int), 2000)
	contract.SetCodeOptionalHash(nil, &codeAndHash{
		code: wasmBytes,
	})

	_, err = wasmInterpreter.Run(contract, nil, false)
	if err != nil {
		t.Errorf("error: %v", err)
	}
}

func TestUseGasContract(t *testing.T) {
	var (
		env             = NewEVM(BlockContext{}, TxContext{}, nil, params.TestChainConfig, Config{})
		wasmInterpreter = NewWASMInterpreter(env, env.Config)
	)

	wasmBytes, err := wasmtime.Wat2Wasm(`
	(module
		(func $useGas (import "" "useGas") (param i64))
		(func (export "run") (call $useGas (i64.const 1)))
	  )
		`)

	if err != nil {
		t.Errorf("error: %v", err)
	}
	// Track time taken and memory usage
	// defer trackTime(time.Now(), "wasm")

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(&dummyContractRef{}, &dummyContractRef{}, new(big.Int), 2000)
	contract.SetCodeOptionalHash(nil, &codeAndHash{
		code: wasmBytes,
	})

	_, err = wasmInterpreter.Run(contract, nil, false)
	if err != nil {
		t.Errorf("error: %v", err)
	}
}
func TestGetAddressContract(t *testing.T) {
	var (
		env             = NewEVM(BlockContext{}, TxContext{}, nil, params.TestChainConfig, Config{})
		wasmInterpreter = NewWASMInterpreter(env, env.Config)
	)

	wasmBytes, err := wasmtime.Wat2Wasm(`
		(module
			(func $getAddress (import "" "getAddress") (param i32))
			(func (export "run") 
				(local i32)
				(local.set 0 (i32.const 0))  ;; Set memory offset to 0
				(call $getAddress (local.get 0))  ;; Call getAddress with memory offset
			)
		)
		`)

	if err != nil {
		t.Errorf("error: %v", err)
	}

	contract := NewContract(&dummyContractRef{}, &dummyContractRef{}, new(big.Int), 2000)
	contract.SetCodeOptionalHash(&common.ZeroAddr, &codeAndHash{
		code: wasmBytes,
		hash: crypto.Keccak256Hash(wasmBytes),
	})

	_, err = wasmInterpreter.Run(contract, nil, false)
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// Optionally, you can add code to read the memory and verify the address
}
