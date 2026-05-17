package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net"
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
	"github.com/wetee-dao/tee-dsecret/pkg/model"
)

type NodeConfig struct {
	Name  string `json:"name"`
	SS58  string `json:"ss58"`
	PSS58 string `json:"p_ss58"`
	Ip    string `json:"ip"`
	Port  uint32 `json:"port"`
}

type WorkerConfig struct {
	Name     string `json:"name"`
	SS58     string `json:"ss58"`
	Domain   string `json:"domain"`
	Port     uint32 `json:"port"`
	Level    byte   `json:"level"`
	Region   uint32 `json:"region"`
	Cpu      uint32 `json:"cpu"`
	Memory   uint32 `json:"memory"`
	Disk     uint32 `json:"disk"`
	Gpu      uint32 `json:"gpu"`
	Mortgage int64  `json:"mortgage"`
}

type GenesisConfig struct {
	Secrets    []NodeConfig   `json:"secrets"`
	BootNodes  []uint64       `json:"boot_nodes"`
	Validators []uint64       `json:"validators"`
	Region     string         `json:"region"`
	Workers    []WorkerConfig `json:"workers"`
}

type EnvConfig struct {
	URL     string        `json:"url"`
	Suri    string        `json:"suri"`
	Genesis GenesisConfig `json:"genesis"`
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

func ipToUint32(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP: %s", ipStr)
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, fmt.Errorf("not an IPv4: %s", ipStr)
	}
	return uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3]), nil
}

func main() {
	var (
		network uint
		dir     string
		env     string
	)

	flag.UintVar(&network, "network", 42, "ss58 network id")
	flag.StringVar(&dir, "dir", ".", "workspace root directory (contains target/)")
	flag.StringVar(&env, "env", "", "environment: local | test | main (loads configs/<env>.json)")
	flag.Parse()

	rootDir, err := filepath.Abs(dir)
	if err != nil {
		exitf("resolve dir: %v", err)
	}

	// Load all config from JSON file
	envCfg, err := loadEnvConfig(env)
	if err != nil {
		exitf("load env config: %v", err)
	}

	client, err := chain.InitClient([]string{envCfg.URL}, true)
	if err != nil {
		exitf("init client: %v", err)
	}

	pk, err := chain.Sr25519PairFromSecret(envCfg.Suri, uint16(network))
	if err != nil {
		exitf("init signer: %v", err)
	}

	genesisCfg := envCfg.Genesis

	// show account info and ensure map account
	ensureMapAccount(client, pk)

	targetDir := filepath.Join(rootDir, "target")

	// Upload and deploy pod implementation contract
	podData, err := os.ReadFile(filepath.Join(targetDir, "pod.release.polkavm"))
	if err != nil {
		exitf("read pod code: %v", err)
	}
	podCodeHash, err := client.UploadInkCode(podData, &pk)
	if err != nil {
		exitf("upload pod code: %v", err)
	}

	// Deploy Pod implementation contract (standalone, Proxy will delegate_call to this)
	podImplAddress, err := client.DeployContract(
		util.InkCode{Existing: podCodeHash},
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
	fmt.Println("pod impl address: ", podImplAddress.Hex())

	// Deploy full system
	subnetImplAddress, _ := deploySubnetContract(client, pk, targetDir)
	cloudProxyAddress, _ := deployCloudContract(client, *subnetImplAddress, *podImplAddress, *podCodeHash, pk, targetDir)

	initSubnet(client, pk, subnetImplAddress.Hex(), genesisCfg)
	initWorker(client, pk, subnetImplAddress.Hex(), genesisCfg)

	fmt.Println("========================================")
	fmt.Println("subnet address (proxy2) => ", subnetImplAddress.Hex())
	fmt.Println("cloud  address (proxy1) => ", cloudProxyAddress.Hex())
	fmt.Println("========================================")
}

func deploySubnetContract(client *chain.ChainClient, pk chain.Signer, targetDir string) (*types.H160, *subnet.Subnet) {
	data, err := os.ReadFile(filepath.Join(targetDir, "subnet.release.polkavm"))
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	res, err := subnet.DeploySubnetWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &data},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		util.LogWithPurple("DeployContract", err)
		panic(err)
	}
	fmt.Println("subnet address: ", res.Hex())

	proxyCode, err := os.ReadFile(filepath.Join(targetDir, "proxy.release.polkavm"))
	if err != nil {
		util.LogWithPurple("read proxy file error", err)
		panic(err)
	}
	subnetProxyAddress, err := proxy.DeployProxyWithNew(*res, util.NewSome(pk.H160Address()), chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &proxyCode},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		util.LogWithPurple("DeployProxyWithNew", err)
		panic(err)
	}
	fmt.Println("subnet proxy address: ", subnetProxyAddress.Hex())

	subnetContract, err := subnet.InitSubnetContract(client, subnetProxyAddress.Hex())
	if err != nil {
		panic(err)
	}

	err = subnetContract.ExecInit(chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}

	return subnetProxyAddress, subnetContract
}

