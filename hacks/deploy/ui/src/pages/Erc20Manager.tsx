import { useCallback, useEffect, useState } from "react";
import {
    Loader2,
    RefreshCw,
    AlertTriangle,
    Pencil,
    Save,
    X,
    Plus,
    Coins,
} from "lucide-react";
import { useEnv } from "../EnvContext";
import {
    fetchErc20List,
    fetchErc20Count,
    fetchNativeInfo,
    saveErc20Token,
    saveNativeRate,
    saveNativeActive,
    callContract,
    type Erc20Item,
    type CallResponse,
} from "../api";

const inputCls =
    "w-full border border-(--border-subtle) bg-(--bg-surface) px-3 py-2 text-sm font-mono focus:outline-none focus:border-(--text-secondary)";
const labelCls = "block mb-1 text-[11px] font-bold uppercase text-(--text-muted)";
const btnCls =
    "border border-(--border-subtle) px-4 py-2 text-xs font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center gap-1.5";

// 大数格式化：1234567890123456789 → 1.23e18
function fmtBig(s: string): string {
    if (!s || s === "0") return s || "0";
    const n = Number(s);
    if (!Number.isFinite(n) || n === 0) return s;
    if (n >= 1e15) return `${(n / 1e18).toFixed(3).replace(/\.?0+$/, "")}e18`;
    if (n >= 1e6) return `${(n / 1e6).toFixed(2).replace(/\.?0+$/, "")}e6`;
    return n.toLocaleString();
}

// 统一列表项：native 与 ERC20
interface TokenItem {
    token: string;
    isNative: boolean;
    active: boolean;
    rate: string;
    unit: string;
}

interface EditState {
    token: string;
    isNative: boolean;
    active: boolean;
    rate: string;
    unit: string;
}

const emptyEdit: EditState = {
    token: "",
    isNative: false,
    active: true,
    rate: "",
    unit: "",
};

