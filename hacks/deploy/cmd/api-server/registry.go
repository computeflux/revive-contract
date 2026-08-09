package main

// registry.go — 合约方法注册表
//
// 把 Go 绑定中的 ExecXxx / QueryXxx 方法注册为可通过 API 调用的通用方法。
// 前端通过 GET /api/contracts/{env}/{name}/methods 拿到方法清单后动态渲染表单，
// 提交的参数经 jsonutil 转换后，通过反射调用对应的绑定方法。

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"
	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
)

// ParamSpec — 方法参数定义（前端表单元数据）
type ParamSpec struct {
	Name string `json:"name"` // 参数名（ink 方法中的名称）
	Type string `json:"type"` // 展示类型：H160 / U128 / U256 / Account / bool / u8 / u32 / u64 / bytes / string / Option<...> / Ip / RunPrice / AssetInfo / u64[]
}

// MethodSpec — 方法定义
type MethodSpec struct {
	Name   string      `json:"name"` // ink 方法名，如 set_rate
	Kind   string      `json:"kind"` // exec | query
	Method string      `json:"-"`    // Go 绑定方法名，如 ExecSetRate
	Params []ParamSpec `json:"params"`
	// Payable 表示该方法需要/允许携带链上转账金额
	Payable bool `json:"payable"`
}

// ContractSpec — 合约定义
type ContractSpec struct {
	Name    string `json:"name"`
	AddrKey string `json:"-"` // 环境配置 contracts 中的键名
	// Init 根据地址创建合约实例（返回 *token.Token / *subnet.Subnet / *proxy.Proxy）
	Init func(client *chain.ChainClient, address string) (any, error)
	// InstType 实例类型（用于启动时校验方法签名）
	InstType reflect.Type
	// Methods 方法列表
	Methods []MethodSpec
}

// 参数构造器
func p(name, typ string) ParamSpec {
	return ParamSpec{Name: name, Type: typ}
}

func m(name, kind, method string, params ...ParamSpec) MethodSpec {
	if params == nil {
		params = []ParamSpec{} // 无参数方法返回空数组而非 null（前端防崩溃）
	}
	return MethodSpec{Name: name, Kind: kind, Method: method, Params: params}
}

func mPay(name, kind, method string, params ...ParamSpec) MethodSpec {
	ms := m(name, kind, method, params...)
	ms.Payable = true
	return ms
}

// ──────────────────────────────────────────────
// 合约注册
// ──────────────────────────────────────────────

var tokenContract = ContractSpec{
	Name:     "token",
	AddrKey:  "token",
	InstType: reflect.TypeOf(&token.Token{}),
	Init: func(client *chain.ChainClient, address string) (any, error) {
		return token.InitTokenContract(client, address)
	},
	Methods: []MethodSpec{
		// ── 交易 ──
		m("init", "exec", "ExecInit", p("owner", "Option<H160>")),
		m("set_subnet", "exec", "ExecSetSubnet", p("subnet_addr", "H160")),
		m("set_rate", "exec", "ExecSetRate", p("new_rate", "U256")),
		m("set_token_unit", "exec", "ExecSetTokenUnit", p("unit", "U256")),
		m("set_native_active", "exec", "ExecSetNativeActive", p("active", "bool")),
		m("set_erc20_token", "exec", "ExecSetErc20Token", p("token", "H160"), p("active", "bool"), p("rate", "U256"), p("unit", "U256")),
		mPay("recharge", "exec", "ExecRecharge"),
		m("withdraw", "exec", "ExecWithdraw", p("user", "H160"), p("points", "U256"), p("nonce", "u64")),
		m("withdraw_erc20", "exec", "ExecWithdrawErc20", p("token", "H160"), p("user", "H160"), p("points", "U256"), p("nonce", "u64")),
		mPay("recharge_erc20", "exec", "ExecRechargeErc20", p("token", "H160"), p("amount", "U256")),
		m("claim_withdrawal_sol", "exec", "ExecClaimWithdrawalSol", p("nonce", "u64")),
		m("cancel_withdrawal_sol", "exec", "ExecCancelWithdrawalSol", p("nonce", "u64")),
		// ── 查询 ──
		m("get_latest_nonce", "query", "QueryGetLatestNonce"),
		m("get_events", "query", "QueryGetEvents", p("from", "u64"), p("to", "u64")),
		m("get_event", "query", "QueryGetEvent", p("nonce", "u64")),
		m("to_points", "query", "QueryToPoints", p("dot_amount", "U256")),
		m("to_points_sol", "query", "QueryToPointsSol", p("dot_amount", "U256")),
		m("get_rate", "query", "QueryGetRate"),
		m("get_rate_sol", "query", "QueryGetRateSol"),
		m("get_token_unit", "query", "QueryGetTokenUnit"),
		m("get_native_active", "query", "QueryGetNativeActive"),
		m("get_native_active_sol", "query", "QueryGetNativeActiveSol"),
		m("get_subnet", "query", "QueryGetSubnet"),
		m("owner", "query", "QueryOwner"),
		m("owner_sol", "query", "QueryOwnerSol"),
		m("get_erc20_config", "query", "QueryGetErc20Config", p("token", "H160")),
		m("get_erc20_config_sol", "query", "QueryGetErc20ConfigSol", p("token", "H160")),
		m("get_erc20_count", "query", "QueryGetErc20Count"),
		m("get_erc20_count_sol", "query", "QueryGetErc20CountSol"),
		m("get_erc20_list", "query", "QueryGetErc20List"),
		m("get_erc20_list_sol", "query", "QueryGetErc20ListSol"),
		m("get_erc20_balance", "query", "QueryGetErc20Balance", p("token", "H160")),
		m("get_pending_withdrawal_sol", "query", "QueryGetPendingWithdrawalSol", p("nonce", "u64")),
	},
}

