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
	path := filepath.Join("..", "..", "..", "tee-node", "ext", "actives", "flux_config.json")
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

	// 0. 确保 signer 有足够的 FLUXT 余额（不足则 owner 调用 batchMint 铸造）
	bal := balanceOfToken(cli, fluxAddr, signer)
	fmt.Println("FLUXT balance:", bal.String())
	if bal.Cmp(amount) < 0 {
		mintCalldata := batchMintCalldata([]common.Address{signer}, []*big.Int{amount})
		// 注意：pallet-revive 对合约内 native 转账（0.5 ether/人）的 gas 要求极高，
		// gasLimit 必须给足（~500 万），否则执行到转账处 OOG revert
		txMint, err := sendEVMTx(cli, key, chainID, fluxAddr, nil, 5000000, mintCalldata)
		if err != nil {
			t.Fatal("batchMint:", err)
		}
		fmt.Println("batchMint:", txMint.Hash().Hex())
		waitReceipt(t, cli, txMint.Hash())
		fmt.Println("minted, balance:", balanceOfToken(cli, fluxAddr, signer).String())
	}

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
	waitReceipt(t, cli, tx1.Hash()) // 等待确认，确保 allowance 生效

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
	waitReceipt(t, cli, tx2.Hash())

	// 3. 查询充值后 EVENT_NONCE，验证 +1
	nonceAfter := getEventNonce(t, cli, uupsToken)
	if nonceAfter != nonceBefore+1 {
		t.Fatalf("EVENT_NONCE did not increment: before=%d, after=%d", nonceBefore, nonceAfter)
	}
	t.Logf("EVENT_NONCE: %d -> %d (OK)", nonceBefore, nonceAfter)
}

// waitReceipt 轮询等待交易确认，失败则直接 Fatal
func waitReceipt(t *testing.T, cli *ethclient.Client, hash common.Hash) *ethtypes.Receipt {
	t.Helper()
	for i := 0; i < 60; i++ {
		rcpt, err := cli.TransactionReceipt(context.Background(), hash)
		if err == nil {
			if rcpt.Status != 1 {
				t.Fatalf("tx %s failed with status %d", hash.Hex(), rcpt.Status)
			}
			return rcpt
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("tx %s not confirmed after 120s", hash.Hex())
	return nil
}

// balanceOfToken 手动 calldata 查询 ERC20 余额
func balanceOfToken(cli *ethclient.Client, token common.Address, who common.Address) *big.Int {
	sel := crypto.Keccak256Hash([]byte("balanceOf(address)")).Bytes()[:4]
	data := append(append([]byte{}, sel...), padAddress(who)...)
	ret, err := callEVM(cli, token, data)
	if err != nil || len(ret) < 32 {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(ret[:32])
}

// batchMintCalldata 手动 ABI 编码 batchMint(address[],uint256[])
func batchMintCalldata(tos []common.Address, amounts []*big.Int) []byte {
	sel := crypto.Keccak256Hash([]byte("batchMint(address[],uint256[])")).Bytes()[:4]
	headSize := 4 + 32 + 32 // selector + 2 个动态数组 offset
	data := append([]byte{}, sel...)
	data = append(data, padU256(big.NewInt(int64(headSize)))...)
	data = append(data, padU256(big.NewInt(int64(headSize+32+len(tos)*32)))...)
	data = append(data, padU256(big.NewInt(int64(len(tos))))...)
	for _, to := range tos {
		data = append(data, padAddress(to)...)
	}
	data = append(data, padU256(big.NewInt(int64(len(amounts))))...)
	for _, a := range amounts {
		data = append(data, padU256(a)...)
	}
	return data
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
