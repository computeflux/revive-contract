// EnvContext.tsx — 全局环境状态
//
// 右上角切换环境后，所有页面（部署/调试/ERC20/账户）共享同一个环境，
// 使用该环境 configs/<env>.json 的地址和合约。
// 选择会持久化到 localStorage，刷新页面后自动恢复上次的环境。

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { fetchEnvs, type EnvPublic } from "./api";

const STORAGE_KEY = "wetee.contract.env";

interface EnvState {
  envs: EnvPublic[];
  env: string;
  setEnv: (env: string) => void;
}

const EnvContext = createContext<EnvState>({ envs: [], env: "", setEnv: () => {} });

function loadSavedEnv(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || "";
  } catch {
    return "";
  }
}

function saveEnv(name: string) {
  try {
    localStorage.setItem(STORAGE_KEY, name);
  } catch {
    // localStorage 不可用时忽略（隐私模式等）
  }
}

export function EnvProvider({ children }: { children: ReactNode }) {
  const [envs, setEnvs] = useState<EnvPublic[]>([]);
  // 初始值优先取上次保存的环境
  const [env, setEnv] = useState<string>(() => loadSavedEnv());

  useEffect(() => {
    fetchEnvs()
      .then(({ envs }) => {
        setEnvs(envs);
        if (envs.length > 0) {
          // 上次保存的环境仍然存在则恢复，否则回退到第一个
          const saved = loadSavedEnv();
          const valid = envs.some((e) => e.name === saved);
          setEnv(valid ? saved : envs[0].name);
        }
      })
      .catch(() => {
        // 后端不可达时保持空，页面会提示
      });
  }, []);

  const changeEnv = (name: string) => {
    setEnv(name);
    saveEnv(name);
  };

  return (
    <EnvContext.Provider value={{ envs, env, setEnv: changeEnv }}>
      {children}
    </EnvContext.Provider>
  );
}

export function useEnv(): EnvState {
  return useContext(EnvContext);
}
