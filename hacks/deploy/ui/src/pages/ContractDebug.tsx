import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, AlertTriangle, Play, Eye, History } from "lucide-react";
import { useEnv } from "../EnvContext";
import {
    fetchContractMethods,
    callContract,
    batchContract,
    type ContractMethods,
    type MethodSpec,
    type ParamSpec,
    type CallResponse,
    type BatchItem,
} from "../api";

const inputCls =
    "w-full border border-(--border-subtle) bg-(--bg-surface) px-3 py-2 text-sm font-mono focus:outline-none focus:border-(--text-secondary)";
const labelCls = "block mb-1 text-[11px] font-bold uppercase text-(--text-muted)";

interface HistoryItem {
    contract: string;
    method: string;
    kind: string;
    args: Record<string, any>;
    time: string;
    response: CallResponse;
}

interface BatchItemExt extends BatchItem {
    id: number;
}

// ──────────────────────────────────────────────
// 参数输入组件
// ──────────────────────────────────────────────

function ParamInput({
    param,
    value,
    onChange,
}: {
    param: ParamSpec;
    value: string;
    onChange: (v: string) => void;
}) {
    const t = param.type;
    const isStruct = ["Ip", "RunPrice", "AssetInfo"].includes(t);
    const isVec = t.endsWith("[]");

    if (t === "bool") {
        return (
            <select className={inputCls} value={value} onChange={(e) => onChange(e.target.value)}>
                <option value="false">false</option>
                <option value="true">true</option>
            </select>
        );
    }
    if (isStruct || isVec) {
        return (
            <>
                <textarea
                    className={`${inputCls} min-h-20 font-mono text-xs`}
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    placeholder={
                        isStruct
                            ? `JSON 对象，如 ${t === "Ip" ? '{"ipv4": 3232263935, "domain": "example.com"}' : t === "RunPrice" ? '{"cpu_per": "1000", "memory_per": "1000", "disk_per": "0", "gpu_per": "0"}' : '{"native": "0x00"}'}`
                            : `JSON 数组，如 [0, 1, 2]`
                    }
                />
                <div className="text-[10px] text-(--text-muted) mt-1">
                    {isStruct ? "JSON 对象格式" : "JSON 数组格式"}
                </div>
            </>
        );
    }
    if (t === "u8" || t === "u32" || t === "u64") {
        return (
            <input
                className={inputCls}
                type="number"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={`0`}
            />
        );
    }
    return (
        <input
            className={inputCls}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={
                t === "H160"
                    ? "0x... (20 字节)"
                    : t === "Account"
                      ? "0x... (32 字节公钥)"
                      : t === "U128" || t === "U256"
                        ? "十进制或 0x 十六进制"
                        : t === "bytes"
                          ? "0x... (十六进制)"
                          : t.startsWith("Option<")
                            ? "留空 = None"
                            : ""
            }
        />
    );
}

// 解析参数输入为 API 值
function parseArgValue(param: ParamSpec, raw: string): any {
    const t = param.type;
    const v = raw.trim();
    if (t === "bool") return v === "true";
    if (["u8", "u32", "u64"].includes(t)) return v === "" ? "0" : v;
    if (["H160", "Account", "U128", "U256", "bytes"].includes(t)) return v;
    if (t.startsWith("Option<")) return v;
    if (t === "string") return v;
    if (t.endsWith("[]")) {
        if (v === "") return [];
        try {
            const arr = JSON.parse(v);
            return Array.isArray(arr) ? arr : v.split(",").map((s) => s.trim());
        } catch {
            return v.split(",").map((s) => s.trim()).filter(Boolean);
        }
    }
    if (["Ip", "RunPrice", "AssetInfo"].includes(t)) {
        if (v === "") return {};
        try {
            return JSON.parse(v);
        } catch {
            return v;
        }
    }
    return v;
}

// ──────────────────────────────────────────────
// 主页面
// ──────────────────────────────────────────────

