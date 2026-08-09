// api.ts — WeTEE 合约部署与调试 API 客户端
//
// 后端: hacks/deploy/cmd/api-server（内嵌 UI，同源服务）
// 默认请求同源 /api，可用 VITE_API_URL 覆盖

export function getApiBase(): string {
  const raw = import.meta.env.VITE_API_URL as string | undefined;
  if (raw) return raw.trim().replace(/\/$/, "");
  // 同源模式：UI 由 api-server 托管，直接请求 /api（无 CORS）
  return "";
}

// ──────────────────────────────────────────────
// 类型定义
// ──────────────────────────────────────────────

export interface EnvPublic {
  name: string;
  url: string;
  has_suri: boolean;
  contracts: Record<string, string>;
  genesis: {
    secrets: Array<{ name: string; ss58: string; p_ss58: string; ip: string; port: number; bls_validator_key: string }>;
    boot_nodes: number[];
    validators: number[];
    region: string;
  };
}

export interface AccountInfo {
  ss58: string;
  h160: string;
  free_balance: string;
  mapped: boolean;
}

export interface EnvDetail {
  env: EnvPublic;
  account: AccountInfo;
}

export interface ParamSpec {
  name: string;
  type: string;
}

export interface MethodSpec {
  name: string;
  kind: "exec" | "query";
  params: ParamSpec[];
  payable: boolean;
}

export interface ContractMethods {
  contract: string;
  address: string;
  methods: MethodSpec[];
}

export interface CallResponse {
  kind: "exec" | "query";
  method: string;
  result?: any;
  gas?: any;
  error?: string;
}

export interface DeployResponse {
  address?: string;
  subnet?: string;
  token?: string;
  proxy?: string;
  impl?: string;
  logs?: string[];
  error?: string;
}

// ──────────────────────────────────────────────
// 基础请求
// ──────────────────────────────────────────────

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${getApiBase()}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
  });
  if (!res.ok) {
    const data = await res.json().catch(() => null);
    throw new Error(data?.error || `请求失败: ${res.status} ${path}`);
  }
  return res.json();
}

// ──────────────────────────────────────────────
// 环境 / 账户
// ──────────────────────────────────────────────

export async function fetchEnvs(): Promise<{ envs: EnvPublic[] }> {
  return request("/api/envs");
}

export async function fetchEnvDetail(env: string): Promise<EnvDetail> {
  return request(`/api/envs/${env}`);
}

export async function mapAccount(env: string): Promise<{ ok: boolean; account: AccountInfo }> {
  return request(`/api/account/${env}/map`, { method: "POST" });
}

// ──────────────────────────────────────────────
// 合约方法 / 调用
// ──────────────────────────────────────────────

export async function fetchContractMethods(env: string, name: string): Promise<ContractMethods> {
  return request(`/api/contracts/${env}/${name}/methods`);
}

export async function callContract(
  env: string,
  name: string,
  method: string,
  kind: "exec" | "query",
  args: Record<string, any>,
  payAmount = "",
): Promise<CallResponse> {
  return request(`/api/contracts/${env}/${name}/call`, {
    method: "POST",
    body: JSON.stringify({ method, kind, args, pay_amount: payAmount }),
  });
}

export interface BatchItem {
  method: string;
  args: Record<string, any>;
  pay_amount?: string;
}

export async function batchContract(
  env: string,
  name: string,
  calls: BatchItem[],
): Promise<{ ok: boolean; calls?: string[]; error?: string }> {
  return request(`/api/contracts/${env}/${name}/batch`, {
    method: "POST",
    body: JSON.stringify({ calls }),
  });
}

// ──────────────────────────────────────────────
// 部署 / 升级
// ──────────────────────────────────────────────

export interface DeployRequest {
  env: string;
  name?: string;
  dir?: string;
  code?: string;
  build?: boolean;
  network?: number;
}

export async function deployContract(req: DeployRequest): Promise<DeployResponse> {
  return request("/api/deploy", { method: "POST", body: JSON.stringify(req) });
}

export async function deployFull(req: DeployRequest): Promise<DeployResponse> {
  return request("/api/deploy-full", { method: "POST", body: JSON.stringify(req) });
}

export async function upgradeContract(req: DeployRequest): Promise<DeployResponse> {
  return request("/api/upgrade", { method: "POST", body: JSON.stringify(req) });
}

// ──────────────────────────────────────────────
// ERC20 代币管理（token 合约）
// ──────────────────────────────────────────────

export interface Erc20Item {
  F0: string; // 代币地址 (H160)
  F1: boolean; // active
  F2: string; // rate
  F3: string; // unit
}

export async function fetchErc20List(env: string): Promise<Erc20Item[]> {
  const res = await callContract(env, "token", "get_erc20_list", "query", {});
  if (res.error) throw new Error(res.error);
  return (res.result as Erc20Item[]) || [];
}

export async function fetchErc20Count(env: string): Promise<number> {
  const res = await callContract(env, "token", "get_erc20_count", "query", {});
  if (res.error) throw new Error(res.error);
  return Number(res.result) || 0;
}

export interface Erc20ConfigInput {
  token: string; // 代币合约地址
  active: boolean;
  rate: string; // 汇率：1 个代币兑换的积分数量
  unit: string; // 计价单位（小数精度对应值，通常 10^18）
}

// 注册新代币 / 更新已有代币的汇率与状态（对应 set_erc20_token）
export async function saveErc20Token(env: string, cfg: Erc20ConfigInput): Promise<CallResponse> {
  return callContract(env, "token", "set_erc20_token", "exec", {
    token: cfg.token,
    active: cfg.active,
    rate: cfg.rate,
    unit: cfg.unit,
  });
}

// ──────────────────────────────────────────────
// Native 代币管理（token 合约）
// ──────────────────────────────────────────────

export interface NativeInfo {
  active: boolean;
  rate: string;
  unit: string;
}

export async function fetchNativeInfo(env: string): Promise<NativeInfo> {
  const [a, r, u] = await Promise.all([
    callContract(env, "token", "get_native_active", "query", {}),
    callContract(env, "token", "get_rate", "query", {}),
    callContract(env, "token", "get_token_unit", "query", {}),
  ]);
  const err = a.error || r.error || u.error;
  if (err) throw new Error(err);
  return { active: !!a.result, rate: String(r.result), unit: String(u.result) };
}

// 调整 native 汇率（set_rate + set_token_unit 批量提交）
export async function saveNativeRate(env: string, rate: string, unit: string) {
  return batchContract(env, "token", [
    { method: "set_rate", args: { new_rate: rate } },
    { method: "set_token_unit", args: { unit } },
  ]);
}

// 启用 / 停用 native 充值
export async function saveNativeActive(env: string, active: boolean): Promise<CallResponse> {
  return callContract(env, "token", "set_native_active", "exec", { active });
}
