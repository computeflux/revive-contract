package contracts

import (
	"fmt"
	"math/big"
	"testing"

	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/util"
)

func TestRecharge(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	amount := types.NewU128(*big.NewInt(1_000_000_000_000))
	err = tokenIns.ExecRecharge(chain.ExecParams{
		Signer:    pk,
		PayAmount: amount,
	})
	if err != nil {
		t.Fatal("recharge:", err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	h160, _ := util.H160FromPublicKey(pk.Public())
	bal, _, err := tokenIns.QueryGetBalance(h160, param)
	if err != nil {
		t.Fatal("get_balance:", err)
	}
	fmt.Println("balance after recharge:", bal)
}

func TestGetRate(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	rate, _, err := tokenIns.QueryGetRate(param)
	if err != nil {
		t.Fatal("get_rate:", err)
	}
	fmt.Println("rate:", rate.String())
}

func TestSetRate(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	oldRate, _, _ := tokenIns.QueryGetRate(param)
	fmt.Println("old rate:", oldRate.String())

	newRate := types.NewU256(*new(big.Int).SetUint64(2))
	err = tokenIns.ExecSetRate(newRate, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal("set_rate:", err)
	}

	rate, _, err := tokenIns.QueryGetRate(param)
	if err != nil {
		t.Fatal("get_rate:", err)
	}
	fmt.Println("new rate:", rate)
}

func TestToPoints(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	// 先查询 TOKEN_UNIT
	param := chain.DefaultParamWithOrigin(pk.AccountID())
	unit, _, err := tokenIns.QueryGetTokenUnit(param)
	if err != nil {
		t.Fatal("get_token_unit:", err)
	}

	// 3 DOT = 3 × TOKEN_UNIT Planck
	dots := new(big.Int).SetUint64(3)
	threeDOT := new(big.Int).Mul(dots, unit.Int)
	ethAmount := types.NewU256(*threeDOT)

	points, _, err := tokenIns.QueryToPoints(ethAmount, param)
	if err != nil {
		t.Fatal("to_points:", err)
	}
	fmt.Println("3 DOT =", points, "points")
}

func TestGetBalance(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	h160, _ := util.H160FromPublicKey(pk.Public())
	bal, _, err := tokenIns.QueryGetBalance(h160, param)
	if err != nil {
		t.Fatal("get_balance:", err)
	}
	fmt.Println("balance:", bal)
}

func TestTokenUnit(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())

	unit, _, err := tokenIns.QueryGetTokenUnit(param)
	if err != nil {
		t.Fatal("get_token_unit:", err)
	}
	fmt.Println("default unit:", unit.String())

	newUnit := types.NewU256(*new(big.Int).SetUint64(10_000_000_000))
	err = tokenIns.ExecSetTokenUnit(newUnit, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal("set_token_unit:", err)
	}

	unit, _, err = tokenIns.QueryGetTokenUnit(param)
	if err != nil {
		t.Fatal("get_token_unit:", err)
	}
	fmt.Println("new unit:", unit.String())
}
