// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package tokensol

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
	_ = time.Tick
	_ = context.Background
)

// Struct1 is an auto generated low-level Go binding around an user-defined struct.
type Struct1 struct {
	User   common.Address
	Token  common.Address
	Amount *big.Int
	Status uint8
}

// Struct0 is an auto generated low-level Go binding around an user-defined struct.
type Struct0 struct {
	Arg1 common.Address
	Arg2 bool
	Arg3 *big.Int
	Arg4 *big.Int
}

// Struct2 is an auto generated low-level Go binding around an user-defined struct.
type Struct2 struct {
	Arg1 bool
	Arg2 *big.Int
	Arg3 *big.Int
}

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"recharge_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"token\",\"type\":\"address\"},{\"components\":[],\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"recharge_erc20\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"dot_amount\",\"type\":\"uint256\"}],\"name\":\"to_points_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_rate_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_native_active_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"token\",\"type\":\"address\"}],\"name\":\"get_erc20_config_sol\",\"outputs\":[{\"components\":[{\"components\":[],\"name\":\"arg1\",\"type\":\"bool\"},{\"components\":[],\"name\":\"arg2\",\"type\":\"uint256\"},{\"components\":[],\"name\":\"arg3\",\"type\":\"uint256\"}],\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_erc20_count_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_erc20_list_sol\",\"outputs\":[{\"components\":[{\"components\":[],\"name\":\"arg1\",\"type\":\"address\"},{\"components\":[],\"name\":\"arg2\",\"type\":\"bool\"},{\"components\":[],\"name\":\"arg3\",\"type\":\"uint256\"},{\"components\":[],\"name\":\"arg4\",\"type\":\"uint256\"}],\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"nonce\",\"type\":\"uint64\"}],\"name\":\"get_pending_withdrawal_sol\",\"outputs\":[{\"components\":[{\"components\":[],\"name\":\"user\",\"type\":\"address\"},{\"components\":[],\"name\":\"token\",\"type\":\"address\"},{\"components\":[],\"name\":\"amount\",\"type\":\"uint256\"},{\"components\":[],\"name\":\"status\",\"type\":\"uint8\"}],\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"nonce\",\"type\":\"uint64\"}],\"name\":\"claim_withdrawal_sol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"nonce\",\"type\":\"uint64\"}],\"name\":\"cancel_withdrawal_sol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"nonpayable\",\"type\":\"fallback\"}]",
}

// TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenMetaData.ABI instead.
var TokenABI = TokenMetaData.ABI

// Token is an auto generated Go binding around an Ethereum contract.
type Token struct {
	TokenCaller     // Read-only binding to the contract
	TokenTransactor // Write-only binding to the contract
	TokenFilterer   // Log filterer for contract events
}

// TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenSession struct {
	Contract     *Token            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCallerSession struct {
	Contract *TokenCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenTransactorSession struct {
	Contract     *TokenTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenRaw struct {
	Contract *Token // Generic contract binding to access the raw methods on
}

// TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCallerRaw struct {
	Contract *TokenCaller // Generic read-only contract binding to access the raw methods on
}

// TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenTransactorRaw struct {
	Contract *TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewToken creates a new instance of Token, bound to a specific deployed contract.
func NewToken(address common.Address, backend bind.ContractBackend) (*Token, error) {
	contract, err := bindToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Token{TokenCaller: TokenCaller{contract: contract}, TokenTransactor: TokenTransactor{contract: contract}, TokenFilterer: TokenFilterer{contract: contract}}, nil
}

// NewTokenCaller creates a new read-only instance of Token, bound to a specific deployed contract.
func NewTokenCaller(address common.Address, caller bind.ContractCaller) (*TokenCaller, error) {
	contract, err := bindToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCaller{contract: contract}, nil
}

// NewTokenTransactor creates a new write-only instance of Token, bound to a specific deployed contract.
func NewTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenTransactor, error) {
	contract, err := bindToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenTransactor{contract: contract}, nil
}

// NewTokenFilterer creates a new log filterer instance of Token, bound to a specific deployed contract.
func NewTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenFilterer, error) {
	contract, err := bindToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenFilterer{contract: contract}, nil
}

