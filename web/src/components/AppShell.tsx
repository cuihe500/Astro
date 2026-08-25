import { LogOut, Orbit } from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { clearSession } from "../features/auth/session";
import { projectsPath } from "../lib/routes";

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
          <NavLink className="brand" to={projectsPath} aria-label="Astro 项目列表">
            <Orbit size={25} aria-hidden="true" />
            <span>Astro</span>
          </NavLink>
          <nav className="main-nav" aria-label="主导航">
            <NavLink to={projectsPath} end>
              项目
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
