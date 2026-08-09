import { useCallback, useEffect, useState } from "react";
import { Loader2, RefreshCw, AlertTriangle } from "lucide-react";
import { useEnv } from "../EnvContext";
import { fetchEnvDetail, mapAccount, type AccountInfo } from "../api";

export function Account() {
    const { env } = useEnv();
    const [account, setAccount] = useState<AccountInfo | null>(null);
    const [loading, setLoading] = useState(false);
    const [mapping, setMapping] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const load = useCallback(async () => {
        if (!env) return;
        setLoading(true);
        setError(null);
        try {
            const detail = await fetchEnvDetail(env);
            setAccount(detail.account);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    }, [env]);

    useEffect(() => {
        load();
    }, [load]);

    const handleMap = async () => {
        setMapping(true);
        setError(null);
        try {
            const res = await mapAccount(env);
            setAccount(res.account);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setMapping(false);
        }
    };

    return (
        <div className="flex-1 p-4 overflow-y-auto">
            <header className="mb-8 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold tracking-tighter uppercase">
                        Account
                    </h1>
                    <p className="mt-1 text-sm text-(--text-muted)">
                        查看签名账户信息，执行 Revive Map Account
                    </p>
                </div>
                <button
                    onClick={load}
                    disabled={loading}
                    className="border border-(--border-subtle) px-4 py-2 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center gap-2"
                >
                    {loading ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                        <RefreshCw className="h-4 w-4" />
                    )}{" "}
                    REFRESH
                </button>
            </header>

            {error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm font-bold bg-(--bg-elevated) text-red-500">
                    <AlertTriangle className="inline h-4 w-4 mr-2" />
                    {error}
                </div>
            )}

            <div className="max-w-xl border border-(--border-subtle) bg-(--bg-surface">
                <div className="px-4 py-3 border-b border-(--border-subtle) flex items-center justify-between">
                    <span className="font-bold uppercase text-sm">账户信息</span>
                    <span className="text-[10px] font-bold uppercase border border-(--border-subtle) px-2 py-0.5 text-(--info)">
                        {env}
                    </span>
                </div>

                {account ? (
                    <div className="p-4 space-y-3 text-sm">
                        <div>
                            <div className="text-[11px] font-bold uppercase text-(--text-muted) mb-1">
                                SS58
                            </div>
                            <div className="font-mono text-(--text-secondary) break-all">
                                {account.ss58}
                            </div>
                        </div>
                        <div>
                            <div className="text-[11px] font-bold uppercase text-(--text-muted) mb-1">
                                H160（合约调用身份）
                            </div>
                            <div className="font-mono text-(--text-secondary) break-all">
                                {account.h160}
                            </div>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-[11px] font-bold uppercase text-(--text-muted)">
                                链上余额（planck）
                            </span>
                            <span className="font-mono">{account.free_balance}</span>
                        </div>
                        <div className="flex justify-between items-center border-t border-(--border-divider) pt-3">
                            <div>
                                <div className="text-[11px] font-bold uppercase text-(--text-muted)">
                                    Revive 映射
                                </div>
                                <div
                                    className={`mt-1 text-xs font-bold ${account.mapped ? "text-(--success)" : "text-(--warning)"}`}
                                >
                                    {account.mapped
                                        ? "已映射，可直接部署/调用合约"
                                        : "未映射，需要先执行 Map Account"}
                                </div>
                            </div>
                            {!account.mapped && (
                                <button
                                    onClick={handleMap}
                                    disabled={mapping}
                                    className="border border-(--border-subtle) px-4 py-2 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center gap-2"
                                >
                                    {mapping && (
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                    )}
                                    MAP ACCOUNT
                                </button>
                            )}
                        </div>
                    </div>
                ) : (
                    <div className="p-8 text-center text-xs font-mono text-(--text-muted)">
                        {loading ? "LOADING..." : "NO ACCOUNT INFO"}
                    </div>
                )}
            </div>
        </div>
    );
}
