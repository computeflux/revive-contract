package main

// api-server — WeTEE 合约部署与调试 API 服务
//
// 把 hacks/deploy 下的命令行部署工具（deploy-contract / deploy-full / upgrade-contract）
// 以及合约绑定（token / subnet / proxy）封装为 REST API，供 Web UI 调用。
//
// 用法（在 hacks/deploy 目录下）:
//   go run ./cmd/api-server --port 8000
//
// 接口一览:
//   GET  /api/health                         健康检查
//   GET  /api/envs                           环境列表（脱敏）
//   GET  /api/envs/{env}                     环境详情 + 账户信息
//   POST /api/account/{env}/map              执行 map account
//   GET  /api/contracts/{env}/{name}/methods 合约方法清单（含参数定义）
//   POST /api/contracts/{env}/{name}/call    通用合约调用（exec / query）
//   POST /api/deploy                         单合约部署
//   POST /api/deploy-full                    全量部署（subnet + token + genesis）
//   POST /api/upgrade                        合约升级（token / subnet）

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	port      = flag.Int("port", 8000, "http listen port")
	configDir = flag.String("config-dir", "configs", "directory containing <env>.json configs")
	rootDir   = flag.String("root", ".", "workspace root directory (contains target/)")
	network   = flag.Uint("network", 42, "ss58 network id")
)

func main() {
	flag.Parse()

	if err := validateRegistry(); err != nil {
		log.Fatalf("方法注册表校验失败: %v", err)
	}

	mux := http.NewServeMux()

	// 前端静态资源（SPA）
	mux.Handle("/", spaHandler(staticFS))

	// 健康检查
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "weTEE contract api"})
	})

	// 环境
	mux.HandleFunc("GET /api/envs", handleListEnvs)
	mux.HandleFunc("GET /api/envs/{env}", handleGetEnv)

	// 账户
	mux.HandleFunc("POST /api/account/{env}/map", handleMapAccount)

	// 合约方法 / 调用
	mux.HandleFunc("GET /api/contracts/{env}/{name}/methods", handleListMethods)
	mux.HandleFunc("POST /api/contracts/{env}/{name}/call", handleContractCall)
	mux.HandleFunc("POST /api/contracts/{env}/{name}/batch", handleContractBatch)

	// 部署
	mux.HandleFunc("POST /api/deploy", handleDeploy)
	mux.HandleFunc("POST /api/deploy-full", handleDeployFull)
	mux.HandleFunc("POST /api/upgrade", handleUpgrade)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("WeTEE contract API server listening on http://0.0.0.0%s", addr)
	log.Printf("config-dir: %s, root: %s", *configDir, *rootDir)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// spaHandler — 前端静态资源服务（SPA 路由回退到 index.html）
func spaHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if f, err := dist.Open(name); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA 路由回退 → index.html
		idx, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "前端产物缺失，请先构建: cd ui && npm run build", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(idx)
	})
}

// ──────────────────────────────────────────────
// 中间件 / 工具
// ──────────────────────────────────────────────

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	return dec.Decode(v)
}

// ──────────────────────────────────────────────
// 环境
// ──────────────────────────────────────────────

func handleListEnvs(w http.ResponseWriter, r *http.Request) {
	envs, err := listEnvs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]EnvPublic, 0, len(envs))
	for _, name := range envs {
		cfg, err := loadEnvConfig(name)
		if err != nil {
			continue // 配置损坏则跳过
		}
		out = append(out, EnvPublic{
			Name:      name,
			URL:       cfg.URL,
			HasSuri:   cfg.Suri != "",
			Contracts: cfg.Contracts,
			Genesis:   cfg.Genesis,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"envs": out})
}

func handleGetEnv(w http.ResponseWriter, r *http.Request) {
	env := r.PathValue("env")
	cfg, err := loadEnvConfig(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	client, err := newClient(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("init client: %w", err))
		return
	}
	pk, err := newSigner(cfg, uint16(*network))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("init signer: %w", err))
		return
	}
	account, err := getAccountInfo(client, *pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get account: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"env": EnvPublic{
			Name:      env,
			URL:       cfg.URL,
			HasSuri:   cfg.Suri != "",
			Contracts: cfg.Contracts,
			Genesis:   cfg.Genesis,
		},
		"account": account,
	})
}

// ──────────────────────────────────────────────
// 账户
// ──────────────────────────────────────────────

