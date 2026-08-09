import { BrowserRouter, Routes, Route } from "react-router-dom";
import { EnvProvider } from "./EnvContext";
import { Sidebar } from "./components/Sidebar";
import { TopBar } from "./components/TopBar";
import { Dashboard } from "./pages/Dashboard";
import { Deploy } from "./pages/Deploy";
import { ContractDebug } from "./pages/ContractDebug";
import { Account } from "./pages/Account";
import { Erc20Manager } from "./pages/Erc20Manager";

export default function App() {
    return (
        <BrowserRouter>
            <EnvProvider>
                <div className="flex h-screen text-[#FAFAFA] overflow-hidden">
                    <Sidebar />
                    <div className="flex-1 flex flex-col min-w-0">
                        <TopBar />
                        <div className="flex-1 overflow-y-auto">
                            <Routes>
                                <Route path="/" element={<Dashboard />} />
                                <Route path="/deploy" element={<Deploy />} />
                                <Route path="/debug" element={<ContractDebug />} />
                                <Route path="/erc20" element={<Erc20Manager />} />
                                <Route path="/account" element={<Account />} />
                            </Routes>
                        </div>
                    </div>
                </div>
            </EnvProvider>
        </BrowserRouter>
    );
}