var subnetContract = ContractSpec{
	Name:     "subnet",
	AddrKey:  "subnet",
	InstType: reflect.TypeOf(&subnet.Subnet{}),
	Init: func(client *chain.ChainClient, address string) (any, error) {
		return subnet.InitSubnetContract(client, address)
	},
	Methods: []MethodSpec{
		// ── 交易 ──
		m("init", "exec", "ExecInit"),
		m("set_epoch_slot", "exec", "ExecSetEpochSlot", p("epoch_slot", "u32")),
		m("set_region", "exec", "ExecSetRegion", p("name", "bytes")),
		m("set_level_price", "exec", "ExecSetLevelPrice", p("level", "u8"), p("price", "RunPrice")),
		m("set_asset", "exec", "ExecSetAsset", p("info", "AssetInfo"), p("price", "U256")),
		m("set_min_mortgage", "exec", "ExecSetMinMortgage", p("amount", "U256")),
		m("set_level_min_mortgage", "exec", "ExecSetLevelMinMortgage", p("level", "u8"), p("amount", "U256")),
		m("slash_worker_mortgage", "exec", "ExecSlashWorkerMortgage", p("worker_id", "u64"), p("amount", "U256"), p("to", "H160")),
		m("worker_register", "exec", "ExecWorkerRegister", p("name", "bytes"), p("p2p_id", "Account"), p("ip", "Ip"), p("port", "u32"), p("level", "u8"), p("region_id", "u32")),
		m("worker_update", "exec", "ExecWorkerUpdate", p("id", "u64"), p("name", "bytes"), p("ip", "Ip"), p("port", "u32")),
		m("worker_mortgage", "exec", "ExecWorkerMortgage", p("id", "u64"), p("cpu", "u32"), p("mem", "u32"), p("cvm_cpu", "u32"), p("cvm_mem", "u32"), p("disk", "u32"), p("gpu", "u32"), p("deposit", "U256")),
		m("worker_unmortgage", "exec", "ExecWorkerUnmortgage", p("worker_id", "u64"), p("mortgage_id", "u32")),
		m("worker_start", "exec", "ExecWorkerStart", p("id", "u64")),
		m("worker_stop", "exec", "ExecWorkerStop", p("id", "u64")),
		m("set_boot_nodes", "exec", "ExecSetBootNodes", p("nodes", "u64[]")),
		m("secret_register", "exec", "ExecSecretRegister", p("name", "bytes"), p("validator_id", "Account"), p("p2p_id", "Account"), p("ip", "Ip"), p("port", "u32"), p("bls", "bytes")),
		m("secret_update", "exec", "ExecSecretUpdate", p("id", "u64"), p("name", "bytes"), p("ip", "Ip"), p("port", "u32")),
		mPay("secret_deposit", "exec", "ExecSecretDeposit", p("id", "u64"), p("deposit", "U256")),
		m("secret_delete", "exec", "ExecSecretDelete", p("id", "u64")),
		m("validator_join", "exec", "ExecValidatorJoin", p("id", "u64")),
		m("validator_delete", "exec", "ExecValidatorDelete", p("id", "u64")),
		m("set_next_epoch", "exec", "ExecSetNextEpoch", p("_node_id", "u64")),
		m("add_code_version", "exec", "ExecAddCodeVersion", p("signer", "bytes"), p("signature", "bytes")),
		m("delete_code_version", "exec", "ExecDeleteCodeVersion", p("id", "u64")),
		// ── 查询 ──
		m("epoch_info", "query", "QueryEpochInfo"),
		m("tee_chain_key", "query", "QueryTeeChainKey"),
		m("region", "query", "QueryRegion", p("id", "u32")),
		m("regions", "query", "QueryRegions"),
		m("level_price", "query", "QueryLevelPrice", p("level", "u8")),
		m("asset", "query", "QueryAsset", p("id", "u32")),
		m("min_mortgage", "query", "QueryMinMortgage"),
		m("level_min_mortgage", "query", "QueryLevelMinMortgage", p("level", "u8")),
		m("worker_total_resources", "query", "QueryWorkerTotalResources", p("worker_id", "u64")),
		m("worker_total_mortgage", "query", "QueryWorkerTotalMortgage", p("worker_id", "u64")),
		m("worker", "query", "QueryWorker", p("id", "u64")),
		m("workers", "query", "QueryWorkers", p("start", "Option<u64>"), p("size", "u64")),
		m("user_worker", "query", "QueryUserWorker", p("user", "H160")),
		m("mint_worker", "query", "QueryMintWorker", p("id", "Account")),
		m("boot_nodes", "query", "QueryBootNodes"),
		m("get_pending_secrets", "query", "QueryGetPendingSecrets"),
		m("secrets", "query", "QuerySecrets"),
		m("validators", "query", "QueryValidators"),
		m("next_epoch_validators", "query", "QueryNextEpochValidators"),
		m("code_version", "query", "QueryCodeVersion", p("id", "u64")),
		m("code_version_len", "query", "QueryCodeVersionLen"),
		m("code_versions", "query", "QueryCodeVersions"),
	},
}