func handleMapAccount(w http.ResponseWriter, r *http.Request) {
	env := r.PathValue("env")
	cfg, err := loadEnvConfig(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	client, err := newClient(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	pk, err := newSigner(cfg, uint16(*network))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	info, err := ensureMapAccount(client, *pk, log.Printf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account": info})
}

// ──────────────────────────────────────────────
// 合约方法 / 调用
// ──────────────────────────────────────────────

func handleListMethods(w http.ResponseWriter, r *http.Request) {
	env := r.PathValue("env")
	name := r.PathValue("name")

	spec, ok := contracts[name]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown contract: %s (expected token, subnet, proxy)", name))
		return
	}
	cfg, err := loadEnvConfig(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	address := cfg.Contracts[spec.AddrKey]

	writeJSON(w, http.StatusOK, map[string]any{
		"contract": spec.Name,
		"address":  address,
		"methods":  spec.Methods,
	})
}

func handleContractCall(w http.ResponseWriter, r *http.Request) {
	env := r.PathValue("env")
	name := r.PathValue("name")

	spec, ok := contracts[name]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown contract: %s", name))
		return
	}
	cfg, err := loadEnvConfig(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	address := cfg.Contracts[spec.AddrKey]
	if address == "" || address == "0x0000000000000000000000000000000000000000" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("contract %s 未部署，请先在配置中填写地址", name))
		return
	}

	var req CallRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if req.Method == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing method"))
		return
	}

	// 查找方法定义
	var methodSpec *MethodSpec
	for i := range spec.Methods {
		if spec.Methods[i].Name == req.Method {
			methodSpec = &spec.Methods[i]
			break
		}
	}
	if methodSpec == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown method: %s", req.Method))
		return
	}

	client, err := newClient(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	pk, err := newSigner(cfg, uint16(*network))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// 合约实例
	inst, err := spec.Init(client, address)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("init contract: %w", err))
		return
	}

	payAmount, err := parsePayAmount(req.PayAmount)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := callMethod(inst, methodSpec, req.Args, pk, payAmount)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"kind":   methodSpec.Kind,
			"method": req.Method,
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BatchCallRequest — 批量交易请求（等价于测试里的 BatchCall("batch_all", calls)）
type BatchCallRequest struct {
	Calls []CallRequest `json:"calls"`
}

func handleContractBatch(w http.ResponseWriter, r *http.Request) {
	env := r.PathValue("env")
	name := r.PathValue("name")

	spec, ok := contracts[name]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown contract: %s", name))
		return
	}
	cfg, err := loadEnvConfig(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	address := cfg.Contracts[spec.AddrKey]
	if address == "" || address == "0x0000000000000000000000000000000000000000" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("contract %s 未部署，请先在配置中填写地址", name))
		return
	}

	var req BatchCallRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if len(req.Calls) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("calls 不能为空"))
		return
	}

	client, err := newClient(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	pk, err := newSigner(cfg, uint16(*network))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	inst, err := spec.Init(client, address)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("init contract: %w", err))
		return
	}

	// 逐个构建 Call（CallOfXxx，只 dry-run 不提交）
	calls := make([]types.Call, 0, len(req.Calls))
	built := make([]string, 0, len(req.Calls))
	for i, cr := range req.Calls {
		var methodSpec *MethodSpec
		for j := range spec.Methods {
			if spec.Methods[j].Name == cr.Method {
				methodSpec = &spec.Methods[j]
				break
			}
		}
		if methodSpec == nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("calls[%d]: unknown method: %s", i, cr.Method))
			return
		}
		if methodSpec.Kind != "exec" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("calls[%d]: 批量只支持 exec 方法，%s 是 %s", i, cr.Method, methodSpec.Kind))
			return
		}
		payAmount, err := parsePayAmount(cr.PayAmount)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		call, err := callOfMethod(inst, methodSpec, cr.Args, pk, payAmount)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
			return
		}
		calls = append(calls, *call)
		built = append(built, cr.Method)
	}

	// batch_all 一次性提交
	batchCall, err := client.BatchCall("batch_all", calls)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("batch call error: %w", err))
		return
	}
	if err := client.SignAndSubmit(pk, *batchCall, true, 0); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("sign and submit: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"calls": built,
	})
}

// ──────────────────────────────────────────────
// 部署 / 升级
// ──────────────────────────────────────────────

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	var req DeployRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if req.Dir == "" {
		req.Dir = *rootDir
	}
	if req.Network == 0 {
		req.Network = *network
	}
	res, err := deployContract(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func handleDeployFull(w http.ResponseWriter, r *http.Request) {
	var req DeployFullRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if req.Dir == "" {
		req.Dir = *rootDir
	}
	if req.Network == 0 {
		req.Network = *network
	}
	res, err := deployFull(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func handleUpgrade(w http.ResponseWriter, r *http.Request) {
	var req UpgradeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if req.Dir == "" {
		req.Dir = *rootDir
	}
	if req.Network == 0 {
		req.Network = *network
	}
	res, err := upgradeContract(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

var _ = strings.TrimSpace // keep strings import if unused elsewhere

// validateRegistry — 启动时校验方法注册表：
// 每个注册方法必须存在于绑定类型上，且业务参数个数与方法签名一致
func validateRegistry() error {
	for _, spec := range contracts {
		if spec.InstType == nil {
			return fmt.Errorf("contract %s: InstType 未设置", spec.Name)
		}
		for _, ms := range spec.Methods {
			mv, ok := spec.InstType.MethodByName(ms.Method)
			if !ok {
				return fmt.Errorf("contract %s: 方法 %s 不存在于绑定（%s）", spec.Name, ms.Method, spec.InstType)
			}
			// 方法签名参数 = 接收者 + 业务参数 + 1（ExecParams / DryRunParams）
			if mv.Type.NumIn()-2 != len(ms.Params) {
				return fmt.Errorf("contract %s: 方法 %s 注册参数 %d 与签名 %d 不一致",
					spec.Name, ms.Method, len(ms.Params), mv.Type.NumIn()-2)
			}
		}
	}
	return nil
}
