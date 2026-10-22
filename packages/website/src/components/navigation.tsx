"use client";

import { ArrowUpRight, Menu, X } from "lucide-react";
import { useEffect, useRef } from "react";
import Link from "next/link";
import { Brand } from "@/components/brand";
import { GithubIcon } from "@/components/github-icon";

const links = [
  ["Features", "/#features"],
  ["How it works", "/#how-it-works"],
] as const;

export function Navigation({ github }: { github: string }) {
  const menu = useRef<HTMLDetailsElement>(null);
  useEffect(() => {
    const desktop = window.matchMedia("(min-width: 1024px)");
    const close = () => {
      if (menu.current) menu.current.open = false;
    };
    const dismissOutside = (event: PointerEvent) => {
      if (event.target instanceof Node && !menu.current?.contains(event.target))
        close();
    };
    document.addEventListener("pointerdown", dismissOutside);
    desktop.addEventListener("change", close);
    return () => {
      document.removeEventListener("pointerdown", dismissOutside);
      desktop.removeEventListener("change", close);
    };
  }, []);

  return (
    <header className="site-header">
      <div className="page-container header-inner">
        <Brand />
        <nav aria-label="Main navigation" className="header-navigation">
          {links.map(([label, href]) => (
            <Link href={href} className="nav-link" key={href}>
              {label}
            </Link>
          ))}
          <a href={github} className="nav-link flex items-center gap-2">
            <GithubIcon size={16} /> GitHub
          </a>
        </nav>
        <div className="header-actions">
          <Link
            href="/#install"
            className="button-primary shrink-0 whitespace-nowrap"
          >
            Install Burrow <ArrowUpRight size={15} aria-hidden="true" />
          </Link>
          <details
            ref={menu}
            className="header-menu group"
            onBlur={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget))
                event.currentTarget.open = false;
            }}
            onKeyDown={(event) => {
              if (event.key === "Escape" && menu.current) {
                menu.current.open = false;
                menu.current.querySelector("summary")?.focus();
              }
            }}
          >
            <summary className="header-menu-toggle">
              <Menu
                size={18}
                className="group-open:hidden"
                aria-hidden="true"
              />
              <X
                size={18}
                className="hidden group-open:block"
                aria-hidden="true"
              />
              <span className="sr-only">Navigation menu</span>
            </summary>
            <nav
              aria-label="Mobile navigation"
              className="absolute inset-x-0 top-full border-b border-line bg-paper p-6 shadow-lg"
            >
              {[
                ...links,
                ["GitHub", github],
                ["Install Burrow", "/#install"],
              ].map(([label, href]) => (
                <Link
                  key={href}
                  href={href}
                  className="nav-link flex border-b border-line py-3 last:border-0"
                  onClick={() => {
                    if (menu.current) menu.current.open = false;
                  }}
                >
                  {label}{" "}
                  <ArrowUpRight
                    size={15}
                    className="ml-auto"
                    aria-hidden="true"
                  />
                </Link>
              ))}
            </nav>
          </details>
        </div>
      </div>
    </header>
  );
}
