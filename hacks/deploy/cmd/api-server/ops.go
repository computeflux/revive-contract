package main

// ops.go — 部署 / 升级 / 账户操作
//
// 复用 cmd/deploy-contract、cmd/deploy-full、cmd/upgrade-contract 的逻辑，
// 改为通过 logf 回调收集输出，供 API 返回给前端展示。

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"
	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/revive"
	"github.com/wetee-dao/ink.go/pallet/system"
	"github.com/wetee-dao/ink.go/util"
	"github.com/wetee-dao/tee-dsecret/pkg/model"
)

// logf 收集器：命令行的 fmt.Println → API 日志行
type logCollector struct {
	Lines []string `json:"lines"`
}

func (l *logCollector) logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	l.Lines = append(l.Lines, line)
	fmt.Println(line)
}

// ──────────────────────────────────────────────
// 环境配置
// ──────────────────────────────────────────────

type NodeConfig struct {
	Name            string `json:"name"`
	SS58            string `json:"ss58"`
	PSS58           string `json:"p_ss58"`
	Ip              string `json:"ip"`
	Port            uint32 `json:"port"`
	BlsValidatorKey string `json:"bls_validator_key"`
}

type GenesisConfig struct {
	Secrets    []NodeConfig `json:"secrets"`
	BootNodes  []uint64     `json:"boot_nodes"`
	Validators []uint64     `json:"validators"`
	Region     string       `json:"region"`
}

type EnvConfig struct {
	URL       string            `json:"url"`
	Suri      string            `json:"suri"`
	Contracts map[string]string `json:"contracts"`
	Genesis   GenesisConfig     `json:"genesis"`
}

// EnvPublic — 返回给前端的配置（脱敏 suri）
type EnvPublic struct {
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	HasSuri   bool              `json:"has_suri"`
	Contracts map[string]string `json:"contracts"`
	Genesis   GenesisConfig     `json:"genesis"`
}

func loadEnvConfig(env string) (*EnvConfig, error) {
	path := filepath.Join(*configDir, env+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env config %s: %w", path, err)
	}
	var cfg EnvConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse env config %s: %w", path, err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("missing url in env config %s", path)
	}
	if cfg.Suri == "" {
		return nil, fmt.Errorf("missing suri in env config %s", path)
	}
	return &cfg, nil
}

// listEnvs — 列出所有可用环境
func listEnvs() ([]string, error) {
	entries, err := os.ReadDir(*configDir)
	if err != nil {
		return nil, err
	}
	var envs []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		envs = append(envs, strings.TrimSuffix(name, ".json"))
	}
	return envs, nil
}

// ──────────────────────────────────────────────
// 客户端 / 签名器
// ──────────────────────────────────────────────

func newClient(cfg *EnvConfig) (*chain.ChainClient, error) {
	return chain.InitClient([]string{cfg.URL}, true)
}

func newSigner(cfg *EnvConfig, network uint16) (*chain.Signer, error) {
	pk, err := chain.Sr25519PairFromSecret(cfg.Suri, network)
	if err != nil {
		return nil, err
	}
	return &pk, nil
}

// ensureMapAccount — 检查并执行 map account，返回账户信息
type AccountInfo struct {
	SS58        string `json:"ss58"`
	H160        string `json:"h160"`
	FreeBalance string `json:"free_balance"`
	Mapped      bool   `json:"mapped"`
}

