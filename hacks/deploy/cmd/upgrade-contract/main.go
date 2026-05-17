package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"wetee/test/contracts/cloud"
	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/revive"
	"github.com/wetee-dao/ink.go/pallet/system"
	"github.com/wetee-dao/ink.go/util"
)

type EnvConfig struct {
	URL       string            `json:"url"`
	Suri      string            `json:"suri"`
	Contracts map[string]string `json:"contracts"`
}

func loadEnvConfig(env string) (EnvConfig, error) {
	if env == "" {
		return EnvConfig{}, fmt.Errorf("missing required flag: -env")
	}
	path := filepath.Join("configs", env+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return EnvConfig{}, fmt.Errorf("read env config %s: %w", path, err)
	}
	var cfg EnvConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return EnvConfig{}, fmt.Errorf("parse env config %s: %w", path, err)
	}
	if cfg.URL == "" {
		return EnvConfig{}, fmt.Errorf("missing url in env config %s", path)
	}
	if cfg.Suri == "" {
		return EnvConfig{}, fmt.Errorf("missing suri in env config %s", path)
	}
	return cfg, nil
}

func main() {
	var (
		env     string
		name    string
		podID   uint64
		dir     string
		network uint
	)

	const podIDUnset = ^uint64(0)

	flag.StringVar(&env, "env", "", "environment: local | test | main (loads configs/<env>.json)")
	flag.StringVar(&name, "name", "", "contract to upgrade: cloud | subnet | pod-code | pod-contract")
	flag.Uint64Var(&podID, "pod-id", podIDUnset, "pod id (required when name=pod-contract)")
	flag.StringVar(&dir, "dir", ".", "workspace root directory (contains target/)")
	flag.UintVar(&network, "network", 42, "ss58 network id")
	flag.Parse()

	if name == "" {
		exitf("missing required flag: -name (cloud | subnet | pod-code | pod-contract)")
	}

	validNames := map[string]bool{"cloud": true, "subnet": true, "pod-code": true, "pod-contract": true}
	if !validNames[name] {
		exitf("invalid contract name: %s (expected cloud, subnet, pod-code, or pod-contract)", name)
	}

	if name == "pod-contract" && podID == podIDUnset {
		exitf("missing required flag: -pod-id (required when name=pod-contract)")
	}

	// Load env config
	envCfg, err := loadEnvConfig(env)
	if err != nil {
		exitf("load env config: %v", err)
	}

	rootDir, err := filepath.Abs(dir)
	if err != nil {
		exitf("resolve dir: %v", err)
	}

	client, err := chain.InitClient([]string{envCfg.URL}, true)
	if err != nil {
		exitf("init client: %v", err)
	}

	pk, err := chain.Sr25519PairFromSecret(envCfg.Suri, uint16(network))
	if err != nil {
		exitf("init signer: %v", err)
	}

	// show account info and ensure map account
	ensureMapAccount(client, pk, uint16(network))

	targetDir := filepath.Join(rootDir, "target")
	callParams := chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}

	switch name {
	case "cloud":
		upgradeCloud(client, pk, envCfg, targetDir, callParams)
	case "subnet":
		upgradeSubnet(client, pk, envCfg, targetDir, callParams)
	case "pod-code":
		upgradePodCode(client, pk, envCfg, targetDir, callParams)
	case "pod-contract":
		upgradePodContract(client, pk, envCfg, callParams, podID)
	}
}