var proxyContract = ContractSpec{
	Name:     "proxy",
	AddrKey:  "proxy",
	InstType: reflect.TypeOf(&proxy.Proxy{}),
	Init: func(client *chain.ChainClient, address string) (any, error) {
		return proxy.InitProxyContract(client, address)
	},
	Methods: []MethodSpec{
		m("get_implementation", "query", "QueryGetImplementation"),
		m("get_admin", "query", "QueryGetAdmin"),
		m("upgrade", "exec", "ExecUpgrade", p("implementation", "H160")),
		m("transfer_admin", "exec", "ExecTransferAdmin", p("new_admin", "H160")),
	},
}

var contracts = map[string]*ContractSpec{
	"token":  &tokenContract,
	"subnet": &subnetContract,
	"proxy":  &proxyContract,
}

// ──────────────────────────────────────────────
// 通用方法调用（反射）
// ──────────────────────────────────────────────

type CallRequest struct {
	Method    string         `json:"method"`     // ink 方法名
	Kind      string         `json:"kind"`       // exec | query（缺省时自动推断）
	Args      map[string]any `json:"args"`       // 参数名 → 值
	PayAmount string         `json:"pay_amount"` // 链上转账金额（可选，默认 0）
}

type CallResponse struct {
	Kind   string `json:"kind"`
	Method string `json:"method"`
	// exec 成功 → {ok: true}
	// query 成功 → {result: ..., gas: {...}}
	Result any `json:"result,omitempty"`
	Gas    any `json:"gas,omitempty"`
	Error  any `json:"error,omitempty"`
}

// callMethod — 通过反射调用绑定方法
func callMethod(inst any, spec *MethodSpec, args map[string]any, signer chain.SignerType, payAmount types.U128) (CallResponse, error) {
	mv := reflect.ValueOf(inst).MethodByName(spec.Method)
	if !mv.IsValid() {
		return CallResponse{}, fmt.Errorf("绑定方法不存在: %s", spec.Method)
	}

	callArgs, err := prepareArgs(mv, spec, args, signer, payAmount, spec.Kind == "query")
	if err != nil {
		return CallResponse{}, err
	}

	// 调用
	outs := mv.Call(callArgs)
	if spec.Kind == "query" {
		if err, ok := outs[2].Interface().(error); ok && err != nil {
			return CallResponse{}, err
		}
		resp := CallResponse{Kind: "query", Method: spec.Name}
		if !outs[0].IsNil() {
			resp.Result = toJSON(outs[0].Interface())
		}
		if !outs[1].IsNil() {
			resp.Gas = toJSON(outs[1].Interface())
		}
		return resp, nil
	}

	// exec
	if err, ok := outs[0].Interface().(error); ok && err != nil {
		return CallResponse{}, err
	}
	return CallResponse{Kind: "exec", Method: spec.Name, Result: map[string]any{"ok": true}}, nil
}

