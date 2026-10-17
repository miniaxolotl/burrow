"use client";

import { ArrowUpRight, Menu, X } from "lucide-react";
import { useRef } from "react";
import { Brand } from "@/components/brand";
import { GithubIcon } from "@/components/github-icon";

const links = [
  ["Features", "#features"],
  ["How it works", "#how-it-works"],
] as const;

export function Navigation({ github }: { github: string }) {
  const menu = useRef<HTMLDetailsElement>(null);

  return (
    <header className="site-header">
      <div className="page-container flex h-[76px] items-center justify-between gap-2 sm:gap-4">
        <Brand />
        <nav
          aria-label="Main navigation"
          className="hidden items-center gap-8 lg:flex"
        >
          {links.map(([label, href]) => (
            <a href={href} className="nav-link" key={href}>
              {label}
            </a>
          ))}
          <a href={github} className="nav-link flex items-center gap-2">
            <GithubIcon size={16} /> GitHub
          </a>
        </nav>
        <div className="flex items-center gap-2 sm:gap-3">
          <a
            href="#install"
            className="button-primary min-h-11! shrink-0 gap-3! whitespace-nowrap px-3! py-2.5! text-xs! sm:px-4!"
          >
            Install Burrow{" "}
            <ArrowUpRight
              size={15}
              className="hidden sm:block"
              aria-hidden="true"
            />
          </a>
          <details
            ref={menu}
            className="group lg:hidden"
            onKeyDown={(event) => {
              if (event.key === "Escape" && menu.current) {
                menu.current.open = false;
                menu.current.querySelector("summary")?.focus();
              }
            }}
          >
            <summary className="grid size-11 cursor-pointer list-none place-items-center rounded-md border border-line [&::-webkit-details-marker]:hidden">
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
                ["Install Burrow", "#install"],
              ].map(([label, href]) => (
                <a
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
                </a>
              ))}
            </nav>
          </details>
        </div>
      </div>
    </header>
  );
}
