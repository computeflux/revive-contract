// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package tokensol

import (
	"errors"
	"math/big"
	"strings"

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
)

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"recharge_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"user\",\"type\":\"address\"},{\"components\":[],\"name\":\"points\",\"type\":\"uint256\"}],\"name\":\"withdraw_sol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"user\",\"type\":\"address\"}],\"name\":\"get_balance_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[],\"name\":\"dot_amount\",\"type\":\"uint256\"}],\"name\":\"to_points_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_rate_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner_sol\",\"outputs\":[{\"components\":[],\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"nonpayable\",\"type\":\"fallback\"}]",
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

// GetBalanceSol is a free data retrieval call binding the contract method 0x33097d86.
//
// Solidity: function get_balance_sol(address user) view returns(uint256)
func (_Token *TokenCaller) GetBalanceSol(opts *bind.CallOpts, user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "get_balance_sol", user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBalanceSol is a free data retrieval call binding the contract method 0x33097d86.
//
// Solidity: function get_balance_sol(address user) view returns(uint256)
func (_Token *TokenSession) GetBalanceSol(user common.Address) (*big.Int, error) {
	return _Token.Contract.GetBalanceSol(&_Token.CallOpts, user)
}

// GetBalanceSol is a free data retrieval call binding the contract method 0x33097d86.
//
// Solidity: function get_balance_sol(address user) view returns(uint256)
func (_Token *TokenCallerSession) GetBalanceSol(user common.Address) (*big.Int, error) {
	return _Token.Contract.GetBalanceSol(&_Token.CallOpts, user)
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

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() returns(uint256)
func (_Token *TokenTransactor) RechargeSol(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "recharge_sol")
}

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() returns(uint256)
func (_Token *TokenSession) RechargeSol() (*types.Transaction, error) {
	return _Token.Contract.RechargeSol(&_Token.TransactOpts)
}

// RechargeSol is a paid mutator transaction binding the contract method 0xf924a387.
//
// Solidity: function recharge_sol() returns(uint256)
func (_Token *TokenTransactorSession) RechargeSol() (*types.Transaction, error) {
	return _Token.Contract.RechargeSol(&_Token.TransactOpts)
}

// WithdrawSol is a paid mutator transaction binding the contract method 0xafde79c9.
//
// Solidity: function withdraw_sol(address user, uint256 points) returns()
func (_Token *TokenTransactor) WithdrawSol(opts *bind.TransactOpts, user common.Address, points *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "withdraw_sol", user, points)
}

// WithdrawSol is a paid mutator transaction binding the contract method 0xafde79c9.
//
// Solidity: function withdraw_sol(address user, uint256 points) returns()
func (_Token *TokenSession) WithdrawSol(user common.Address, points *big.Int) (*types.Transaction, error) {
	return _Token.Contract.WithdrawSol(&_Token.TransactOpts, user, points)
}

// WithdrawSol is a paid mutator transaction binding the contract method 0xafde79c9.
//
// Solidity: function withdraw_sol(address user, uint256 points) returns()
func (_Token *TokenTransactorSession) WithdrawSol(user common.Address, points *big.Int) (*types.Transaction, error) {
	return _Token.Contract.WithdrawSol(&_Token.TransactOpts, user, points)
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
