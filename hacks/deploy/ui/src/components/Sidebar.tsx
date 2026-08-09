import { Link, useLocation } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import {
  LayoutDashboard,
  Rocket,
  Bug,
  Coins,
  User,
  ChevronDown,
  Check,
  Globe,
  Workflow,
  Terminal,
  Radar,
  type LucideIcon,
} from "lucide-react";

interface ServiceEntry {
  name: string;
  label: string;
  description?: string;
  url: string;
}

// 按 urls.json 中的 name 匹配图标（默认 Globe）
const serviceIcons: Record<string, LucideIcon> = {
  workflow: Workflow,
  "contract-console": Terminal,
  "chain-monitor": Radar,
};

// urls.json 中 host 使用 xxxx 占位符，打开时替换为当前主机名
function resolveServiceUrl(raw: string): string {
  try {
    const u = new URL(raw);
    if (u.hostname === "xxxx") u.hostname = window.location.hostname;
    return u.toString();
  } catch {
    return raw;
  }
}

export function Sidebar() {
  const location = useLocation();
  const [services, setServices] = useState<ServiceEntry[]>([]);
  const [open, setOpen] = useState(false);
  const [currentUrl, setCurrentUrl] = useState<string | null>(null);
  const switcherRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // 与 workflow(board) 共享同一份服务列表：从 5173 的 urls.json 获取
    fetch(resolveServiceUrl("http://xxxx:5173/urls.json"))
      .then((r) => {
        if (!r.ok) throw new Error(`urls.json: ${r.status}`);
        return r.json();
      })
      .then((data: { services?: ServiceEntry[] }) => {
        const list = data.services || [];
        setServices(list);
        const resolved = list.map((s) => ({ ...s, url: resolveServiceUrl(s.url) }));
        const cur = resolved.find((s) => {
          try {
            return new URL(s.url).host === window.location.host;
          } catch {
            return false;
          }
        });
        setCurrentUrl(cur ? cur.url : null);
      })
      .catch(() => setServices([]));
  }, []);

  // 点击下拉外部时关闭
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (switcherRef.current && !switcherRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  const navItems = [
    { path: "/", label: "总览", icon: LayoutDashboard },
    { path: "/deploy", label: "合约部署", icon: Rocket },
    { path: "/debug", label: "合约调试", icon: Bug },
    { path: "/erc20", label: "ERC20 管理", icon: Coins },
    { path: "/account", label: "账户管理", icon: User },
  ];

  const currentService = services.find((s) => resolveServiceUrl(s.url) === currentUrl);

  return (
    <aside className="w-44 flex flex-col bg-(--bg-sidebar)">
      <div ref={switcherRef} className="relative">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          title="切换服务"
          className="w-full py-4 pl-5 pr-3 border-b border-(--border-default) flex items-center gap-2 hover:bg-(--bg-elevated) transition-colors cursor-pointer"
        >
          <img src="/logo-mini.svg" alt="logo" className="h-7 w-auto shrink-0" />
          <span className="text-lg text-white tracking-tight flex-1 text-left truncate">
            {currentService?.label ?? "weTEE 合约台"}
          </span>
          <ChevronDown
            className={`h-3.5 w-3.5 text-(--text-secondary) transition-transform ${open ? "rotate-180" : ""}`}
          />
        </button>
        {open && (
          <div className="absolute top-full left-0 w-52 z-20 border border-(--border-default) bg-(--bg-sidebar) shadow-2xl">
            <div className="px-4 pt-3 pb-2 text-[10px] font-bold uppercase tracking-widest text-(--text-muted) border-b border-(--border-divider)">
              切换服务
            </div>
            {services.length === 0 ? (
              <div className="px-4 py-3 text-xs text-(--text-muted)">服务列表不可用</div>
            ) : (
              services.map((s) => {
                const url = resolveServiceUrl(s.url);
                const isCurrent = url === currentUrl;
                const Icon = serviceIcons[s.name] || Globe;
                return (
                  <a
                    key={s.name || url}
                    href={url}
                    onClick={(e) => {
                      if (isCurrent) e.preventDefault();
                      setOpen(false);
                    }}
                    className={`group flex items-center gap-3 px-4 py-3 transition-colors ${isCurrent ? "bg-(--bg-elevated)" : "hover:bg-(--bg-elevated)"}`}
                  >
                    <span
                      className={`shrink-0 transition-colors ${isCurrent ? "text-white" : "text-(--text-secondary) group-hover:text-white"}`}
                    >
                      <Icon className="h-4 w-4" />
                    </span>
                    <span
                      className={`flex-1 text-sm font-medium truncate transition-colors ${isCurrent ? "text-white" : "text-(--text-secondary) group-hover:text-white"}`}
                    >
                      {s.label}
                    </span>
                    {isCurrent && <Check className="h-3.5 w-3.5 shrink-0 text-white" />}
                  </a>
                );
              })
            )}
          </div>
        )}
      </div>
      <nav className="flex-1 p-2 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          return (
            <Link
              key={item.path}
              to={item.path}
              className={`w-full flex items-center gap-3 px-4 py-2.5 font-medium text-sm rounded-none transition-colors ${isActive ? "bg-(--bg-elevated) text-white" : "text-(--text-secondary) hover:bg-(--bg-elevated) hover:text-white"}`}
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="p-4 border-t border-(--border-default) text-[10px] text-(--text-muted) leading-relaxed">
        后端: cmd/api-server
        <br />
        合约: token / subnet / proxy
      </div>
    </aside>
  );
}