func ensureMapAccount(client *chain.ChainClient, pk chain.Signer, logf func(string, ...any)) (*AccountInfo, error) {
	ss58 := pk.SS58Address(42)
	h160 := pk.H160Address()

	logf("Account SS58: %s", ss58)
	logf("Account H160: %s", h160.Hex())

	accountInfo, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		return nil, fmt.Errorf("get account balance: %w", err)
	}
	logf("Account Free Balance: %s", accountInfo.Data.Free.Int.String())

	_, isSome, err := revive.GetOriginalAccountLatest(client.Api().RPC.State, h160)
	if err != nil {
		return nil, fmt.Errorf("get original account: %w", err)
	}
	if !isSome {
		runtimeCall := revive.MakeMapAccountCall()
		call, err := runtimeCall.AsCall()
		if err != nil {
			return nil, fmt.Errorf("make map account call: %w", err)
		}
		logf("MakeMapAccount for %s", ss58)
		if err := client.SignAndSubmit(&pk, call, true, 0); err != nil {
			return nil, fmt.Errorf("sign and submit map account: %w", err)
		}
		logf("MapAccount success")
	} else {
		logf("Account already mapped in revive")
	}

	return &AccountInfo{
		SS58:        ss58,
		H160:        h160.Hex(),
		FreeBalance: accountInfo.Data.Free.Int.String(),
		Mapped:      true,
	}, nil
}

// getAccountInfo — 只读账户信息（不做 map）
func getAccountInfo(client *chain.ChainClient, pk chain.Signer) (*AccountInfo, error) {
	ss58 := pk.SS58Address(42)
	h160 := pk.H160Address()

	accountInfo, err := system.GetAccountLatest(client.Api().RPC.State, pk.AccountID())
	if err != nil {
		return nil, fmt.Errorf("get account balance: %w", err)
	}
	_, isSome, err := revive.GetOriginalAccountLatest(client.Api().RPC.State, h160)
	if err != nil {
		return nil, fmt.Errorf("get original account: %w", err)
	}
	return &AccountInfo{
		SS58:        ss58,
		H160:        h160.Hex(),
		FreeBalance: accountInfo.Data.Free.Int.String(),
		Mapped:      isSome,
	}, nil
}

// ──────────────────────────────────────────────
// 单合约部署
// ──────────────────────────────────────────────

type DeployRequest struct {
	Env     string `json:"env"`
	Name    string `json:"name"`
	Dir     string `json:"dir"`     // 工作区根目录（含 target/），默认 "."
	Code    string `json:"code"`    // .polkavm 文件路径，默认 <dir>/target/<name>.release.polkavm
	Build   bool   `json:"build"`   // 是否先编译（cargo wrevive build）
	Network uint   `json:"network"` // ss58 network id，默认 42
}

type DeployResult struct {
	Address string   `json:"address"`
	Logs    []string `json:"logs"`
}

func deployContract(req DeployRequest) (*DeployResult, error) {
	lc := &logCollector{}
	logf := lc.logf

	if req.Env == "" {
		return nil, fmt.Errorf("missing env")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("missing name")
	}
	network := uint16(req.Network)
	if network == 0 {
		network = 42
	}

	envCfg, err := loadEnvConfig(req.Env)
	if err != nil {
		return nil, err
	}

	rootDir, err := filepath.Abs(req.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolve dir: %w", err)
	}
	codePath := req.Code
	if codePath == "" {
		codePath = filepath.Join(rootDir, "target", req.Name+".release.polkavm")
	}
	codePath, err = filepath.Abs(codePath)
	if err != nil {
		return nil, fmt.Errorf("resolve code path: %w", err)
	}

	if req.Build {
		// 编译：cargo wrevive build --manifest-path <dir>/Cargo.toml
		logf("Building contract %s ...", req.Name)
		if err := buildContract(rootDir, req.Name); err != nil {
			return nil, err
		}
	}

	code, err := os.ReadFile(codePath)
	if err != nil {
		return nil, fmt.Errorf("read contract code %s: %w", codePath, err)
	}
	logf("Contract code: %s (%d bytes)", codePath, len(code))

	client, err := newClient(envCfg)
	if err != nil {
		return nil, fmt.Errorf("init client: %w", err)
	}
	pk, err := newSigner(envCfg, network)
	if err != nil {
		return nil, fmt.Errorf("init signer: %w", err)
	}

	if _, err := ensureMapAccount(client, *pk, logf); err != nil {
		return nil, err
	}

	logf("Deploying %s ...", req.Name)
	address, err := client.DeployContract(
		util.InkCode{Upload: &code},
		pk,
		types.NewU128(*big.NewInt(0)),
		util.InkContractInput{
			Selector: "0x00000000",
			Args:     []any{},
		},
		util.NewSome(genSalt()),
	)
	if err != nil {
		return nil, fmt.Errorf("deploy %s: %w", req.Name, err)
	}

	logf("deploy success")
	logf("contract: %s", req.Name)
	logf("address: %s", address.Hex())

	return &DeployResult{Address: address.Hex(), Logs: lc.Lines}, nil
}

