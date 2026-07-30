package contracts

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	tokensol "wetee/test/evm/binds"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FluxConfig 对应 tee-node/ext/actives/flux_config.json
type FluxConfig struct {
	RPCURL       string `json:"rpc_url"`
	PrivateKey   string `json:"private_key"`
	TokenAddress string `json:"token_address"`
}

func loadFluxConfig(t *testing.T) *FluxConfig {
	t.Helper()
	path := filepath.Join("..", "..", "tee-node", "ext", "actives", "flux_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read flux config %s: %v", path, err)
	}
	var cfg FluxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse flux config: %v", err)
	}
	return &cfg
}

func TestRechargeErc20(t *testing.T) {
	fCfg := loadFluxConfig(t)

	cli, err := ethclient.Dial(fCfg.RPCURL)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	cfg := loadConfig(t)
	uupsToken := common.HexToAddress(cfg.Contracts.Token)
	fmt.Println("token:", uupsToken.Hex())

	key, err := crypto.HexToECDSA(fCfg.PrivateKey[2:])
	if err != nil {
		t.Fatal(err)
	}
	chainID, err := cli.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	signer := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Println("signer:", signer.Hex())

	// 创建 Token 合约绑定
	tokenContract, err := tokensol.NewToken(uupsToken, cli)
	if err != nil {
		t.Fatal("NewToken:", err)
	}

	amount := big.NewInt(1e18)
	fluxAddr := common.HexToAddress(fCfg.TokenAddress)

	// 0. 查询充值前 EVENT_NONCE（SCALE 编码，手动 calldata）
	nonceBefore := getEventNonce(t, cli, uupsToken)

	// 1. approve: Flux ERC20 合约不在 Token 绑定里，仍用手动 calldata
	calldata := append(
		append([]byte{0x09, 0x5e, 0xa7, 0xb3}, padAddress(uupsToken)...),
		padU256(amount)...,
	)
	tx1, err := sendEVMTx(cli, key, chainID, fluxAddr, nil, 100000, calldata)
	if err != nil {
		t.Fatal("approve:", err)
	}
	fmt.Println("approve:", tx1.Hash().Hex())

	// 2. recharge_erc20: 使用 Token 绑定
	transactOpts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		t.Fatal("NewKeyedTransactorWithChainID:", err)
	}
	tx2, err := tokenContract.RechargeErc20(transactOpts, fluxAddr, amount)
	if err != nil {
		t.Fatalf("recharge: %v", err)
	}
	fmt.Println("recharge:", tx2.Hash().Hex())

	time.Sleep(time.Second * 10)

	// 3. 查询充值后 EVENT_NONCE，验证 +1
	nonceAfter := getEventNonce(t, cli, uupsToken)
	if nonceAfter != nonceBefore+1 {
		t.Fatalf("EVENT_NONCE did not increment: before=%d, after=%d", nonceBefore, nonceAfter)
	}
	t.Logf("EVENT_NONCE: %d -> %d (OK)", nonceBefore, nonceAfter)
}

func sendEVMTx(cli *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int, to common.Address, value *big.Int, gasLimit uint64, data []byte) (*ethtypes.Transaction, error) {
	gasPrice, _ := cli.SuggestGasPrice(context.Background())
	nonce, _ := cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(key.PublicKey))
	tx := ethtypes.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	signed, _ := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), key)
	return signed, cli.SendTransaction(context.Background(), signed)
}

func callEVM(cli *ethclient.Client, to common.Address, data []byte) ([]byte, error) {
	return cli.CallContract(context.Background(), ethereum.CallMsg{To: &to, Data: data}, nil)
}

func padAddress(addr common.Address) []byte { b := make([]byte, 32); copy(b[12:], addr[:]); return b }
func padU256(v *big.Int) []byte             { b := make([]byte, 32); v.FillBytes(b); return b }

// getEventNonce 查询合约 EVENT_NONCE（SCALE 编码，不在 Sol ABI 中）
// selector = keccak256("get_latest_nonce")[0:4]
func getEventNonce(t *testing.T, cli *ethclient.Client, contract common.Address) uint64 {
	t.Helper()
	// keccak256("get_latest_nonce") first 4 bytes
	sel := crypto.Keccak256Hash([]byte("get_latest_nonce")).Bytes()[:4]
	ret, err := callEVM(cli, contract, sel)
	if err != nil {
		t.Fatalf("get_latest_nonce: %v", err)
	}
	// SCALE u64: 8 bytes little-endian（至少 8 字节）
	if len(ret) < 8 {
		t.Fatalf("get_latest_nonce: short response (%d bytes)", len(ret))
	}
	return uint64(ret[0]) | uint64(ret[1])<<8 | uint64(ret[2])<<16 | uint64(ret[3])<<24 |
		uint64(ret[4])<<40 | uint64(ret[5])<<48 | uint64(ret[6])<<56
}
