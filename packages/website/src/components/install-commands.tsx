"use client";

import { ArrowDownToLine, ArrowUpRight, Check } from "lucide-react";
import { useState } from "react";
import { CopyButton } from "@/components/copy-button";

const installers = {
  npm: "npm install -g @miniaxolotl/burrowctl",
  pnpm: "pnpm add -g @miniaxolotl/burrowctl",
};

export function InstallCommands({ releases }: { releases: string }) {
  const [installer, setInstaller] = useState<keyof typeof installers>("npm");

  return (
    <div className="min-w-0 rounded-xl border border-line bg-paper p-5 sm:p-7">
      <div className="mb-7 flex items-center justify-between gap-3">
        <span className="flex items-center gap-2 text-xs font-medium">
          <ArrowDownToLine size={15} aria-hidden="true" /> Quick start
        </span>
        <div
          className="flex rounded-lg bg-mist/50 p-1"
          aria-label="Package manager"
        >
          {(Object.keys(installers) as (keyof typeof installers)[]).map(
            (name) => (
              <button
                key={name}
                type="button"
                aria-pressed={installer === name}
                onClick={() => setInstaller(name)}
                className="min-h-9 rounded-md px-4 font-mono text-xs transition-colors hover:text-forest aria-pressed:bg-forest aria-pressed:text-paper"
              >
                {name}
              </button>
            ),
          )}
        </div>
      </div>
      {[
        {
          number: "01",
          title: "Install the CLI",
          command: installers[installer],
        },
        {
          number: "02",
          title: "Open a tunnel to your app",
          command: "burrowctl tunnel 3000",
        },
      ].map(({ number, title, command }) => (
        <div key={number} className="mt-6">
          <p className="mb-3 flex items-center gap-3 text-xs font-medium">
            <span className="font-mono text-[10px] text-muted">{number}</span>{" "}
            {title}
          </p>
          <div className="command-block">
            <code className="min-w-0 flex-1 break-all py-1 font-mono text-[11px] leading-6 sm:text-xs">
              <span className="mr-2 select-none text-lime" aria-hidden="true">
                $
              </span>
              {command}
            </code>
            <CopyButton
              key={command}
              text={command}
              label={`Copy ${title.toLowerCase()} command`}
            />
          </div>
        </div>
      ))}
      <p className="mt-5 text-xs leading-6 text-muted">
        Run your app first, swap <code className="text-forest">3000</code> for
        its port, and keep the terminal open. Your public link appears in the
        tunnel manager.
      </p>
      <div className="mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-line pt-5 text-[11px]">
        <span className="flex items-center gap-1.5 text-muted">
          <Check size={13} aria-hidden="true" /> Node.js required for npm / pnpm
        </span>
        <a
          href={releases}
          className="inline-flex min-h-9 items-center gap-1 font-semibold underline-offset-4 hover:underline"
        >
          Prefer a binary? <ArrowUpRight size={13} aria-hidden="true" />
        </a>
      </div>
    </div>
  );
}
