package contracts

import (
	"fmt"
	"math/big"
	"testing"

	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/system"
	"github.com/wetee-dao/ink.go/util"
)

const Unit = 10_000_000_000

// 4905.0_859_764_437
func TestGetChainBalance(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	// 账户地址
	signer := pk.(*chain.Signer)
	fmt.Println("Account SS58:", signer.Address)
	h160, _ := util.H160FromPublicKey(pk.Public())
	fmt.Println("Account H160:", h160.Hex())

	// 查询链上余额
	accountInfo, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		t.Fatal("get account balance:", err)
	}
	fmt.Println("Chain Free Balance:", accountInfo.Data.Free)
}

func TestRecharge(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	h160, _ := util.H160FromPublicKey(pk.Public())

	// 查询充值前余额
	chainBefore, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		t.Fatal("get chain balance before:", err)
	}
	fmt.Println("chain balance before recharge:", chainBefore.Data.Free)

	balBefore, _, err := tokenIns.QueryGetBalance(h160, param)
	if err != nil {
		t.Fatal("get_balance before:", err)
	}
	fmt.Println("credits before recharge:  ", balBefore.String())

	amount := types.NewU128(*big.NewInt(Unit))
	err = tokenIns.ExecRecharge(chain.ExecParams{
		Signer:    pk,
		PayAmount: amount,
	})
	if err != nil {
		t.Fatal("recharge:", err)
	}

	// 查询充值后余额
	chainAfter, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		t.Fatal("get chain balance after:", err)
	}
	fmt.Println("chain balance after recharge: ", chainAfter.Data.Free)

	balAfter, _, err := tokenIns.QueryGetBalance(h160, param)
	if err != nil {
		t.Fatal("get_balance after:", err)
	}
	fmt.Println("credits after recharge:   ", balAfter.String())

	// 计算充值消耗和积分
	chainDiff := new(big.Int).Sub(chainBefore.Data.Free.Int, chainAfter.Data.Free.Int)
	creditDiff := new(big.Int).Sub(balAfter.Int, balBefore.Int)
	fmt.Println("chain spent (Planck):", chainDiff)
	fmt.Println("recharged credits:   ", creditDiff.Uint64())
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

	newRate := types.NewU256(*new(big.Int).SetUint64(20_0000))
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

	// 3 DOT = 3 × TOKEN_UNIT Planck
	dots := new(big.Int).SetUint64(3 * Unit)
	ethAmount := types.NewU256(*dots)

	points, _, err := tokenIns.QueryToPoints(ethAmount, param)
	if err != nil {
		t.Fatal("to_points:", err)
	}
	fmt.Println(*points)
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

func TestSetTokenUnit(t *testing.T) {
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

	newUnit := types.NewU256(*new(big.Int).SetUint64(100_000_000 * Unit))
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
