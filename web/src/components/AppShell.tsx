import { LogOut, Orbit, Plus } from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { clearSession } from "../features/auth/session";

export function AppShell() {
  const navigate = useNavigate();

  function logout() {
    clearSession();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-inner">
          <NavLink className="brand" to="/apps" aria-label="Astro 应用列表">
            <Orbit size={25} aria-hidden="true" />
            <span>Astro</span>
          </NavLink>
          <nav className="main-nav" aria-label="主导航">
            <NavLink to="/apps" end>
              应用
            </NavLink>
            <NavLink className="button button-primary button-small" to="/apps/new">
              <Plus size={17} aria-hidden="true" />
              创建应用
            </NavLink>
            <button className="icon-button" type="button" onClick={logout} aria-label="退出登录" title="退出登录">
              <LogOut size={19} aria-hidden="true" />
            </button>
          </nav>
        </div>
      </header>
      <main className="workspace">
        <Outlet />
      </main>
    </div>
  );
}
