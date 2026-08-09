import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Loader2, RefreshCw, AlertTriangle, ArrowRight } from "lucide-react";
import { useEnv } from "../EnvContext";
import { fetchEnvs, fetchEnvDetail, type EnvPublic, type EnvDetail } from "../api";

export function Dashboard() {
    const { setEnv } = useEnv();
    const [envs, setEnvs] = useState<EnvPublic[]>([]);
    const [details, setDetails] = useState<Record<string, EnvDetail>>({});
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const load = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const { envs } = await fetchEnvs();
            setEnvs(envs);
            const detailMap: Record<string, EnvDetail> = {};
            await Promise.all(
                envs.map(async (e) => {
                    try {
                        detailMap[e.name] = await fetchEnvDetail(e.name);
                    } catch {
                        // 链不可达时跳过详情
                    }
                }),
            );
            setDetails(detailMap);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        load();
    }, [load]);

    return (
        <div className="flex-1 p-4 overflow-y-auto">
            <header className="mb-8 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold tracking-tighter uppercase">
                        WeTEE Contract Console
                    </h1>
                    <p className="mt-1 text-sm text-(--text-muted)">
                        合约部署、调试与账户总览
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

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
                {envs.map((e) => {
                    const detail = details[e.name];
                    const account = detail?.account;
                    return (
                        <div
                            key={e.name}
                            className="border border-(--border-subtle) bg-(--bg-surface)"
                        >
                            <div className="px-4 py-3 border-b border-(--border-subtle) flex items-center justify-between">
                                <span className="font-bold uppercase text-sm">
                                    {e.name}
                                </span>
                                {e.has_suri ? (
                                    <span className="text-[10px] text-(--success) border border-(--border-subtle) px-2 py-0.5">
                                        SURI 已配置
                                    </span>
                                ) : (
                                    <span className="text-[10px] text-(--warning) border border-(--border-subtle) px-2 py-0.5">
                                        无 SURI
                                    </span>
                                )}
                            </div>
                            <div className="p-4 space-y-2 text-xs">
                                <div>
                                    <span className="text-(--text-muted) uppercase">
                                        RPC{" "}
                                    </span>
                                    <span className="font-mono text-(--text-secondary) break-all">
                                        {e.url}
                                    </span>
                                </div>
                                {account ? (
                                    <>
                                        <div>
                                            <span className="text-(--text-muted) uppercase">
                                                SS58{" "}
                                            </span>
                                            <span className="font-mono text-(--text-secondary) break-all">
                                                {account.ss58}
                                            </span>
                                        </div>
                                        <div>
                                            <span className="text-(--text-muted) uppercase">
                                                H160{" "}
                                            </span>
                                            <span className="font-mono text-(--text-secondary) break-all">
                                                {account.h160}
                                            </span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-(--text-muted) uppercase">
                                                余额
                                            </span>
                                            <span className="font-mono text-(--text-primary)">
                                                {account.free_balance}
                                            </span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-(--text-muted) uppercase">
                                                Revive
                                            </span>
                                            {account.mapped ? (
                                                <span className="text-(--success)">已映射</span>
                                            ) : (
                                                <span className="text-(--warning)">未映射</span>
                                            )}
                                        </div>
                                    </>
                                ) : (
                                    <div className="text-(--text-muted)">
                                        链不可达或无账户信息
                                    </div>
                                )}
                                {Object.entries(e.contracts || {}).filter(
                                    ([, v]) => v,
                                ).length > 0 && (
                                    <div className="pt-2 border-t border-(--border-divider)">
                                        {Object.entries(e.contracts || {})
                                            .filter(([, v]) => v)
                                            .map(([k, v]) => (
                                                <div
                                                    key={k}
                                                    className="flex justify-between"
                                                >
                                                    <span className="text-(--text-muted) uppercase">
                                                        {k}
                                                    </span>
                                                    <span className="font-mono text-(--text-secondary) break-all max-w-[220px] text-right">
                                                        {v}
                                                    </span>
                                                </div>
                                            ))}
                                    </div>
                                )}
                            </div>
                            <div className="px-4 py-3 border-t border-(--border-subtle) flex gap-3">
                                <Link
                                    to="/debug"
                                    className="text-xs font-bold underline text-(--text-secondary) hover:text-white"
                                >
                                    调试合约
                                </Link>
                                <Link
                                    to="/deploy"
                                    className="text-xs font-bold underline text-(--text-secondary) hover:text-white"
                                >
                                    部署
                                </Link>
                                <button
                                    onClick={() => setEnv(e.name)}
                                    className="ml-auto text-xs font-bold text-(--info) hover:text-white flex items-center gap-1"
                                >
                                    切换到此环境
                                    <ArrowRight className="h-3 w-3" />
                                </button>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