// ──────────────────────────────────────────────
// 全量部署（subnet + token + proxy + 创世初始化）
// ──────────────────────────────────────────────

type DeployFullRequest struct {
	Env     string `json:"env"`
	Dir     string `json:"dir"`
	Build   bool   `json:"build"`
	Network uint   `json:"network"`
}

type DeployFullResult struct {
	Subnet string   `json:"subnet"`
	Token  string   `json:"token"`
	Logs   []string `json:"logs"`
}

func deployFull(req DeployFullRequest) (*DeployFullResult, error) {
	lc := &logCollector{}
	logf := lc.logf

	if req.Env == "" {
		return nil, fmt.Errorf("missing env")
	}
	network := uint16(req.Network)
	if network == 0 {
		network = 42
	}

	envCfg, err := loadEnvConfig(req.Env)
	if err != nil {
		return nil, err
	}
	rootDir, err := filepath.Abs(req.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolve dir: %w", err)
	}
	targetDir := filepath.Join(rootDir, "target")

	client, err := newClient(envCfg)
	if err != nil {
		return nil, fmt.Errorf("init client: %w", err)
	}
	pk, err := newSigner(envCfg, network)
	if err != nil {
		return nil, fmt.Errorf("init signer: %w", err)
	}

	if _, err := ensureMapAccount(client, *pk, logf); err != nil {
		return nil, err
	}

	subnetAddr, err := deploySubnet(client, *pk, targetDir, logf)
	if err != nil {
		return nil, err
	}
	tokenAddr, err := deployToken(client, *pk, *subnetAddr, targetDir, logf)
	if err != nil {
		return nil, err
	}

	if err := initSubnetGenesis(client, *pk, subnetAddr.Hex(), envCfg.Genesis, logf); err != nil {
		return nil, err
	}

	logf("========================================")
	logf("subnet address (proxy) => %s", subnetAddr.Hex())
	logf("token  address (proxy) => %s", tokenAddr.Hex())
	logf("========================================")

	return &DeployFullResult{Subnet: subnetAddr.Hex(), Token: tokenAddr.Hex(), Logs: lc.Lines}, nil
}

func deploySubnet(client *chain.ChainClient, pk chain.Signer, targetDir string, logf func(string, ...any)) (*types.H160, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, "subnet.release.polkavm"))
	if err != nil {
		return nil, fmt.Errorf("read subnet code: %w", err)
	}

	res, err := subnet.DeploySubnetWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &data},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		return nil, fmt.Errorf("deploy subnet impl: %w", err)
	}
	logf("subnet implementation address: %s", res.Hex())

	proxyCode, err := os.ReadFile(filepath.Join(targetDir, "proxy.release.polkavm"))
	if err != nil {
		return nil, fmt.Errorf("read proxy code: %w", err)
	}
	subnetProxyAddress, err := proxy.DeployProxyWithNew(*res, util.NewSome(pk.H160Address()), chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &proxyCode},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		return nil, fmt.Errorf("deploy subnet proxy: %w", err)
	}
	logf("subnet proxy address: %s", subnetProxyAddress.Hex())

	subnetContract, err := subnet.InitSubnetContract(client, subnetProxyAddress.Hex())
	if err != nil {
		return nil, err
	}
	if err := subnetContract.ExecInit(chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}); err != nil {
		return nil, fmt.Errorf("subnet init: %w", err)
	}
	logf("subnet initialized")

	return subnetProxyAddress, nil
}

