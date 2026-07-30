package contracts

import (
	"fmt"
	"math/big"
	"testing"

	tokensol "wetee/test/evm/binds"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ethRPC = "https://eth-rpc-testnet.polkadot.io"
const tokenAddr = "0xe701aeca672b5e1fd177df8096ac8888796e6427"

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
	t.Run("ToPoints", TestTokenSolToPoints)
}