export function ContractDebug() {
    const { env } = useEnv();
    const [contract, setContract] = useState("token");
    const [methods, setMethods] = useState<ContractMethods | null>(null);
    const [selected, setSelected] = useState<MethodSpec | null>(null);
    const [args, setArgs] = useState<Record<string, string>>({});
    const [payAmount, setPayAmount] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [response, setResponse] = useState<CallResponse | null>(null);
    const [history, setHistory] = useState<HistoryItem[]>([]);
    const [filter, setFilter] = useState<"all" | "query" | "exec">("all");
    const [batchItems, setBatchItems] = useState<BatchItemExt[]>([]);
    const [batchMsg, setBatchMsg] = useState<string | null>(null);

    useEffect(() => {
        setSelected(null);
        setResponse(null);
        setHistory([]);
        setBatchItems([]);
    }, [env]);

    const loadMethods = useCallback(async () => {
        if (!env || !contract) return;
        setLoading(true);
        setError(null);
        setSelected(null);
        setResponse(null);
        try {
            setMethods(await fetchContractMethods(env, contract));
        } catch (err) {
            setMethods(null);
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    }, [env, contract]);

    useEffect(() => {
        loadMethods();
    }, [loadMethods]);

    const visibleMethods = useMemo(() => {
        if (!methods) return [];
        return methods.methods.filter((m) => filter === "all" || m.kind === filter);
    }, [methods, filter]);

    const selectMethod = (m: MethodSpec) => {
        setSelected(m);
        setResponse(null);
        setArgs({});
        setPayAmount("");
    };

    const execute = async (kind: "exec" | "query") => {
        if (!selected || !env || !contract) return;
        setLoading(true);
        setError(null);
        try {
            const argValues: Record<string, any> = {};
            for (const p of selected.params || []) {
                argValues[p.name] = parseArgValue(p, args[p.name] ?? "");
            }
            const resp = await callContract(
                env,
                contract,
                selected.name,
                kind,
                argValues,
                kind === "exec" ? payAmount : "",
            );
            setResponse(resp);
            setHistory((h) =>
                [
                    {
                        contract,
                        method: selected.name,
                        kind,
                        args: argValues,
                        time: new Date().toLocaleTimeString(),
                        response: resp,
                    },
                    ...h,
                ].slice(0, 50),
            );
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    // 把当前表单加入批量队列
    const addToBatch = () => {
        if (!selected || selected.kind !== "exec") return;
        const argValues: Record<string, any> = {};
        for (const p of selected.params || []) {
            argValues[p.name] = parseArgValue(p, args[p.name] ?? "");
        }
        setBatchItems((items) => [
            ...items,
            {
                id: Date.now(),
                method: selected.name,
                args: argValues,
                pay_amount: payAmount || undefined,
            },
        ]);
        setBatchMsg(null);
    };

    // 提交批量交易（batch_all，一次签名）
    const submitBatch = async () => {
        if (batchItems.length === 0 || !env || !contract) return;
        setLoading(true);
        setError(null);
        setBatchMsg(null);
        try {
            const res = await batchContract(
                env,
                contract,
                batchItems.map(({ id, ...item }) => item),
            );
            if (res.error) {
                setBatchMsg(`❌ ${res.error}`);
            } else {
                setBatchMsg(`✅ 批量交易已提交: ${(res.calls || []).join(" + ")}`);
                setBatchItems([]);
            }
        } catch (err) {
            setBatchMsg(`❌ ${err instanceof Error ? err.message : String(err)}`);
        } finally {
            setLoading(false);
        }
    };

    const renderResult = (resp: CallResponse) => {
        if (resp.error) {
            return (
                <div className="border border-red-500/40 bg-red-500/5 p-3 text-sm text-red-400 font-mono whitespace-pre-wrap break-all">
                    {resp.error}
                </div>
            );
        }
        return (
            <div className="space-y-2">
                <pre className="border border-(--border-subtle) bg-black/50 p-3 text-xs font-mono text-(--text-primary) whitespace-pre-wrap break-all max-h-96 overflow-y-auto">
                    {JSON.stringify(resp.result, null, 2)}
                </pre>
                {resp.gas && (
                    <div className="text-[11px] font-mono text-(--text-muted)">
                        gas: {JSON.stringify(resp.gas)}
                    </div>
                )}
            </div>
        );
    };

    return (
        <div className="flex-1 p-4">
            <header className="mb-8">
                <h1 className="text-2xl font-bold tracking-tighter uppercase">
                    Contract Debugger
                </h1>
                <p className="mt-1 text-sm text-(--text-muted)">
                    选择环境与合约，动态生成函数调用表单（查询 / 交易）
                </p>
            </header>

            {/* 选择器 */}
            <div className="mb-4 flex flex-wrap items-end gap-4 border border-(--border-subtle) bg-(--bg-surface) p-4">
                <div className="w-40">
                    <label className={labelCls}>合约</label>
                    <select className={inputCls} value={contract} onChange={(e) => setContract(e.target.value)}>
                        <option value="token">token</option>
                        <option value="subnet">subnet</option>
                        <option value="proxy">proxy</option>
                    </select>
                </div>
                {methods?.address && (
                    <div className="flex-1 min-w-60">
                        <label className={labelCls}>合约地址</label>
                        <div className="font-mono text-xs text-(--text-secondary) break-all pt-2">
                            {methods.address}
                        </div>
                    </div>
                )}
                <button
                    onClick={loadMethods}
                    disabled={loading}
                    className="border border-(--border-subtle) px-4 py-2 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50"
                >
                    {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "REFRESH"}
                </button>
            </div>

            {error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm font-bold bg-(--bg-elevated) text-red-500">
                    <AlertTriangle className="inline h-4 w-4 mr-2" />
                    {error}
                </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                {/* 方法列表 */}
                <div className="lg:col-span-1 border border-(--border-subtle) bg-(--bg-surface">
                    <div className="px-4 py-2 border-b border-(--border-subtle) flex items-center gap-2">
                        {(
                            [
                                ["all", "全部"],
                                ["query", "查询"],
                                ["exec", "交易"],
                            ] as [typeof filter, string][]
                        ).map(([k, label]) => (
                            <button
                                key={k}
                                onClick={() => setFilter(k)}
                                className={`px-3 py-1 text-[11px] font-bold uppercase transition ${filter === k ? "bg-(--accent) text-(--accent-foreground)" : "text-(--text-muted) hover:text-white"}`}
                            >
                                {label}
                            </button>
                        ))}
                    </div>
                    <div className="max-h-[560px] overflow-y-auto">
                        {visibleMethods.length === 0 && (
                            <div className="p-6 text-center text-xs font-mono text-(--text-muted)">
                                NO METHODS
                            </div>
                        )}
                        {visibleMethods.map((m) => (
                            <button
                                key={m.name}
                                onClick={() => selectMethod(m)}
                                className={`w-full flex items-center justify-between px-4 py-2.5 text-left text-sm border-b border-(--border-divider) transition ${selected?.name === m.name ? "bg-(--bg-elevated) text-white" : "text-(--text-secondary) hover:bg-(--bg-elevated)"}`}
                            >
                                <span className="font-mono">{m.name}</span>
                                <span
                                    className={`text-[10px] font-bold uppercase px-1.5 py-0.5 border ${m.kind === "query" ? "text-(--info) border-(--border-subtle)" : "text-(--warning) border-(--border-subtle)"}`}
                                >
                                    {m.kind}
                                </span>
                            </button>
                        ))}
                    </div>
                </div>

                {/* 参数表单 + 结果 */}
                <div className="lg:col-span-2 space-y-4">
                    {selected ? (
                        <div className="border border-(--border-subtle) bg-(--bg-surface">
                            <div className="px-4 py-3 border-b border-(--border-subtle) flex items-center justify-between">
                                <span className="font-mono font-bold text-sm">
                                    {selected.name}
                                    <span className="ml-2 text-[10px] font-bold uppercase text-(--text-muted)">
                                        {selected.kind}
                                    </span>
                                </span>
                            </div>
                            <div className="p-4 space-y-4">
                                {(selected.params || []).length === 0 && (
                                    <div className="text-xs text-(--text-muted)">
                                        无参数
                                    </div>
                                )}
                                {(selected.params || []).map((p) => (
                                    <div key={p.name}>
                                        <label className={labelCls}>
                                            {p.name}
                                            <span className="ml-2 normal-case font-mono text-(--text-muted)">
                                                ({p.type})
                                            </span>
                                        </label>
                                        <ParamInput
                                            param={p}
                                            value={args[p.name] ?? ""}
                                            onChange={(v) =>
                                                setArgs((a) => ({ ...a, [p.name]: v }))
                                            }
                                        />
                                    </div>
                                ))}

                                {selected.kind === "exec" && (
                                    <div>
                                        <label className={labelCls}>
                                            Pay Amount（链上转账，planck）
                                        </label>
                                        <input
                                            className={inputCls}
                                            value={payAmount}
                                            onChange={(e) => setPayAmount(e.target.value)}
                                            placeholder="0（recharge 等函数需要填写）"
                                        />
                                    </div>
                                )}

                                <div className="flex gap-3">
                                    <button
                                        onClick={() => execute("query")}
                                        disabled={loading}
                                        className="flex-1 border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                                    >
                                        {loading ? (
                                            <Loader2 className="h-4 w-4 animate-spin" />
                                        ) : (
                                            <Eye className="h-4 w-4" />
                                        )}
                                        查询（DryRun）
                                    </button>
                                    <button
                                        onClick={() => execute("exec")}
                                        disabled={loading}
                                        className="flex-1 border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                                    >
                                        {loading ? (
                                            <Loader2 className="h-4 w-4 animate-spin" />
                                        ) : (
                                            <Play className="h-4 w-4" />
                                        )}
                                        提交交易
                                    </button>
                                    {selected.kind === "exec" && (
                                        <button
                                            onClick={addToBatch}
                                            disabled={loading}
                                            className="border border-(--border-subtle) px-4 py-2.5 text-sm font-bold text-(--purple) hover:bg-(--bg-elevated) transition disabled:opacity-50"
                                        >
                                            加入批量
                                        </button>
                                    )}
                                </div>
                                {selected.kind === "exec" && (
                                    <div className="text-[11px] text-(--text-muted)">
                                        支持把多个交易加入批量队列，通过 batch_all 一次签名提交
                                        （如 set_rate + set_token_unit 同时调整）。
                                    </div>
                                )}
                            </div>
                        </div>
                    ) : (
                        <div className="border border-(--border-subtle) bg-(--bg-surface p-8 text-center text-sm text-(--text-muted)">
                            从左侧选择要调用的函数
                        </div>
                    )}

                    {response && (
                        <div className="border border-(--border-subtle) bg-(--bg-surface)">
                            <div className="px-4 py-2 border-b border-(--border-subtle) text-[11px] font-bold uppercase text-(--text-muted)">
                                调用结果 · {response.method} · {response.kind}
                            </div>
                            <div className="p-4">{renderResult(response)}</div>
                        </div>
                    )}

                    {batchItems.length > 0 && (
                        <div className="border border-(--border-subtle) bg-(--bg-surface">
                            <div className="px-4 py-2 border-b border-(--border-subtle) flex items-center justify-between text-[11px] font-bold uppercase text-(--text-muted)">
                                <span>批量队列（{batchItems.length}）</span>
                                <button
                                    onClick={() => setBatchItems([])}
                                    className="text-(--text-secondary) hover:text-white"
                                >
                                    清空
                                </button>
                            </div>
                            <div className="divide-y divide-(--border-divider)">
                                {batchItems.map((item) => (
                                    <div
                                        key={item.id}
                                        className="px-4 py-2 flex items-center gap-3 text-xs"
                                    >
                                        <span className="font-mono font-bold">
                                            {contract}.{item.method}
                                        </span>
                                        <span className="font-mono text-(--text-secondary) truncate flex-1">
                                            {Object.entries(item.args)
                                                .map(([k, v]) => `${k}=${JSON.stringify(v)}`)
                                                .join(", ")}
                                        </span>
                                        <button
                                            onClick={() =>
                                                setBatchItems((items) =>
                                                    items.filter((i) => i.id !== item.id),
                                                )
                                            }
                                            className="text-red-400 hover:text-red-300"
                                        >
                                            移除
                                        </button>
                                    </div>
                                ))}
                            </div>
                            {batchMsg && (
                                <div className="px-4 py-2 text-xs font-mono border-t border-(--border-divider) break-all">
                                    {batchMsg}
                                </div>
                            )}
                            <div className="p-4 border-t border-(--border-subtle)">
                                <button
                                    onClick={submitBatch}
                                    disabled={loading}
                                    className="w-full border border-(--border-subtle) px-4 py-2.5 text-sm font-bold text-(--purple) hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                                >
                                    {loading && (
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                    )}
                                    提交批量交易（batch_all）
                                </button>
                            </div>
                        </div>
                    )}

                    {history.length > 0 && (
                        <div className="border border-(--border-subtle) bg-(--bg-surface">
                            <div className="px-4 py-2 border-b border-(--border-subtle) flex items-center gap-2 text-[11px] font-bold uppercase text-(--text-muted)">
                                <History className="h-3.5 w-3.5" />
                                调用历史
                            </div>
                            <div className="max-h-64 overflow-y-auto divide-y divide-(--border-divider)">
                                {history.map((h, i) => (
                                    <button
                                        key={i}
                                        onClick={() => setResponse(h.response)}
                                        className="w-full text-left px-4 py-2 hover:bg-(--bg-elevated) flex items-center gap-3 text-xs"
                                    >
                                        <span className="font-mono text-(--text-secondary)">
                                            {h.time}
                                        </span>
                                        <span className="font-mono font-bold">
                                            {h.contract}.{h.method}
                                        </span>
                                        <span
                                            className={`text-[10px] font-bold uppercase ${h.kind === "query" ? "text-(--info)" : "text-(--warning)"}`}
                                        >
                                            {h.kind}
                                        </span>
                                        {h.response.error ? (
                                            <span className="text-red-400 ml-auto font-mono truncate max-w-40">
                                                {h.response.error}
                                            </span>
                                        ) : (
                                            <span className="text-(--success) ml-auto">
                                                ✓
                                            </span>
                                        )}
                                    </button>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