export function Erc20Manager() {
    const { env } = useEnv();
    const [list, setList] = useState<TokenItem[]>([]);
    const [erc20Count, setErc20Count] = useState<number | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [result, setResult] = useState<CallResponse | null>(null);
    // 行内编辑
    const [editing, setEditing] = useState<EditState | null>(null);
    // 注册新代币
    const [showRegister, setShowRegister] = useState(false);
    const [register, setRegister] = useState<EditState>(emptyEdit);

    useEffect(() => {
        setList([]);
        setErc20Count(null);
        setEditing(null);
        setResult(null);
    }, [env]);

    const load = useCallback(async () => {
        if (!env) return;
        setLoading(true);
        setError(null);
        try {
            const [erc20s, cnt, native] = await Promise.all([
                fetchErc20List(env),
                fetchErc20Count(env),
                fetchNativeInfo(env),
            ]);
            const items: TokenItem[] = [
                {
                    token: "NATIVE",
                    isNative: true,
                    active: native.active,
                    rate: native.rate,
                    unit: native.unit,
                },
                ...erc20s.map((e: Erc20Item) => ({
                    token: e.F0,
                    isNative: false,
                    active: e.F1,
                    rate: e.F2,
                    unit: e.F3,
                })),
            ];
            setList(items);
            setErc20Count(cnt);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    }, [env]);

    useEffect(() => {
        load();
    }, [load]);

    // 保存（native: set_rate+set_token_unit 批量；erc20: set_erc20_token）
    const save = async (cfg: EditState, isNew: boolean) => {
        setLoading(true);
        setError(null);
        setResult(null);
        try {
            const resp = cfg.isNative
                ? await saveNativeRate(env, cfg.rate, cfg.unit)
                : await saveErc20Token(env, {
                      token: cfg.token,
                      active: cfg.active,
                      rate: cfg.rate,
                      unit: cfg.unit,
                  });
            const err = (resp as any).error;
            setResult(resp as CallResponse);
            if (err) {
                setError(err);
            } else {
                if (isNew) {
                    setRegister(emptyEdit);
                    setShowRegister(false);
                }
                setEditing(null);
                await load();
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    // 启用 / 停用（native: set_native_active；erc20: set_erc20_token active）
    const setActive = async (item: TokenItem, active: boolean) => {
        setLoading(true);
        setError(null);
        setResult(null);
        try {
            const resp = item.isNative
                ? await saveNativeActive(env, active)
                : await saveErc20Token(env, {
                      token: item.token,
                      active,
                      rate: item.rate,
                      unit: item.unit,
                  });
            setResult(resp);
            if (resp.error) {
                setError(resp.error);
            } else {
                await load();
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    const startEdit = (item: TokenItem) => {
        setEditing({
            token: item.token,
            isNative: item.isNative,
            active: item.active,
            rate: item.rate,
            unit: item.unit,
        });
    };

    return (
        <div className="flex-1 p-4 overflow-y-auto">
            <header className="mb-8 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold tracking-tighter uppercase">
                        Token Management
                    </h1>
                    <p className="mt-1 text-sm text-(--text-muted)">
                        Native 与 ERC20 代币：注册、调整汇率（rate）与计价单位（unit）、启停
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <button onClick={load} disabled={loading} className={btnCls}>
                        {loading ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                            <RefreshCw className="h-3.5 w-3.5" />
                        )}
                        REFRESH
                    </button>
                </div>
            </header>

            {error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm font-bold bg-(--bg-elevated) text-red-500">
                    <AlertTriangle className="inline h-4 w-4 mr-2" />
                    {error}
                </div>
            )}
            {result && !(result as any).error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm bg-(--bg-elevated) text-(--success)">
                    ✅ 交易已提交
                </div>
            )}

            {/* 统计 + 注册入口 */}
            <div className="mb-4 flex items-center gap-4">
                <div className="border border-(--border-subtle) bg-(--bg-surface h-9 px-4 flex items-center gap-2">
                    <span className="text-[11px] font-bold uppercase text-(--text-muted)">
                        代币总数
                    </span>
                    <span className="text-base font-bold font-mono leading-none">
                        {list.length > 0 ? list.length : "—"}
                    </span>
                </div>
                <div className="border border-(--border-subtle) bg-(--bg-surface h-9 px-4 flex items-center gap-2">
                    <span className="text-[11px] font-bold uppercase text-(--text-muted)">
                        已注册 ERC20
                    </span>
                    <span className="text-base font-bold font-mono leading-none">
                        {erc20Count ?? "—"}
                    </span>
                </div>
                <button
                    onClick={() => setShowRegister((v) => !v)}
                    className={btnCls + " h-9"}
                >
                    <Plus className="h-3.5 w-3.5" />
                    {showRegister ? "收起" : "注册新代币"}
                </button>
            </div>

            {/* 注册新代币 */}
            {showRegister && (
                <div className="mb-6 border border-(--border-subtle) bg-(--bg-surface max-w-2xl">
                    <div className="px-4 py-3 border-b border-(--border-subtle) flex items-center gap-2 font-bold uppercase text-sm">
                        <Coins className="h-4 w-4" />
                        注册新 ERC20 代币
                    </div>
                    <div className="p-4 space-y-4">
                        <div>
                            <label className={labelCls}>代币合约地址</label>
                            <input
                                className={inputCls}
                                value={register.token}
                                onChange={(e) =>
                                    setRegister({ ...register, token: e.target.value })
                                }
                                placeholder="0x... (ERC20 合约地址)"
                            />
                        </div>
                        <div className="grid grid-cols-3 gap-4">
                            <div>
                                <label className={labelCls}>汇率 rate</label>
                                <input
                                    className={inputCls}
                                    value={register.rate}
                                    onChange={(e) =>
                                        setRegister({ ...register, rate: e.target.value })
                                    }
                                    placeholder="10000"
                                />
                            </div>
                            <div>
                                <label className={labelCls}>计价单位 unit</label>
                                <input
                                    className={inputCls}
                                    value={register.unit}
                                    onChange={(e) =>
                                        setRegister({ ...register, unit: e.target.value })
                                    }
                                    placeholder="1000000000000000000"
                                />
                            </div>
                            <div>
                                <label className={labelCls}>启用</label>
                                <select
                                    className={inputCls}
                                    value={String(register.active)}
                                    onChange={(e) =>
                                        setRegister({
                                            ...register,
                                            active: e.target.value === "true",
                                        })
                                    }
                                >
                                    <option value="true">启用</option>
                                    <option value="false">停用</option>
                                </select>
                            </div>
                        </div>
                        <div className="text-[11px] text-(--text-muted) leading-relaxed">
                            rate = 1 个代币兑换的积分数量；unit = 计价精度
                            （常用 10^18）。对已注册地址再次提交即为更新。
                        </div>
                        <button
                            onClick={() => save(register, true)}
                            disabled={loading || !register.token || !register.rate || !register.unit}
                            className="w-full border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                            注册 / 更新代币
                        </button>
                    </div>
                </div>
            )}

            {/* 代币列表 */}
            <div className="border border-(--border-subtle) mb-8">
                <table className="w-full text-left text-sm">
                    <thead className="border-b border-(--border-subtle) text-(--text-muted)">
                        <tr>
                            <th className="px-4 py-3 font-bold uppercase">代币</th>
                            <th className="px-4 py-3 font-bold uppercase">状态</th>
                            <th className="px-4 py-3 font-bold uppercase">汇率 rate</th>
                            <th className="px-4 py-3 font-bold uppercase">计价单位 unit</th>
                            <th className="px-4 py-3 font-bold uppercase">余额</th>
                            <th className="px-4 py-3 text-right font-bold uppercase">
                                操作
                            </th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-(--border-divider)">
                        {list.length === 0 && !loading ? (
                            <tr>
                                <td colSpan={6} className="px-4 py-8 text-center text-(--text-muted) font-mono text-xs">
                                    NO TOKENS
                                </td>
                            </tr>
                        ) : (
                            list.map((item) => (
                                <Row
                                    key={item.token}
                                    item={item}
                                    env={env}
                                    editing={editing?.token === item.token ? editing : null}
                                    loading={loading}
                                    onEdit={() => startEdit(item)}
                                    onCancel={() => setEditing(null)}
                                    onSave={(cfg) => save(cfg, false)}
                                    onChange={(cfg) => setEditing(cfg)}
                                    onSetActive={(active) => setActive(item, active)}
                                />
                            ))
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

// ──────────────────────────────────────────────
// 列表行（含行内编辑 / 启停）
// ──────────────────────────────────────────────

function Row({
    item,
    env,
    editing,
    loading,
    onEdit,
    onCancel,
    onSave,
    onChange,
    onSetActive,
}: {
    item: TokenItem;
    env: string;
    editing: EditState | null;
    loading: boolean;
    onEdit: () => void;
    onCancel: () => void;
    onSave: (cfg: EditState) => void;
    onChange: (cfg: EditState) => void;
    onSetActive: (active: boolean) => void;
}) {
    const [balance, setBalance] = useState<string | null>(null);
    const [askDisable, setAskDisable] = useState(false);

    useEffect(() => {
        setBalance(null);
        if (item.isNative) {
            setBalance("—");
            return;
        }
        callContract(env, "token", "get_erc20_balance", "query", {
            token: item.token,
        })
            .then((res) => setBalance(res.error ? "—" : String(res.result)))
            .catch(() => setBalance("—"));
    }, [env, item.token, item.isNative]);

    return (
        <tr className="hover:bg-(--bg-elevated)">
            <td className="px-4 py-3 font-mono text-(--text-secondary)">
                {item.isNative ? (
                    <span className="inline-flex items-center gap-2">
                        <span className="font-bold text-(--text-primary)">NATIVE</span>
                        <span className="text-[10px] font-bold uppercase border border-(--border-subtle) px-1.5 py-0.5 text-(--purple)">
                            原生代币
                        </span>
                    </span>
                ) : (
                    <span className="break-all max-w-56 inline-block align-middle">
                        {item.token}
                    </span>
                )}
            </td>
            <td className="px-4 py-3">
                {editing ? (
                    <select
                        className="border border-(--border-subtle) bg-(--bg-surface) px-2 py-1 text-xs font-mono"
                        value={String(editing.active)}
                        onChange={(e) =>
                            onChange({ ...editing, active: e.target.value === "true" })
                        }
                    >
                        <option value="true">启用</option>
                        <option value="false">停用</option>
                    </select>
                ) : item.active ? (
                    <span className="text-[10px] font-bold uppercase border border-(--border-subtle) px-2 py-0.5 text-(--success)">
                        启用
                    </span>
                ) : (
                    <span className="text-[10px] font-bold uppercase border border-(--border-subtle) px-2 py-0.5 text-(--danger)">
                        停用
                    </span>
                )}
            </td>
            <td className="px-4 py-3 font-mono">
                {editing ? (
                    <input
                        className="border border-(--border-subtle) bg-(--bg-surface) px-2 py-1 text-xs font-mono w-32"
                        value={editing.rate}
                        onChange={(e) => onChange({ ...editing, rate: e.target.value })}
                    />
                ) : (
                    <span title={item.rate}>{fmtBig(item.rate)}</span>
                )}
            </td>
            <td className="px-4 py-3 font-mono">
                {editing ? (
                    <input
                        className="border border-(--border-subtle) bg-(--bg-surface) px-2 py-1 text-xs font-mono w-48"
                        value={editing.unit}
                        onChange={(e) => onChange({ ...editing, unit: e.target.value })}
                    />
                ) : (
                    <span title={item.unit}>{fmtBig(item.unit)}</span>
                )}
            </td>
            <td className="px-4 py-3 font-mono text-(--text-secondary)">
                {balance ?? "…"}
            </td>
            <td className="px-4 py-3 text-right">
                {editing ? (
                    <span className="inline-flex items-center gap-2">
                        <button
                            onClick={() => onSave(editing)}
                            disabled={loading}
                            className="text-(--success) underline font-bold disabled:opacity-50 inline-flex items-center gap-1"
                        >
                            <Save className="h-3.5 w-3.5" />
                            {loading ? "PROC..." : "保存"}
                        </button>
                        <button
                            onClick={onCancel}
                            disabled={loading}
                            className="text-(--text-secondary) underline font-bold disabled:opacity-50 inline-flex items-center gap-1"
                        >
                            <X className="h-3.5 w-3.5" />
                            取消
                        </button>
                    </span>
                ) : (
                    <span className="inline-flex items-center gap-3">
                        <button
                            onClick={onEdit}
                            className="text-white underline font-bold inline-flex items-center gap-1"
                        >
                            <Pencil className="h-3.5 w-3.5" />
                            调整汇率
                        </button>
                        {item.active ? (
                            askDisable ? (
                                <span className="inline-flex items-center gap-2">
                                    <button
                                        onClick={() => onSetActive(false)}
                                        disabled={loading}
                                        className="text-red-400 underline font-bold disabled:opacity-50"
                                    >
                                        {loading ? "PROC..." : "确认停用?"}
                                    </button>
                                    <button
                                        onClick={() => setAskDisable(false)}
                                        disabled={loading}
                                        className="text-(--text-secondary) underline font-bold disabled:opacity-50"
                                    >
                                        取消
                                    </button>
                                </span>
                            ) : (
                                <button
                                    onClick={() => setAskDisable(true)}
                                    className="text-red-400 underline font-bold"
                                >
                                    停用
                                </button>
                            )
                        ) : (
                            <button
                                onClick={() => onSetActive(true)}
                                disabled={loading}
                                className="text-(--success) underline font-bold disabled:opacity-50"
                            >
                                {loading ? "PROC..." : "启用"}
                            </button>
                        )}
                    </span>
                )}
            </td>
        </tr>
    );
}
