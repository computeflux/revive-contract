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

	// 查询充值前链上余额 & 事件 nonce
	chainBefore, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		t.Fatal("get chain balance before:", err)
	}
	fmt.Println("chain balance before recharge:", chainBefore.Data.Free)

	nonceBefore, _, err := tokenIns.QueryGetLatestNonce(param)
	if err != nil {
		t.Fatal("get_latest_nonce before:", err)
	}
	fmt.Println("latest event nonce before:", *nonceBefore)

	amount := types.NewU128(*big.NewInt(Unit))
	err = tokenIns.ExecRecharge(chain.ExecParams{
		Signer:    pk,
		PayAmount: amount,
	})
	if err != nil {
		t.Fatal("recharge:", err)
	}

	// 查询充值后链上余额
	chainAfter, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		t.Fatal("get chain balance after:", err)
	}
	fmt.Println("chain balance after recharge: ", chainAfter.Data.Free)

	// 计算链上消耗
	chainDiff := new(big.Int).Sub(chainBefore.Data.Free.Int, chainAfter.Data.Free.Int)
	fmt.Println("chain spent (Planck):", chainDiff)

	// 通过事件记录验证充值 | Verify recharge via events
	nonceAfter, _, err := tokenIns.QueryGetLatestNonce(param)
	if err != nil {
		t.Fatal("get_latest_nonce after:", err)
	}
	fmt.Println("latest event nonce after:", *nonceAfter)

	// 获取新增的事件
	events, _, err := tokenIns.QueryGetEvents(*nonceBefore+1, *nonceAfter, param)
	if err != nil {
		t.Fatal("get_events:", err)
	}
	if events == nil || len(*events) == 0 {
		t.Error("no new events after recharge")
	} else {
		fmt.Printf("new events count: %d\n", len(*events))
		for i, ev := range *events {
			fmt.Printf("  event[%d] contract=%s type=%s data_len=%d\n",
				i, string(ev.TargetContract), string(ev.EventType), len(ev.EventData))
		}
	}
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

	newRate := types.NewU256(*new(big.Int).SetUint64(20_000))
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

func TestSetErc20Token(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())

	// 查询注册前 ERC20 代币数量
	countBefore, _, err := tokenIns.QueryGetErc20Count(param)
	if err != nil {
		t.Fatal("get_erc20_count before:", err)
	}
	fmt.Println("erc20 count before:", countBefore)

	// 注册一个 ERC20 代币
	erc20Addr, err := util.HexToH160("0x179843f0804D92e85A22b3B105bb33A68a790A4E")
	if err != nil {
		t.Fatal("invalid erc20 address:", err)
	}
	active := true
	rate := types.NewU256(*new(big.Int).SetUint64(10_000))
	unit := types.NewU256(*new(big.Int).SetUint64(1_000_000_000_000_000_000))

	err = tokenIns.ExecSetErc20Token(erc20Addr, active, rate, unit, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal("set_erc20_token:", err)
	}

	// 验证：数量 +1
	countAfter, _, err := tokenIns.QueryGetErc20Count(param)
	if err != nil {
		t.Fatal("get_erc20_count after:", err)
	}
	fmt.Println("erc20 count after:", countAfter)

	// 验证：get_erc20_config 返回正确配置
	erc20Cfg, _, err := tokenIns.QueryGetErc20Config(erc20Addr, param)
	if err != nil {
		t.Fatal("get_erc20_config:", err)
	}
	if erc20Cfg == nil {
		t.Fatal("get_erc20_config returned nil")
	}
	fmt.Printf("erc20 config: active=%v rate=%s unit=%s\n", erc20Cfg.F0, erc20Cfg.F1.String(), erc20Cfg.F2.String())

	// 验证：get_erc20_list 包含注册的代币
	list, _, err := tokenIns.QueryGetErc20List(param)
	if err != nil {
		t.Fatal("get_erc20_list:", err)
	}
	if list == nil {
		t.Fatal("get_erc20_list returned nil")
	}
	fmt.Printf("erc20 list length: %d\n", len(*list))
	found := false
	for _, item := range *list {
		if item.F0 == erc20Addr {
			found = true
			fmt.Printf("  token=%v active=%v rate=%s unit=%s\n", item.F0.Hex(), item.F1, item.F2.String(), item.F3.String())
			break
		}
	}
	if !found {
		t.Error("registered token not found in get_erc20_list")
	}
}

