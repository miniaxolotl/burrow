"use client";

import { ArrowDownToLine, ArrowUpRight, Check } from "lucide-react";
import { useState } from "react";
import Link from "next/link";
import { CopyButton } from "@/components/copy-button";

const installers = {
  npm: "npm install -g @miniaxolotl/burrowctl",
  pnpm: "pnpm add -g @miniaxolotl/burrowctl",
};

export function InstallCommands({ releases }: { releases: string }) {
  const [installer, setInstaller] = useState<keyof typeof installers>("npm");

  return (
    <div
      className="install-guide min-w-0 rounded-xl bg-paper p-5 sm:p-7"
      data-ui
    >
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
          instruction:
            "Run this once in your terminal. Node.js is required for the package-manager install.",
        },
        {
          number: "02",
          title: "Open a tunnel to your app",
          command: "burrowctl tunnel 3000",
          instruction:
            "Start your app first. Replace 3000 with the port where it is running, then run this in another terminal.",
        },
      ].map(({ number, title, command, instruction }) => (
        <div key={number} className="mt-6">
          <p className="mb-3 flex items-center gap-3 text-xs font-medium">
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
              label={
                number === "01"
                  ? `Copy ${installer} install command`
                  : "Copy tunnel command"
              }
              installPlacement={number === "01" ? "install" : undefined}
            />
          </div>
          <p className="install-instruction">{instruction}</p>
        </div>
      ))}
      <div className="install-checkpoint">
        <Check size={20} aria-hidden="true" />
        <div>
          <h3>Open the HTTPS link</h3>
          <p>
            Copy the URL shown in the tunnel manager into a browser. You should
            see your local app. Keep the tunnel terminal running while you share
            it.
          </p>
        </div>
      </div>
      <div className="mt-6 flex flex-wrap items-center justify-between gap-3 pt-5 text-sm">
        <Link href="/docs/client" className="text-link">
          Installation and troubleshooting
        </Link>
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