func deployToken(client *chain.ChainClient, pk chain.Signer, subnetAddress types.H160, targetDir string, logf func(string, ...any)) (*types.H160, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, "token.release.polkavm"))
	if err != nil {
		return nil, fmt.Errorf("read token code: %w", err)
	}

	res, err := token.DeployTokenWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &data},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		return nil, fmt.Errorf("deploy token impl: %w", err)
	}
	logf("token implementation address: %s", res.Hex())

	proxyCode, err := os.ReadFile(filepath.Join(targetDir, "proxy.release.polkavm"))
	if err != nil {
		return nil, fmt.Errorf("read proxy code: %w", err)
	}
	tokenProxyAddress, err := proxy.DeployProxyWithNew(*res, util.NewSome(pk.H160Address()), chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &proxyCode},
		Salt:   util.NewSome(genSalt()),
	})
	if err != nil {
		return nil, fmt.Errorf("deploy token proxy: %w", err)
	}
	logf("token proxy address: %s", tokenProxyAddress.Hex())

	tokenContract, err := token.InitTokenContract(client, tokenProxyAddress.Hex())
	if err != nil {
		return nil, err
	}

	if err := tokenContract.ExecInit(util.NewSome(pk.H160Address()), chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}); err != nil {
		return nil, fmt.Errorf("token init: %w", err)
	}
	if err := tokenContract.ExecSetSubnet(subnetAddress, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}); err != nil {
		return nil, fmt.Errorf("token set_subnet: %w", err)
	}
	logf("token initialized & linked to subnet")

	return tokenProxyAddress, nil
}

func initSubnetGenesis(client *chain.ChainClient, pk chain.Signer, subnetAddress string, cfg GenesisConfig, logf func(string, ...any)) error {
	_call := chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}
	subnetContract, err := subnet.InitSubnetContract(client, subnetAddress)
	if err != nil {
		return err
	}

	for _, node := range cfg.Secrets {
		v, err := model.PubKeyFromSS58(node.SS58)
		if err != nil {
			return fmt.Errorf("invalid ss58 %s: %w", node.SS58, err)
		}
		p, err := model.PubKeyFromSS58(node.PSS58)
		if err != nil {
			return fmt.Errorf("invalid p_ss58 %s: %w", node.PSS58, err)
		}
		nodeIp := ipToSubnet(node.Ip)
		blsKey, _ := hex.DecodeString(node.BlsValidatorKey)
		if err := subnetContract.ExecSecretRegister(
			[]byte(node.Name),
			v.AccountID(),
			p.AccountID(),
			nodeIp,
			node.Port,
			blsKey,
			_call,
		); err != nil {
			return fmt.Errorf("secret_register %s: %w", node.Name, err)
		}
		logf("%s register success", node.Name)
	}

	if len(cfg.BootNodes) > 0 {
		if err := subnetContract.ExecSetBootNodes(cfg.BootNodes, _call); err != nil {
			return fmt.Errorf("set_boot_nodes: %w", err)
		}
		logf("boot nodes set: %v", cfg.BootNodes)
	}

	for _, v := range cfg.Validators {
		if err := subnetContract.ExecValidatorJoin(v, _call); err != nil {
			return fmt.Errorf("validator_join %d: %w", v, err)
		}
		logf("validator %d joined", v)
	}

	return nil
}