func upgradeCloud(client *chain.ChainClient, pk chain.Signer, envCfg EnvConfig, targetDir string, callParams chain.ExecParams) {
	proxyAddr := envCfg.Contracts["cloud"]
	if proxyAddr == "" {
		exitf("cloud proxy address not found in config")
	}

	code, err := os.ReadFile(filepath.Join(targetDir, "cloud.release.polkavm"))
	if err != nil {
		exitf("read cloud code: %v", err)
	}

	newImplAddr, err := cloud.DeployCloudWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &code},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		exitf("deploy cloud implementation: %v", err)
	}
	fmt.Println("cloud implementation deployed: ", newImplAddr.Hex())

	proxyContract, err := proxy.InitProxyContract(client, proxyAddr)
	if err != nil {
		exitf("init cloud proxy: %v", err)
	}

	err = proxyContract.ExecUpgrade(*newImplAddr, callParams)
	if err != nil {
		exitf("upgrade cloud proxy: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("cloud upgraded successfully")
	fmt.Println("proxy address:     ", proxyAddr)
	fmt.Println("new impl address:  ", newImplAddr.Hex())
	fmt.Println("========================================")
}

func upgradeSubnet(client *chain.ChainClient, pk chain.Signer, envCfg EnvConfig, targetDir string, callParams chain.ExecParams) {
	proxyAddr := envCfg.Contracts["subnet"]
	if proxyAddr == "" {
		exitf("subnet proxy address not found in config")
	}

	code, err := os.ReadFile(filepath.Join(targetDir, "subnet.release.polkavm"))
	if err != nil {
		exitf("read subnet code: %v", err)
	}

	newImplAddr, err := subnet.DeploySubnetWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &code},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		exitf("deploy subnet implementation: %v", err)
	}
	fmt.Println("subnet implementation deployed: ", newImplAddr.Hex())

	proxyContract, err := proxy.InitProxyContract(client, proxyAddr)
	if err != nil {
		exitf("init subnet proxy: %v", err)
	}

	err = proxyContract.ExecUpgrade(*newImplAddr, callParams)
	if err != nil {
		exitf("upgrade subnet proxy: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("subnet upgraded successfully")
	fmt.Println("proxy address:     ", proxyAddr)
	fmt.Println("new impl address:  ", newImplAddr.Hex())
	fmt.Println("========================================")
}

func upgradePodCode(client *chain.ChainClient, pk chain.Signer, envCfg EnvConfig, targetDir string, callParams chain.ExecParams) {
	cloudAddr := envCfg.Contracts["cloud"]
	if cloudAddr == "" {
		exitf("cloud address not found in config (required for pod-code upgrade)")
	}

	code, err := os.ReadFile(filepath.Join(targetDir, "pod.release.polkavm"))
	if err != nil {
		exitf("read pod code: %v", err)
	}

	codeHash, err := client.UploadInkCode(code, &pk)
	if err != nil {
		exitf("upload pod code: %v", err)
	}
	fmt.Println("pod code uploaded: ", codeHash.Hex())

	// Deploy the new Pod implementation from uploaded code hash
	podImplAddr, err := client.DeployContract(
		util.InkCode{Existing: codeHash},
		&pk,
		types.NewU128(*big.NewInt(0)),
		util.InkContractInput{
			Selector: "0x00000000",
			Args:     []any{},
		},
		util.NewSome(genSalt()),
	)
	if err != nil {
		exitf("deploy pod impl: %v", err)
	}
	fmt.Println("pod impl deployed: ", podImplAddr.Hex())

	cloudContract, err := cloud.InitCloudContract(client, cloudAddr)
	if err != nil {
		exitf("init cloud contract: %v", err)
	}

	err = cloudContract.ExecSetPodImpl(*podImplAddr, callParams)
	if err != nil {
		exitf("set pod impl: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("pod code upgraded successfully")
	fmt.Println("cloud address:     ", cloudAddr)
	fmt.Println("new pod code hash: ", codeHash.Hex())
	fmt.Println("========================================")
}

func upgradePodContract(client *chain.ChainClient, pk chain.Signer, envCfg EnvConfig, callParams chain.ExecParams, podID uint64) {
	cloudAddr := envCfg.Contracts["cloud"]
	if cloudAddr == "" {
		exitf("cloud address not found in config (required for pod-contract upgrade)")
	}

	cloudContract, err := cloud.InitCloudContract(client, cloudAddr)
	if err != nil {
		exitf("init cloud contract: %v", err)
	}

	err = cloudContract.ExecUpdatePodContract(podID, callParams)
	if err != nil {
		exitf("update pod contract: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("pod contract upgraded successfully")
	fmt.Println("cloud address:     ", cloudAddr)
	fmt.Println("pod id:            ", podID)
	fmt.Println("========================================")
}

func ensureMapAccount(client *chain.ChainClient, pk chain.Signer, network uint16) {
	ss58 := pk.SS58Address(network)
	h160 := pk.H160Address()

	fmt.Println("Account SS58:", ss58)
	fmt.Println("Account H160:", h160.Hex())

	accountInfo, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		exitf("get account balance: %v", err)
	}
	fmt.Println("Account Free Balance:", accountInfo.Data.Free)

	_, isSome, err := revive.GetOriginalAccountLatest(client.Api().RPC.State, h160)
	if err != nil {
		exitf("get original account: %v", err)
	}
	if !isSome {
		runtimeCall := revive.MakeMapAccountCall()
		call, err := runtimeCall.AsCall()
		if err != nil {
			exitf("make map account call: %v", err)
		}

		fmt.Println("MakeMapAccount for", ss58)
		err = client.SignAndSubmit(&pk, call, true, 0)
		if err != nil {
			exitf("sign and submit map account: %v", err)
		}
		fmt.Println("MapAccount success")
	} else {
		fmt.Println("Account already mapped in revive")
	}
}

func genSalt() [32]byte {
	var salt [32]byte
	if _, err := rand.Read(salt[:]); err != nil {
		exitf("gen salt: %v", err)
	}
	return salt
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
