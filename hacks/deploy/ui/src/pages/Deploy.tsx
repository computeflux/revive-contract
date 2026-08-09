import { useState } from "react";
import { Loader2, AlertTriangle, CheckCircle2 } from "lucide-react";
import { useEnv } from "../EnvContext";
import {
    deployContract,
    deployFull,
    upgradeContract,
    type DeployResponse,
} from "../api";

const inputCls =
    "w-full border border-(--border-subtle) bg-(--bg-surface) px-3 py-2 text-sm font-mono focus:outline-none focus:border-(--text-secondary)";
const labelCls = "block mb-1 text-[11px] font-bold uppercase text-(--text-muted)";

type DeployTab = "single" | "full" | "upgrade";

function LogPanel({ logs }: { logs: string[] | undefined }) {
    if (!logs || logs.length === 0) return null;
    return (
        <div className="mt-4 border border-(--border-subtle) bg-black/50">
            <div className="px-4 py-2 border-b border-(--border-subtle) text-[11px] font-bold uppercase text-(--text-muted)">
                执行日志
            </div>
            <pre className="p-4 text-xs font-mono text-(--text-secondary) whitespace-pre-wrap leading-relaxed max-h-80 overflow-y-auto">
                {logs.join("\n")}
            </pre>
        </div>
    );
}