// callOfMethodName — 由 Exec 方法推导 CallOf 方法名（绑定自动生成 CallOfXxx）
func callOfMethodName(spec *MethodSpec) string {
	return "CallOf" + strings.TrimPrefix(spec.Method, "Exec")
}

// callOfMethod — 调用 CallOfXxx 构建链上 Call（不提交），用于 batch_all 批量交易
func callOfMethod(inst any, spec *MethodSpec, args map[string]any, signer chain.SignerType, payAmount types.U128) (*types.Call, error) {
	mv := reflect.ValueOf(inst).MethodByName(callOfMethodName(spec))
	if !mv.IsValid() {
		return nil, fmt.Errorf("批量方法不存在: %s（该函数可能不支持批量）", callOfMethodName(spec))
	}

	// CallOfXxx 接收 DryRunParams 作为最后一个参数
	callArgs, err := prepareArgs(mv, spec, args, signer, payAmount, true)
	if err != nil {
		return nil, err
	}

	outs := mv.Call(callArgs)
	if err, ok := outs[1].Interface().(error); ok && err != nil {
		return nil, err
	}
	if outs[0].IsNil() {
		return nil, fmt.Errorf("%s: 构建 Call 返回 nil", spec.Name)
	}
	call, ok := outs[0].Interface().(*types.Call)
	if !ok {
		return nil, fmt.Errorf("%s: 返回类型不是 *types.Call", spec.Name)
	}
	return call, nil
}

// prepareArgs — 转换业务参数并追加最后的执行参数（ExecParams / DryRunParams）
func prepareArgs(mv reflect.Value, spec *MethodSpec, args map[string]any, signer chain.SignerType, payAmount types.U128, isQuery bool) ([]reflect.Value, error) {
	mt := mv.Type()

	// 参数个数校验：方法参数 = 业务参数 + 1（ExecParams / DryRunParams）
	expectArgs := mt.NumIn() - 1
	if len(spec.Params) != expectArgs {
		return nil, fmt.Errorf("注册参数 %d 与方法签名 %d 不一致", len(spec.Params), expectArgs)
	}

	// 转换业务参数
	callArgs := make([]reflect.Value, 0, mt.NumIn())
	for _, ps := range spec.Params {
		raw, ok := args[ps.Name]
		if !ok {
			// 允许缺省：Option 类型视为 None，其余报错
			if isOptionalDisplayType(ps.Type) {
				raw = nil
			} else {
				return nil, fmt.Errorf("缺少参数: %s (%s)", ps.Name, ps.Type)
			}
		}
		cv, err := convertArg(raw, mt.In(len(callArgs)))
		if err != nil {
			return nil, fmt.Errorf("参数 %s: %w", ps.Name, err)
		}
		callArgs = append(callArgs, cv)
	}

	// 最后一个参数：ExecParams / DryRunParams
	if isQuery {
		param := chain.DefaultParamWithOrigin(signer.AccountID())
		param.PayAmount = payAmount
		callArgs = append(callArgs, reflect.ValueOf(param))
	} else {
		callArgs = append(callArgs, reflect.ValueOf(chain.ExecParams{
			Signer:         signer,
			PayAmount:      payAmount,
			UntilFinalized: true,
		}))
	}
	return callArgs, nil
}

// isOptionalDisplayType — 展示类型是否允许缺省（Option<...> 或可选数组）
func isOptionalDisplayType(display string) bool {
	return len(display) >= 7 && display[:7] == "Option<"
}

// parsePayAmount — 解析转账金额（默认 0）
func parsePayAmount(s string) (types.U128, error) {
	if s == "" {
		return types.NewU128(*big.NewInt(0)), nil
	}
	n, err := toBigInt(s)
	if err != nil {
		return types.U128{}, fmt.Errorf("pay_amount: %w", err)
	}
	return types.NewU128(*n), nil
}
