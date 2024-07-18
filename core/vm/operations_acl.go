// Copyright 2020 The go-ethereum Authors
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

package vm

import (
	"errors"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/common/math"
	"github.com/dominant-strategies/go-quai/params"
)

func makeGasSStoreFunc(clearingRefund uint64) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		// If we fail the minimum gas availability invariant, fail (0)
		if contract.Gas <= params.SstoreSentryGas {
			return 0, errors.New("not enough gas for reentrancy sentry")
		}
		// Gas sentry honoured, do the actual gas calculation based on the stored value
		var (
			y, x              = stack.Back(1), stack.peek()
			slot              = common.Hash(x.Bytes32())
			cost              = uint64(0)
			internalAddr, err = contract.Address().InternalAndQuaiAddress()
		)
		if err != nil {
			return 0, err
		}
		current := evm.StateDB.GetState(internalAddr, slot)
		// Check slot presence in the access list
		if _, slotPresent := evm.StateDB.SlotInAccessList(contract.Address().Bytes20(), slot); !slotPresent {
			cost = params.ColdSloadCost
		}
		value := common.Hash(y.Bytes32())

		if current == value { // noop (1)
			return cost + params.WarmStorageReadCost, nil // SLOAD_GAS
		}
		original := evm.StateDB.GetCommittedState(internalAddr, x.Bytes32())
		if original == current {
			if original == (common.Hash{}) { // create slot (2.1.1)
				return cost + params.SstoreSetGas, nil
			}
			if value == (common.Hash{}) { // delete slot (2.1.2b)
				evm.StateDB.AddRefund(clearingRefund)
			}
			return cost + (params.SstoreResetGas - params.ColdSloadCost), nil // write existing slot (2.1.2)
		}
		if original != (common.Hash{}) {
			if current == (common.Hash{}) { // recreate slot (2.2.1.1)
				evm.StateDB.SubRefund(clearingRefund)
			} else if value == (common.Hash{}) { // delete slot (2.2.1.2)
				evm.StateDB.AddRefund(clearingRefund)
			}
		}
		if original == value {
			if original == (common.Hash{}) { // reset to original inexistent slot (2.2.2.1)
				evm.StateDB.AddRefund(params.SstoreSetGas - params.WarmStorageReadCost)
			} else { // reset to original existing slot (2.2.2.2)
				evm.StateDB.AddRefund((params.SstoreResetGas - params.ColdSloadCost) - params.WarmStorageReadCost)
			}
		}
		return cost + params.WarmStorageReadCost, nil // dirty update (2.2)
	}
}

// gasSLoad calculates dynamic gas for SLOAD
// For SLOAD, if the (address, storage_key) pair (where address is the address of the contract
// whose storage is being read) is not yet in accessed_storage_keys,
// charge 2100 gas and add the pair to accessed_storage_keys.
// If the pair is already in accessed_storage_keys, charge 100 gas.
func gasSLoad(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	loc := stack.peek()
	slot := common.Hash(loc.Bytes32())
	// Check slot presence in the access list
	if _, slotPresent := evm.StateDB.SlotInAccessList(contract.Address().Bytes20(), slot); !slotPresent {
		// If the caller cannot afford the cost, this change will be rolled back
		// If he does afford it, we can skip checking the same thing later on, during execution
		return params.ColdSloadCost, nil
	}
	return params.WarmStorageReadCost, nil
}

// gasExtCodeCopy implements extcodecopy gas calculation
// > If the target is not in accessed_addresses,
// > charge COLD_ACCOUNT_ACCESS_COST gas, and add the address to accessed_addresses.
// > Otherwise, charge WARM_STORAGE_READ_COST gas.
func gasExtCodeCopy(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	// memory expansion first
	gas, err := gasExtCodeCopy(evm, contract, stack, mem, memorySize)
	if err != nil {
		return 0, err
	}
	addr := common.Bytes20ToAddress(stack.peek().Bytes20(), evm.chainConfig.Location)
	// Check slot presence in the access list
	if !evm.StateDB.AddressInAccessList(addr.Bytes20()) {
		var overflow bool
		// We charge (cold-warm), since 'warm' is already charged as constantGas
		if gas, overflow = math.SafeAdd(gas, params.ColdAccountAccessCost-params.WarmStorageReadCost); overflow {
			return 0, ErrGasUintOverflow
		}
		return gas, nil
	}
	return gas, nil
}