func TestGetErc20List(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())

	// 查询 ERC20 代币总数
	count, _, err := tokenIns.QueryGetErc20Count(param)
	if err != nil {
		t.Fatal("get_erc20_count:", err)
	}
	fmt.Println("total erc20 count:", count)

	// 查询 ERC20 代币列表
	list, _, err := tokenIns.QueryGetErc20List(param)
	if err != nil {
		t.Fatal("get_erc20_list:", err)
	}
	if list == nil {
		t.Fatal("get_erc20_list returned nil")
	}
	fmt.Printf("erc20 list length: %d\n", len(*list))
	for _, item := range *list {
		fmt.Printf("  addr=%s active=%v rate=%s unit=%s\n",
			item.F0.Hex(), item.F1, item.F2.String(), item.F3.String())
	}
}

func TestTransErc20(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)
	param := chain.DefaultParamWithOrigin(pk.AccountID())

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	// user, err := util.HexToH160("0x21429C1E80300b503d7a0D933c613aEE3DAf3120")
	// if err != nil {
	// 	t.Fatal("invalid user address:", err)
	// }

	erc20Addr, err := util.HexToH160("0x179843f0804D92e85A22b3B105bb33A68a790A4E")
	if err != nil {
		t.Fatal("invalid erc20 address:", err)
	}

	b, _, err := tokenIns.QueryGetErc20Balance(erc20Addr, param)
	if err != nil {
		t.Fatal("GetErc20Balance :", err)
	}

	fmt.Println(b.String())

	// err = tokenIns.ExecTransErc20(erc20Addr, user, types.NewU256(*new(big.Int).SetUint64(1000)), chain.ExecParams{
	// 	Signer:    pk,
	// 	PayAmount: types.NewU128(*big.NewInt(0)),
	// })

	// if err != nil {
	// 	t.Fatal("trans erc20:", err)
	// }
}

func TestSetSubnet(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())
	oldRate, _, _ := tokenIns.QueryGetSubnet(param)
	fmt.Println("old addr:", oldRate.Hex())

	addr, _ := util.HexToH160("0x576afc0dab34389170845e9b996eca9017d2a505")
	err = tokenIns.ExecSetSubnet(addr, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal("set_subnet:", err)
	}
}

func TestBatchSetUnitAndRate(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenIns, err := token.InitTokenContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	param := chain.DefaultParamWithOrigin(pk.AccountID())

	// 设置新的 Rate
	newRate := types.NewU256(*new(big.Int).SetUint64(20_000))
	callRate, err := tokenIns.CallOfSetRate(newRate, param)
	if err != nil {
		t.Fatal("call of set_rate:", err)
	}

	// 设置新的 Unit
	newUnit := types.NewU256(*new(big.Int).SetUint64(100_000_000 * Unit))
	callUnit, err := tokenIns.CallOfSetTokenUnit(newUnit, param)
	if err != nil {
		t.Fatal("call of set_token_unit:", err)
	}

	// batch_all 同时执行
	calls := []types.Call{*callRate, *callUnit}
	batchCall, err := client.BatchCall("batch_all", calls)
	if err != nil {
		t.Fatal("batch call error:", err)
	}

	if err := client.SignAndSubmit(pk, *batchCall, true, 0); err != nil {
		t.Fatal("sign and submit:", err)
	}

	// 验证结果
	rate, _, err := tokenIns.QueryGetRate(param)
	if err != nil {
		t.Fatal("get_rate:", err)
	}
	fmt.Println("new rate:", rate.String())

	unit, _, err := tokenIns.QueryGetTokenUnit(param)
	if err != nil {
		t.Fatal("get_token_unit:", err)
	}
	fmt.Println("new unit:", unit.String())
}