// bindToken binds a generic wrapper to an already deployed contract.
func bindToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.contract.Transact(opts, method, params...)
}

// GetErc20ConfigSol is a free data retrieval call binding the contract method 0xb4094f72.
//
// Solidity: function get_erc20_config_sol(address token) view returns((bool,uint256,uint256))
func (_Token *TokenCaller) GetErc20ConfigSol(opts *bind.CallOpts, token common.Address) (Struct2, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_erc20_config_sol", token)

	if err != nil {
		return *new(Struct2), err
	}

	out0 := *abi.ConvertType(out[0], new(Struct2)).(*Struct2)

	return out0, err

}

// GetErc20ConfigSol is a free data retrieval call binding the contract method 0xb4094f72.
//
// Solidity: function get_erc20_config_sol(address token) view returns((bool,uint256,uint256))
func (_Token *TokenSession) GetErc20ConfigSol(token common.Address) (Struct2, error) {
	return _Token.Contract.GetErc20ConfigSol(&_Token.CallOpts, token)
}

// GetErc20ConfigSol is a free data retrieval call binding the contract method 0xb4094f72.
//
// Solidity: function get_erc20_config_sol(address token) view returns((bool,uint256,uint256))
func (_Token *TokenCallerSession) GetErc20ConfigSol(token common.Address) (Struct2, error) {
	return _Token.Contract.GetErc20ConfigSol(&_Token.CallOpts, token)
}