// gasAccountCheck checks whether the first stack item (as address) is present in the access list.
// If it is, this method returns '0', otherwise 'cold-warm' gas, presuming that the opcode using it
// is also using 'warm' as constant factor.
// This method is used by:
// - extcodehash,
// - extcodesize,
// - (ext) balance
func gasAccountCheck(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	addr := common.Bytes20ToAddress(stack.peek().Bytes20(), evm.chainConfig.Location)
	// Check slot presence in the access list
	if !evm.StateDB.AddressInAccessList(addr.Bytes20()) {
		// If the caller cannot afford the cost, this change will be rolled back
		// The warm storage read cost is already charged as constantGas
		return params.ColdAccountAccessCost - params.WarmStorageReadCost, nil
	}
	return 0, nil
}

func makeCallVariantGasCall(oldCalculator gasFunc) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		addr := common.Bytes20ToAddress(stack.Back(1).Bytes20(), evm.chainConfig.Location)
		// Check slot presence in the access list
		warmAccess := evm.StateDB.AddressInAccessList(addr.Bytes20())
		// The WarmStorageReadCost (100) is already deducted in the form of a constant cost, so
		// the cost to charge for cold access, if any, is Cold - Warm
		coldCost := params.ColdAccountAccessCost - params.WarmStorageReadCost
		if !warmAccess {
			// Charge the remaining difference here already, to correctly calculate available
			// gas for call
			if !contract.UseGas(coldCost) {
				return 0, ErrOutOfGas
			}
		}
		// Now call the old calculator, which takes into account
		// - create new account
		// - transfer value
		// - memory expansion
		// - 63/64ths rule
		gas, err := oldCalculator(evm, contract, stack, mem, memorySize)
		if warmAccess || err != nil {
			return gas, err
		}
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		contract.Gas += coldCost
		return gas + coldCost, nil
	}
}

var (
	gasCallVariant         = makeCallVariantGasCall(gasCall)
	gasDelegateCallVariant = makeCallVariantGasCall(gasDelegateCall)
	gasStaticCallVariant   = makeCallVariantGasCall(gasStaticCall)
	gasCallCodeVariant     = makeCallVariantGasCall(gasCallCode)
	// gasSelfdestructVariant implements self destruct with no refunds
	gasSelfdestructVariant = makeSelfdestructGasFn(false)

	// gasSStoreVariant implements gas cost for SSTORE
	// Replace `SSTORE_CLEARS_SCHEDULE` with `SSTORE_RESET_GAS + ACCESS_LIST_STORAGE_KEY_COST` (4,800)
	gasSStoreVariant = makeGasSStoreFunc(params.SstoreClearsScheduleRefund)
)

// makeSelfdestructGasFn can create the selfdestruct dynamic gas function
func makeSelfdestructGasFn(refundsEnabled bool) gasFunc {
	gasFunc := func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		var (
			gas                  uint64
			address              = common.Bytes20ToAddress(stack.peek().Bytes20(), evm.chainConfig.Location)
			internalAddress, err = address.InternalAndQuaiAddress()
		)
		if err != nil {
			return 0, err
		}
		contractAddress, err := contract.Address().InternalAndQuaiAddress()
		if err != nil {
			return 0, err
		}
		if !evm.StateDB.AddressInAccessList(address.Bytes20()) {
			// If the caller cannot afford the cost, this change will be rolled back
			gas = params.ColdAccountAccessCost
		}
		// if empty and transfers value
		if evm.StateDB.Empty(internalAddress) && evm.StateDB.GetBalance(contractAddress).Sign() != 0 {
			gas += params.CreateBySelfdestructGas
		}
		if refundsEnabled && !evm.StateDB.HasSuicided(contractAddress) {
			evm.StateDB.AddRefund(params.SelfdestructRefundGas)
		}
		return gas, nil
	}
	return gasFunc
}