func ipToSubnet(ipStr string) subnet.Ip {
	if isIP(ipStr) {
		ipv4, err := ipToUint32(ipStr)
		if err == nil {
			return subnet.Ip{
				Ipv4:   util.NewSome(ipv4),
				Ipv6:   util.NewNone[types.U128](),
				Domain: util.NewNone[[]byte](),
			}
		}
	}
	return subnet.Ip{
		Ipv4:   util.NewNone[uint32](),
		Ipv6:   util.NewNone[types.U128](),
		Domain: util.NewSome([]byte(ipStr)),
	}
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

func isIP(s string) bool {
	return net.ParseIP(s) != nil
}

// ──────────────────────────────────────────────
// 合约升级
// ──────────────────────────────────────────────

type UpgradeRequest struct {
	Env     string `json:"env"`
	Name    string `json:"name"` // token | subnet
	Dir     string `json:"dir"`
	Build   bool   `json:"build"`
	Network uint   `json:"network"`
}

type UpgradeResult struct {
	Proxy string   `json:"proxy"`
	Impl  string   `json:"impl"`
	Logs  []string `json:"logs"`
}

func upgradeContract(req UpgradeRequest) (*UpgradeResult, error) {
	lc := &logCollector{}
	logf := lc.logf

	if req.Env == "" {
		return nil, fmt.Errorf("missing env")
	}
	if req.Name != "token" && req.Name != "subnet" {
		return nil, fmt.Errorf("invalid contract name: %s (expected token, or subnet)", req.Name)
	}
	network := uint16(req.Network)
	if network == 0 {
		network = 42
	}

	envCfg, err := loadEnvConfig(req.Env)
	if err != nil {
		return nil, err
	}
	proxyAddr := envCfg.Contracts[req.Name]
	if proxyAddr == "" {
		return nil, fmt.Errorf("%s proxy address not found in config", req.Name)
	}

	rootDir, err := filepath.Abs(req.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolve dir: %w", err)
	}
	targetDir := filepath.Join(rootDir, "target")

	if req.Build {
		logf("Building contract %s ...", req.Name)
		if err := buildContract(rootDir, req.Name); err != nil {
			return nil, err
		}
	}

	code, err := os.ReadFile(filepath.Join(targetDir, req.Name+".release.polkavm"))
	if err != nil {
		return nil, fmt.Errorf("read %s code: %w", req.Name, err)
	}

	client, err := newClient(envCfg)
	if err != nil {
		return nil, fmt.Errorf("init client: %w", err)
	}
	pk, err := newSigner(envCfg, network)
	if err != nil {
		return nil, fmt.Errorf("init signer: %w", err)
	}

	if _, err := ensureMapAccount(client, *pk, logf); err != nil {
		return nil, err
	}

	var newImplAddr *types.H160
	if req.Name == "token" {
		addr, err := token.DeployTokenWithNew(chain.DeployParams{
			Client: client,
			Signer: pk,
			Code:   util.InkCode{Upload: &code},
			Salt:   util.NewSome(genSalt()),
		})
		if err != nil {
			return nil, fmt.Errorf("deploy token implementation: %w", err)
		}
		newImplAddr = addr
	} else {
		addr, err := subnet.DeploySubnetWithNew(chain.DeployParams{
			Client: client,
			Signer: pk,
			Code:   util.InkCode{Upload: &code},
			Salt:   util.NewSome(genSalt()),
		})
		if err != nil {
			return nil, fmt.Errorf("deploy subnet implementation: %w", err)
		}
		newImplAddr = addr
	}
	logf("%s implementation deployed: %s", req.Name, newImplAddr.Hex())

	proxyContract, err := proxy.InitProxyContract(client, proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("init %s proxy: %w", req.Name, err)
	}

	if err := proxyContract.ExecUpgrade(*newImplAddr, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}); err != nil {
		return nil, fmt.Errorf("upgrade %s proxy: %w", req.Name, err)
	}

	logf("========================================")
	logf("%s upgraded successfully", req.Name)
	logf("proxy address:     %s", proxyAddr)
	logf("new impl address:  %s", newImplAddr.Hex())
	logf("========================================")

	return &UpgradeResult{Proxy: proxyAddr, Impl: newImplAddr.Hex(), Logs: lc.Lines}, nil
}

// ──────────────────────────────────────────────
// 编译
// ──────────────────────────────────────────────

func buildContract(rootDir, name string) error {
	manifest := filepath.Join(rootDir, "Cargo.toml")
	if _, err := os.Stat(manifest); err != nil {
		return fmt.Errorf("no Cargo.toml found in %s", rootDir)
	}
	cmd := exec.Command("cargo", "wrevive", "build", "--manifest-path", manifest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cargo wrevive build %s: %w\n%s", name, err, string(out))
	}
	return nil
}

func genSalt() [32]byte {
	var salt [32]byte
	if _, err := rand.Read(salt[:]); err != nil {
		panic(err)
	}
	return salt
}
