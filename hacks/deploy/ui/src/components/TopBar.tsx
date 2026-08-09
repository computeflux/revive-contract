import { Globe } from "lucide-react";
import { useEnv } from "../EnvContext";

// TopBar — 全局顶栏（右上角环境切换，选择持久化到 localStorage）
export function TopBar() {
  const { envs, env, setEnv } = useEnv();

  return (
    <div className="h-15 flex items-center justify-end gap-3 px-4 border-b border-(--border-default) bg-(--bg-surface)">
      <span className="text-[11px] font-bold uppercase text-(--text-muted) flex items-center gap-1.5">
        <Globe className="h-3.5 w-3.5" />
        环境
      </span>
      <select
        value={env}
        onChange={(e) => setEnv(e.target.value)}
        className="border border-(--border-subtle) bg-(--bg-surface) px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-(--text-secondary)"
      >
        {envs.map((e) => (
          <option key={e.name} value={e.name}>
            {e.name}
          </option>
        ))}
      </select>
    </div>
  );
}
