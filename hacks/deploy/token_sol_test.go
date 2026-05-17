package contracts

import (
	"fmt"
	"math/big"
	"testing"

	"wetee/test/tokensol"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ethRPC = "https://eth-rpc-testnet.polkadot.io"
const tokenAddr = "0xce8491de2c86c3e6fd7cad1035fe1fecac888667"

func newTokenCaller(t *testing.T) *tokensol.TokenCaller {
	t.Helper()
	client, err := ethclient.Dial(ethRPC)
	if err != nil {
		t.Fatalf("dial ETH RPC: %v", err)
	}
	token, err := tokensol.NewTokenCaller(common.HexToAddress(tokenAddr), client)
	if err != nil {
		t.Fatalf("NewTokenCaller: %v", err)
	}
	return token
}

func TestTokenSolGetRate(t *testing.T) {
	token := newTokenCaller(t)
	rate, err := token.GetRateSol(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("GetRateSol: %v", err)
	}
	fmt.Printf("get_rate_sol() = %s (兑换率: 1 ETH = %s 积分)\n", rate.String(), rate.String())
}

func TestTokenSolOwner(t *testing.T) {
	token := newTokenCaller(t)
	owner, err := token.OwnerSol(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("OwnerSol: %v", err)
	}
	fmt.Printf("owner_sol() = %s\n", owner.Hex())
}

func TestTokenSolGetBalance(t *testing.T) {
	token := newTokenCaller(t)
	// 查 Alice 余额
	alice := common.HexToAddress("0xf4a0ce74a91980053056a332fa8e0b4c43569fb3")
	bal, err := token.GetBalanceSol(&bind.CallOpts{}, alice)
	if err != nil {
		t.Fatalf("GetBalanceSol: %v", err)
	}
	fmt.Printf("get_balance_sol(%s) = %s wei\n", alice.Hex(), bal.String())
}

func TestTokenSolToPoints(t *testing.T) {
	token := newTokenCaller(t)
	points, err := token.ToPointsSol(&bind.CallOpts{}, big.NewInt(10))
	if err != nil {
		t.Fatalf("ToPointsSol: %v", err)
	}
	fmt.Printf("to_points_sol(10 wei) = %s 积分\n", points.String())
}

func TestTokenSolAll(t *testing.T) {
	fmt.Println("=== Token Sol ABI via go-ethereum ===")
	fmt.Printf("RPC:   %s\n", ethRPC)
	fmt.Printf("Token: %s\n\n", tokenAddr)

	t.Run("GetRate", TestTokenSolGetRate)
	t.Run("Owner", TestTokenSolOwner)
	t.Run("GetBalance", TestTokenSolGetBalance)
	t.Run("ToPoints", TestTokenSolToPoints)
}