// GetErc20CountSol is a free data retrieval call binding the contract method 0xbf29e1a7.
//
// Solidity: function get_erc20_count_sol() view returns(uint256)
func (_Token *TokenCaller) GetErc20CountSol(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_erc20_count_sol")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetErc20CountSol is a free data retrieval call binding the contract method 0xbf29e1a7.
//
// Solidity: function get_erc20_count_sol() view returns(uint256)
func (_Token *TokenSession) GetErc20CountSol() (*big.Int, error) {
	return _Token.Contract.GetErc20CountSol(&_Token.CallOpts)
}

// GetErc20CountSol is a free data retrieval call binding the contract method 0xbf29e1a7.
//
// Solidity: function get_erc20_count_sol() view returns(uint256)
func (_Token *TokenCallerSession) GetErc20CountSol() (*big.Int, error) {
	return _Token.Contract.GetErc20CountSol(&_Token.CallOpts)
}

// GetErc20ListSol is a free data retrieval call binding the contract method 0x5c0c36b0.
//
// Solidity: function get_erc20_list_sol() view returns((address,bool,uint256,uint256)[])
func (_Token *TokenCaller) GetErc20ListSol(opts *bind.CallOpts) ([]Struct0, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_erc20_list_sol")

	if err != nil {
		return *new([]Struct0), err
	}

	out0 := *abi.ConvertType(out[0], new([]Struct0)).(*[]Struct0)

	return out0, err

}

// GetErc20ListSol is a free data retrieval call binding the contract method 0x5c0c36b0.
//
// Solidity: function get_erc20_list_sol() view returns((address,bool,uint256,uint256)[])
func (_Token *TokenSession) GetErc20ListSol() ([]Struct0, error) {
	return _Token.Contract.GetErc20ListSol(&_Token.CallOpts)
}

// GetErc20ListSol is a free data retrieval call binding the contract method 0x5c0c36b0.
//
// Solidity: function get_erc20_list_sol() view returns((address,bool,uint256,uint256)[])
func (_Token *TokenCallerSession) GetErc20ListSol() ([]Struct0, error) {
	return _Token.Contract.GetErc20ListSol(&_Token.CallOpts)
}

// GetNativeActiveSol is a free data retrieval call binding the contract method 0x3c71900b.
//
// Solidity: function get_native_active_sol() view returns(bool)
func (_Token *TokenCaller) GetNativeActiveSol(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_native_active_sol")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetNativeActiveSol is a free data retrieval call binding the contract method 0x3c71900b.
//
// Solidity: function get_native_active_sol() view returns(bool)
func (_Token *TokenSession) GetNativeActiveSol() (bool, error) {
	return _Token.Contract.GetNativeActiveSol(&_Token.CallOpts)
}

// GetNativeActiveSol is a free data retrieval call binding the contract method 0x3c71900b.
//
// Solidity: function get_native_active_sol() view returns(bool)
func (_Token *TokenCallerSession) GetNativeActiveSol() (bool, error) {
	return _Token.Contract.GetNativeActiveSol(&_Token.CallOpts)
}

// GetPendingWithdrawalSol is a free data retrieval call binding the contract method 0xbaa174cd.
//
// Solidity: function get_pending_withdrawal_sol(uint64 nonce) view returns((address,address,uint256,uint8))
func (_Token *TokenCaller) GetPendingWithdrawalSol(opts *bind.CallOpts, nonce uint64) (Struct1, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_pending_withdrawal_sol", nonce)

	if err != nil {
		return *new(Struct1), err
	}

	out0 := *abi.ConvertType(out[0], new(Struct1)).(*Struct1)

	return out0, err

}

// GetPendingWithdrawalSol is a free data retrieval call binding the contract method 0xbaa174cd.
//
// Solidity: function get_pending_withdrawal_sol(uint64 nonce) view returns((address,address,uint256,uint8))
func (_Token *TokenSession) GetPendingWithdrawalSol(nonce uint64) (Struct1, error) {
	return _Token.Contract.GetPendingWithdrawalSol(&_Token.CallOpts, nonce)
}

// GetPendingWithdrawalSol is a free data retrieval call binding the contract method 0xbaa174cd.
//
// Solidity: function get_pending_withdrawal_sol(uint64 nonce) view returns((address,address,uint256,uint8))
func (_Token *TokenCallerSession) GetPendingWithdrawalSol(nonce uint64) (Struct1, error) {
	return _Token.Contract.GetPendingWithdrawalSol(&_Token.CallOpts, nonce)
}

// GetRateSol is a free data retrieval call binding the contract method 0x14ed3508.
//
// Solidity: function get_rate_sol() view returns(uint256)
func (_Token *TokenCaller) GetRateSol(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_rate_sol")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRateSol is a free data retrieval call binding the contract method 0x14ed3508.
//
// Solidity: function get_rate_sol() view returns(uint256)
func (_Token *TokenSession) GetRateSol() (*big.Int, error) {
	return _Token.Contract.GetRateSol(&_Token.CallOpts)
}

// GetRateSol is a free data retrieval call binding the contract method 0x14ed3508.
//
// Solidity: function get_rate_sol() view returns(uint256)
func (_Token *TokenCallerSession) GetRateSol() (*big.Int, error) {
	return _Token.Contract.GetRateSol(&_Token.CallOpts)
}

// OwnerSol is a free data retrieval call binding the contract method 0x4e6ec904.
//
// Solidity: function owner_sol() view returns(address)
func (_Token *TokenCaller) OwnerSol(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "owner_sol")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerSol is a free data retrieval call binding the contract method 0x4e6ec904.
//
// Solidity: function owner_sol() view returns(address)
func (_Token *TokenSession) OwnerSol() (common.Address, error) {
	return _Token.Contract.OwnerSol(&_Token.CallOpts)
}

// OwnerSol is a free data retrieval call binding the contract method 0x4e6ec904.
//
// Solidity: function owner_sol() view returns(address)
func (_Token *TokenCallerSession) OwnerSol() (common.Address, error) {
	return _Token.Contract.OwnerSol(&_Token.CallOpts)
}

// ToPointsSol is a free data retrieval call binding the contract method 0x1c342812.
//
// Solidity: function to_points_sol(uint256 dot_amount) view returns(uint256)
func (_Token *TokenCaller) ToPointsSol(opts *bind.CallOpts, dot_amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "to_points_sol", dot_amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ToPointsSol is a free data retrieval call binding the contract method 0x1c342812.
//
// Solidity: function to_points_sol(uint256 dot_amount) view returns(uint256)
func (_Token *TokenSession) ToPointsSol(dot_amount *big.Int) (*big.Int, error) {
	return _Token.Contract.ToPointsSol(&_Token.CallOpts, dot_amount)
}

// ToPointsSol is a free data retrieval call binding the contract method 0x1c342812.
//
// Solidity: function to_points_sol(uint256 dot_amount) view returns(uint256)
func (_Token *TokenCallerSession) ToPointsSol(dot_amount *big.Int) (*big.Int, error) {
	return _Token.Contract.ToPointsSol(&_Token.CallOpts, dot_amount)
}

// CancelWithdrawalSol is a paid mutator transaction binding the contract method 0x13164112.
//
// Solidity: function cancel_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenTransactor) CancelWithdrawalSol(opts *bind.TransactOpts, nonce uint64) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "cancel_withdrawal_sol", nonce)
}

// CancelWithdrawalSol is a paid mutator transaction binding the contract method 0x13164112.
//
// Solidity: function cancel_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenSession) CancelWithdrawalSol(nonce uint64) (*types.Transaction, error) {
	return _Token.Contract.CancelWithdrawalSol(&_Token.TransactOpts, nonce)
}

// CancelWithdrawalSol is a paid mutator transaction binding the contract method 0x13164112.
//
// Solidity: function cancel_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenTransactorSession) CancelWithdrawalSol(nonce uint64) (*types.Transaction, error) {
	return _Token.Contract.CancelWithdrawalSol(&_Token.TransactOpts, nonce)
}

// ClaimWithdrawalSol is a paid mutator transaction binding the contract method 0x0d43a902.
//
// Solidity: function claim_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenTransactor) ClaimWithdrawalSol(opts *bind.TransactOpts, nonce uint64) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "claim_withdrawal_sol", nonce)
}

