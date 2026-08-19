import { Orbit } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

export function AuthLayout({ title, subtitle, children }: { title: string; subtitle: string; children: ReactNode }) {
  return (
    <main className="auth-page">
      <section className="auth-panel" aria-labelledby="auth-title">
        <Link className="auth-brand" to="/" aria-label="Astro 首页">
          <Orbit size={31} aria-hidden="true" />
          <span>Astro</span>
        </Link>
        <header className="auth-heading">
          <h1 id="auth-title">{title}</h1>
          <p>{subtitle}</p>
        </header>
        {children}
      </section>
    </main>
  );
}