export function Deploy() {
    const { env } = useEnv();
    const [tab, setTab] = useState<DeployTab>("single");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [result, setResult] = useState<DeployResponse | null>(null);

    // 单合约部署表单
    const [sName, setSName] = useState("token");
    const [sDir, setSDir] = useState(".");
    const [sCode, setSCode] = useState("");
    const [sBuild, setSBuild] = useState(false);

    // 全量部署表单
    const [fDir, setFDir] = useState(".");

    // 升级表单
    const [uName, setUName] = useState("token");

    const run = async (fn: () => Promise<DeployResponse>) => {
        setLoading(true);
        setError(null);
        setResult(null);
        try {
            setResult(await fn());
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="flex-1 p-4 overflow-y-auto">
            <header className="mb-8">
                <h1 className="text-2xl font-bold tracking-tighter uppercase">
                    Contract Deploy
                </h1>
                <p className="mt-1 text-sm text-(--text-muted)">
                    单合约部署 / 全量部署 / 代理升级，全部通过 API 执行
                </p>
            </header>

            {error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm font-bold bg-(--bg-elevated) text-red-500">
                    <AlertTriangle className="inline h-4 w-4 mr-2" />
                    {error}
                </div>
            )}
            {result?.error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm font-bold bg-(--bg-elevated) text-red-500">
                    {result.error}
                </div>
            )}
            {result && !result.error && (
                <div className="mb-4 border border-(--border-subtle) p-3 text-sm bg-(--bg-elevated) text-(--success)">
                    <CheckCircle2 className="inline h-4 w-4 mr-2" />
                    {result.address && <>合约地址: <span className="font-mono">{result.address}</span></>}
                    {result.subnet && <>Subnet: <span className="font-mono">{result.subnet}</span></>}
                    {result.token && <>Token: <span className="font-mono">{result.token}</span></>}
                    {result.proxy && <>Proxy: <span className="font-mono">{result.proxy}</span></>}
                    {result.impl && <>Impl: <span className="font-mono">{result.impl}</span></>}
                </div>
            )}

            <div className="mb-4 flex items-center gap-2 px-2 border border-(--border-subtle)">
                {(
                    [
                        ["single", "单合约部署"],
                        ["full", "全量部署"],
                        ["upgrade", "合约升级"],
                    ] as [DeployTab, string][]
                ).map(([key, label]) => (
                    <button
                        key={key}
                        onClick={() => setTab(key)}
                        className={`px-4 py-2 text-xs font-bold uppercase transition ${tab === key ? "border-b-2 border-white text-white" : "text-(--text-muted) hover:text-(--text-secondary)"}`}
                    >
                        {label}
                    </button>
                ))}
            </div>

            {/* 单合约部署 */}
            {tab === "single" && (
                <div className="max-w-xl border border-(--border-subtle) bg-(--bg-surface)">
                    <div className="px-4 py-3 border-b border-(--border-subtle) font-bold uppercase text-sm">
                        部署单个合约
                    </div>
                    <div className="p-4 space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className={labelCls}>合约名</label>
                                <input
                                    className={inputCls}
                                    value={sName}
                                    onChange={(e) => setSName(e.target.value)}
                                    placeholder="token / subnet / proxy"
                                />
                            </div>
                            <div>
                                <label className={labelCls}>环境</label>
                                <div className={inputCls + " flex items-center text-(--info)"}>
                                    {env || "—"}
                                </div>
                            </div>
                        </div>
                        <div>
                            <label className={labelCls}>工作区目录（含 target/）</label>
                            <input
                                className={inputCls}
                                value={sDir}
                                onChange={(e) => setSDir(e.target.value)}
                                placeholder="."
                            />
                        </div>
                        <div>
                            <label className={labelCls}>
                                .polkavm 路径（默认 &lt;dir&gt;/target/&lt;name&gt;.release.polkavm）
                            </label>
                            <input
                                className={inputCls}
                                value={sCode}
                                onChange={(e) => setSCode(e.target.value)}
                                placeholder="留空自动推断"
                            />
                        </div>
                        <label className="flex items-center gap-2 text-sm text-(--text-secondary)">
                            <input
                                type="checkbox"
                                checked={sBuild}
                                onChange={(e) => setSBuild(e.target.checked)}
                            />
                            先执行 cargo wrevive build 编译
                        </label>
                        <button
                            onClick={() =>
                                run(() =>
                                    deployContract({
                                        env,
                                        name: sName,
                                        dir: sDir,
                                        code: sCode || undefined,
                                        build: sBuild,
                                    }),
                                )
                            }
                            disabled={loading || !env || !sName}
                            className="w-full border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                            部署 {sName}
                        </button>
                    </div>
                </div>
            )}

            {/* 全量部署 */}
            {tab === "full" && (
                <div className="max-w-xl border border-(--border-subtle) bg-(--bg-surface)">
                    <div className="px-4 py-3 border-b border-(--border-subtle) font-bold uppercase text-sm">
                        全量部署（subnet + token + proxy + 创世初始化）
                    </div>
                    <div className="p-4 space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className={labelCls}>环境</label>
                                <div className={inputCls + " flex items-center text-(--info)"}>
                                    {env || "—"}
                                </div>
                            </div>
                            <div>
                                <label className={labelCls}>工作区目录</label>
                                <input
                                    className={inputCls}
                                    value={fDir}
                                    onChange={(e) => setFDir(e.target.value)}
                                    placeholder="."
                                />
                            </div>
                        </div>
                        <div className="text-xs text-(--text-muted) leading-relaxed">
                            按顺序执行：部署 Subnet 实现 + 代理 + init →
                            部署 Token 实现 + 代理 + init + set_subnet →
                            按 configs/&lt;env&gt;.json 的 genesis 初始化节点。
                            <br />
                            需要 <span className="font-mono">target/subnet.release.polkavm</span>、
                            <span className="font-mono">token.release.polkavm</span>、
                            <span className="font-mono">proxy.release.polkavm</span> 三个文件。
                        </div>
                        <button
                            onClick={() =>
                                run(() =>
                                    deployFull({ env, dir: fDir }),
                                )
                            }
                            disabled={loading || !env}
                            className="w-full border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                            开始全量部署
                        </button>
                    </div>
                </div>
            )}

            {/* 升级 */}
            {tab === "upgrade" && (
                <div className="max-w-xl border border-(--border-subtle) bg-(--bg-surface)">
                    <div className="px-4 py-3 border-b border-(--border-subtle) font-bold uppercase text-sm">
                        热升级（部署新实现 + 代理指向切换）
                    </div>
                    <div className="p-4 space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className={labelCls}>环境</label>
                                <div className={inputCls + " flex items-center text-(--info)"}>
                                    {env || "—"}
                                </div>
                            </div>
                            <div>
                                <label className={labelCls}>合约</label>
                                <select
                                    className={inputCls}
                                    value={uName}
                                    onChange={(e) => setUName(e.target.value)}
                                >
                                    <option value="token">token</option>
                                    <option value="subnet">subnet</option>
                                </select>
                            </div>
                        </div>
                        <div className="text-xs text-(--text-muted)">
                            读取 configs/&lt;env&gt;.json 中 contracts 的代理地址，
                            部署新的实现合约后调用代理 upgrade()。状态保留。
                        </div>
                        <button
                            onClick={() =>
                                run(() =>
                                    upgradeContract({ env, name: uName }),
                                )
                            }
                            disabled={loading || !env}
                            className="w-full border border-(--border-subtle) px-4 py-2.5 text-sm font-bold hover:bg-(--bg-elevated) transition disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                            升级 {uName}
                        </button>
                    </div>
                </div>
            )}

            <LogPanel logs={result?.logs} />
        </div>
    );
}
