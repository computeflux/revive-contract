package contracts

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"
	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/revive"
	"github.com/wetee-dao/ink.go/util"
)

// Config 对应 configs/<env>.json
type Config struct {
	URL       string `json:"url"`
	Suri      string `json:"suri"`
	Contracts struct {
		Subnet string `json:"subnet"`
		Token  string `json:"token"`
	} `json:"contracts"`
}

var env = flag.String("env", "test", "environment: local|test|main")

func loadConfig(t *testing.T) *Config {
	t.Helper()
	configPath := filepath.Join("configs", *env+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config %s: %v", configPath, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return &cfg
}

func newClient(t *testing.T, cfg *Config) *chain.ChainClient {
	t.Helper()
	client, err := chain.InitClient([]string{cfg.URL}, true)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func newSigner(t *testing.T, cfg *Config) chain.SignerType {
	t.Helper()
	pk, err := chain.Sr25519PairFromSecret(cfg.Suri, 42)
	if err != nil {
		t.Fatal(err)
	}
	return &pk
}

func TestTokenUpdate(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	tokenData, err := os.ReadFile("../../target/token.release.polkavm")
	if err != nil {
		t.Fatal(err)
	}

	salt := genSalt()
	res, err := token.DeployTokenWithNew(chain.DeployParams{
		Client: client,
		Signer: pk,
		Code:   util.InkCode{Upload: &tokenData},
		Salt:   util.NewSome(salt),
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("token address:", res.Hex())

	tokenIns, err := proxy.InitProxyContract(client, cfg.Contracts.Token)
	if err != nil {
		t.Fatal(err)
	}

	err = tokenIns.ExecUpgrade(*res, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("new token address:", res.Hex())
	fmt.Println("proxy address:", tokenIns.ContractAddress().Hex())
}

func TestSubnetUpdate(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	netData, err := os.ReadFile("../../target/subnet.release.polkavm")
	if err != nil {
		t.Fatal(err)
	}

	salt := genSalt()
	res, err := subnet.DeploySubnetWithNew(chain.DeployParams{
		Client: client,
		Signer: pk,
		Code:   util.InkCode{Upload: &netData},
		Salt:   util.NewSome(salt),
	})
	if err != nil {
		t.Fatal(err)
	}

	subnetIns, err := proxy.InitProxyContract(client, cfg.Contracts.Subnet)
	if err != nil {
		t.Fatal(err)
	}

	err = subnetIns.ExecUpgrade(*res, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("new subnet address:", res.Hex())
	fmt.Println("proxy address:", subnetIns.ContractAddress().Hex())
}

func TestMapAccount(t *testing.T) {
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)

	h160, _ := util.H160FromPublicKey(pk.Public())
	_, isSome, err := revive.GetOriginalAccountLatest(client.Api().RPC.State, h160)
	if err != nil {
		t.Fatal(err)
	}
	if !isSome {
		runtimeCall := revive.MakeMapAccountCall()
		call, err := (runtimeCall).AsCall()
		if err != nil {
			t.Fatal(err)
		}
		if err := client.SignAndSubmit(pk, call, true, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func genSalt() [32]byte {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	var randomBytes [32]byte
	copy(randomBytes[:], bytes)
	return randomBytes
}