// ClaimWithdrawalSol is a paid mutator transaction binding the contract method 0x0d43a902.
//
// Solidity: function claim_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenSession) ClaimWithdrawalSol(nonce uint64) (*types.Transaction, error) {
	return _Token.Contract.ClaimWithdrawalSol(&_Token.TransactOpts, nonce)
}

// ClaimWithdrawalSol is a paid mutator transaction binding the contract method 0x0d43a902.
//
// Solidity: function claim_withdrawal_sol(uint64 nonce) returns()
func (_Token *TokenTransactorSession) ClaimWithdrawalSol(nonce uint64) (*types.Transaction, error) {
	return _Token.Contract.ClaimWithdrawalSol(&_Token.TransactOpts, nonce)
}

// RechargeErc20 is a paid mutator transaction binding the contract method 0xbed17916.
//
// Solidity: function recharge_erc20(address token, uint256 amount) returns(uint256)
func (_Token *TokenTransactor) RechargeErc20(opts *bind.TransactOpts, token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "recharge_erc20", token, amount)
}

// RechargeErc20 is a paid mutator transaction binding the contract method 0xbed17916.
//
// Solidity: function recharge_erc20(address token, uint256 amount) returns(uint256)
func (_Token *TokenSession) RechargeErc20(token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Token.Contract.RechargeErc20(&_Token.TransactOpts, token, amount)
}

// RechargeErc20 is a paid mutator transaction binding the contract method 0xbed17916.
//
// Solidity: function recharge_erc20(address token, uint256 amount) returns(uint256)
func (_Token *TokenTransactorSession) RechargeErc20(token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Token.Contract.RechargeErc20(&_Token.TransactOpts, token, amount)
}

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() payable returns(uint256)
func (_Token *TokenTransactor) RechargeSol(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "recharge_sol")
}

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() payable returns(uint256)
func (_Token *TokenSession) RechargeSol() (*types.Transaction, error) {
	return _Token.Contract.RechargeSol(&_Token.TransactOpts)
}

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() payable returns(uint256)
func (_Token *TokenTransactorSession) RechargeSol() (*types.Transaction, error) {
	return _Token.Contract.RechargeSol(&_Token.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_Token *TokenTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _Token.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_Token *TokenSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Token.Contract.Fallback(&_Token.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_Token *TokenTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Token.Contract.Fallback(&_Token.TransactOpts, calldata)
}