func deployCloudContract(client *chain.ChainClient, subnetAddress types.H160, podImplAddress types.H160, podCodeHash types.H256, pk chain.Signer, targetDir string) (*types.H160, *cloud.Cloud) {
	data, err := os.ReadFile(filepath.Join(targetDir, "cloud.release.polkavm"))
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	res, err := cloud.DeployCloudWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &data},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		util.LogWithPurple("DeployContract", err)
		panic(err)
	}
	fmt.Println("cloud address: ", res.Hex())

	proxyCode, err := os.ReadFile(filepath.Join(targetDir, "proxy.release.polkavm"))
	if err != nil {
		util.LogWithPurple("read proxy file error", err)
		panic(err)
	}
	cloudProxyAddress, err := proxy.DeployProxyWithNew(*res, util.NewSome(pk.H160Address()), chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &proxyCode},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		util.LogWithPurple("DeployProxyWithNew", err)
		panic(err)
	}
	fmt.Println("cloud proxy address: ", cloudProxyAddress.Hex())

	cloudContract, err := cloud.InitCloudContract(client, cloudProxyAddress.Hex())
	if err != nil {
		panic(err)
	}
	err = cloudContract.ExecInit(subnetAddress, podImplAddress, podCodeHash, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}

	return cloudProxyAddress, cloudContract
}

func initSubnet(client *chain.ChainClient, pk chain.Signer, subnetAddress string, cfg GenesisConfig) {
	_call := chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}
	subnetContract, err := subnet.InitSubnetContract(client, subnetAddress)
	if err != nil {
		panic(err)
	}

	for _, node := range cfg.Secrets {
		v, err := model.PubKeyFromSS58(node.SS58)
		if err != nil {
			panic(fmt.Sprintf("invalid ss58 %s: %v", node.SS58, err))
		}
		p, err := model.PubKeyFromSS58(node.PSS58)
		if err != nil {
			panic(fmt.Sprintf("invalid p_ss58 %s: %v", node.PSS58, err))
		}
		ipv4, err := ipToUint32(node.Ip)
		if err != nil {
			panic(fmt.Sprintf("invalid ip %s: %v", node.Ip, err))
		}
		err = subnetContract.ExecSecretRegister(
			[]byte(node.Name),
			v.AccountID(),
			p.AccountID(),
			subnet.Ip{
				Ipv4:   util.NewSome(ipv4),
				Ipv6:   util.NewNone[types.U128](),
				Domain: util.NewNone[[]byte](),
			},
			node.Port,
			_call,
		)
		fmt.Printf("%s register result: %v\n", node.Name, err)
	}

	err = subnetContract.ExecSetBootNodes(cfg.BootNodes, _call)
	if err != nil {
		panic(err)
	}

	for _, v := range cfg.Validators {
		err = subnetContract.ExecValidatorJoin(v, _call)
		if err != nil {
			panic(err)
		}
	}
}

func initWorker(client *chain.ChainClient, pk chain.Signer, subnetAddress string, cfg GenesisConfig) {
	_call := chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}

	subnetContract, err := subnet.InitSubnetContract(client, subnetAddress)
	if err != nil {
		panic(err)
	}

	err = subnetContract.ExecSetRegion([]byte(cfg.Region), _call)
	if err != nil {
		panic(err)
	}

	err = subnetContract.ExecSetLevelPrice(0, subnet.RunPrice{
		CpuPer:       1,
		CvmCpuPer:    1,
		MemoryPer:    1,
		CvmMemoryPer: 1,
		DiskPer:      1,
		GpuPer:       1,
	}, _call)
	if err != nil {
		panic(err)
	}

	name := []byte("T")
	err = subnetContract.ExecSetAsset(subnet.AssetInfo{
		Native: &name,
	}, types.NewU256(*big.NewInt(1000)), chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}

	for _, w := range cfg.Workers {
		pubkey, err := model.PubKeyFromSS58(w.SS58)
		if err != nil {
			panic(fmt.Sprintf("invalid worker ss58 %s: %v", w.SS58, err))
		}
		err = subnetContract.ExecWorkerRegister(
			[]byte(w.Name),
			pubkey.AccountID(),
			subnet.Ip{
				Ipv4:   util.NewNone[uint32](),
				Ipv6:   util.NewNone[types.U128](),
				Domain: util.NewSome([]byte(w.Domain)),
			},
			w.Port,
			w.Level,
			w.Region,
			_call,
		)
		if err != nil {
			panic(err)
		}

		err = subnetContract.ExecWorkerMortgage(
			0,
			w.Cpu, w.Memory,
			0, 0, // cvm_cpu, cvm_mem
			w.Disk, w.Gpu, // disk, gpu
			types.NewU256(*big.NewInt(w.Mortgage)),
			chain.ExecParams{
				Signer:    &pk,
				PayAmount: types.NewU128(*big.NewInt(w.Mortgage)),
			},
		)
		if err != nil {
			fmt.Println("error:", err)
			// panic(err)
		}
	}
}

func ensureMapAccount(client *chain.ChainClient, pk chain.Signer) {
	ss58 := pk.SS58Address(42)
	h160 := pk.H160Address()

	// show account public key (SS58) and balance
	fmt.Println("Account SS58:", ss58)
	fmt.Println("Account H160:", h160.Hex())

	accountInfo, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		exitf("get account balance: %v", err)
	}
	fmt.Println("Account Free Balance:", accountInfo.Data.Free)

	// check account is mapped in revive
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
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	randomBytes := [32]byte{}
	copy(randomBytes[:], bytes)

	return randomBytes
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
