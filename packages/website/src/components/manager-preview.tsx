"use client";

import {
  ArrowLeft,
  ArrowRight,
  ChevronRight,
  Link2,
  List,
  ListFilter,
  Terminal,
} from "lucide-react";
import { useState } from "react";
import { CopyButton } from "@/components/copy-button";

const views = [
  { name: "Tunnel manager", icon: List },
  { name: "Request logs", icon: ListFilter },
] as const;

const tunnels = [
  {
    id: "softly-brave-little-fox",
    name: "Frontend",
    port: 3000,
    latency: "12 ms",
    transferred: "42.8 KB",
    requests: [
      {
        time: "14:32:08",
        method: "GET",
        path: "/",
        size: "4.2 KB",
        duration: "12 ms",
      },
      {
        time: "14:32:08",
        method: "GET",
        path: "/assets/app.js",
        size: "31.4 KB",
        duration: "18 ms",
      },
      {
        time: "14:32:09",
        method: "GET",
        path: "/assets/app.css",
        size: "6.3 KB",
        duration: "9 ms",
      },
      {
        time: "14:32:12",
        method: "GET",
        path: "/favicon.svg",
        size: "0.9 KB",
        duration: "8 ms",
      },
    ],
  },
  {
    id: "quietly-calm-green-owl",
    name: "API",
    port: 4000,
    latency: "16 ms",
    transferred: "18.2 KB",
    requests: [
      {
        time: "14:32:05",
        method: "GET",
        path: "/api/projects",
        size: "8.4 KB",
        duration: "16 ms",
      },
      {
        time: "14:32:07",
        method: "POST",
        path: "/api/projects",
        size: "1.2 KB",
        duration: "24 ms",
      },
      {
        time: "14:32:08",
        method: "GET",
        path: "/api/projects/1",
        size: "3.8 KB",
        duration: "14 ms",
      },
      {
        time: "14:32:10",
        method: "GET",
        path: "/api/activity",
        size: "4.8 KB",
        duration: "11 ms",
      },
    ],
  },
  {
    id: "brightly-bold-blue-hare",
    name: "Webhooks",
    port: 8080,
    latency: "9 ms",
    transferred: "6.1 KB",
    requests: [
      {
        time: "14:31:54",
        method: "POST",
        path: "/webhooks/payment",
        size: "2.1 KB",
        duration: "9 ms",
      },
      {
        time: "14:32:01",
        method: "POST",
        path: "/webhooks/order",
        size: "1.6 KB",
        duration: "12 ms",
      },
      {
        time: "14:32:06",
        method: "POST",
        path: "/webhooks/payment",
        size: "2.1 KB",
        duration: "10 ms",
      },
      {
        time: "14:32:12",
        method: "GET",
        path: "/health",
        size: "0.3 KB",
        duration: "4 ms",
      },
    ],
  },
] as const;

export function ManagerPreview() {
  const [view, setView] = useState(0);
  const [selectedTunnel, setSelectedTunnel] = useState(0);
  const tunnel = tunnels[selectedTunnel];
  const command = `burrowctl tunnel ${tunnel.port}`;

  return (
    <figure className="min-w-0">
      <div className="flex flex-wrap items-center justify-between gap-4 px-1 pb-5">
        <div
          className="inline-flex gap-1 rounded-lg border border-white/15 p-1"
          aria-label="Terminal preview views"
        >
          {views.map(({ name, icon: Icon }, index) => (
            <button
              key={name}
              type="button"
              aria-pressed={index === view}
              aria-controls="manager-preview-content"
              onClick={() => setView(index)}
              className="inline-flex min-h-10 items-center gap-2 rounded-md px-3 text-[11px] text-white/70 transition-colors hover:text-white aria-pressed:bg-lime aria-pressed:text-forest sm:px-5 sm:text-xs"
            >
              <Icon size={14} aria-hidden="true" /> {name}
            </button>
          ))}
        </div>
        <span className="hidden font-mono text-[10px] tracking-wider text-white/55 sm:block">
          SELECT A TUNNEL TO EXPLORE
        </span>
      </div>
      <div className="manager-window">
        <div className="terminal-titlebar">
          <span className="flex gap-1.5" aria-hidden="true">
            <i className="size-2 rounded-full bg-[#c18b7b]" />
            <i className="size-2 rounded-full bg-[#c3b777]" />
            <i className="size-2 rounded-full bg-[#82a682]" />
          </span>
          <span className="flex items-center gap-2">
            <Terminal size={12} aria-hidden="true" /> burrowctl
          </span>
          <span className="text-[9px] tracking-wider">DEMO</span>
        </div>

        <div className="px-4 pb-5 pt-6 sm:px-6 sm:pb-6">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className="font-mono text-xs font-semibold tracking-wider text-lime">
              BURROW
            </span>
            <span className="h-px flex-1 bg-lime/20" aria-hidden="true" />
            <span className="font-mono text-[10px] text-terminal-muted">
              brmbl.dev
            </span>
          </div>
          <div id="manager-preview-content" className="mt-6 min-h-[260px]">
            {view === 0 ? (
              <div>
                <div className="mb-4 flex flex-wrap items-center justify-between gap-2 font-mono text-[10px]">
                  <span className="text-terminal-muted">
                    3 of your tunnels. One place.
                  </span>
                  <span className="flex items-center gap-1.5 text-lime">
                    <span className="size-1.5 rounded-full bg-lime" /> All
                    connected
                  </span>
                </div>
                <div
                  className="manager-columns manager-column-headings"
                  aria-hidden="true"
                >
                  <span>Tunnel ID</span>
                  <span>Port</span>
                  <span>Latency</span>
                  <span className="hidden lg:block">Reconnects</span>
                  <span>Transferred</span>
                  <span>Status</span>
                </div>
                <ul aria-label="Example tunnels" className="space-y-1">
                  {tunnels.map((item, index) => (
                    <li key={item.id}>
                      <button
                        type="button"
                        aria-pressed={index === selectedTunnel}
                        aria-label={`Select ${item.name.toLowerCase()} tunnel on port ${item.port}`}
                        onClick={() => setSelectedTunnel(index)}
                        className="manager-columns manager-tunnel"
                      >
                        <span className="min-w-0">
                          <span className="flex items-center gap-2">
                            <ChevronRight
                              size={13}
                              className={
                                index === selectedTunnel
                                  ? "shrink-0 text-lime"
                                  : "shrink-0 opacity-25"
                              }
                              aria-hidden="true"
                            />
                            <span className="break-all">{item.id}</span>
                          </span>
                          <span className="ml-[21px] mt-2 flex flex-wrap gap-x-3 gap-y-1 text-[10px] text-terminal-muted md:hidden">
                            <span>:{item.port}</span>
                            <span>{item.latency}</span>
                            <span>{item.transferred}</span>
                          </span>
                        </span>
                        <span className="hidden md:block">{item.port}</span>
                        <span className="hidden md:block">{item.latency}</span>
                        <span className="hidden text-terminal-muted lg:block">
                          0
                        </span>
                        <span className="hidden md:block">
                          {item.transferred}
                        </span>
                        <span className="flex items-center gap-1.5 text-[10px] text-lime">
                          <span className="size-1.5 shrink-0 rounded-full bg-lime" />{" "}
                          active
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
                <div className="mt-5 border-t border-white/10 pt-4">
                  <div className="mb-2 flex items-center gap-2 font-mono text-[10px] text-terminal-muted">
                    <Link2 size={12} aria-hidden="true" /> {tunnel.name}{" "}
                    <ArrowRight size={12} aria-hidden="true" /> localhost:
                    {tunnel.port}
                  </div>
                  <p className="break-all font-mono text-[11px] leading-6 text-lime">
                    https://{tunnel.id}.brmbl.dev
                  </p>
                </div>
              </div>
            ) : (
              <div>
                <div className="mb-4 flex flex-wrap items-center justify-between gap-3 font-mono text-[10px]">
                  <span className="text-terminal-muted">
                    {tunnel.name} / localhost:{tunnel.port}
                  </span>
                  <button
                    type="button"
                    onClick={() => setView(0)}
                    className="inline-flex min-h-8 items-center gap-1.5 text-lime underline-offset-4 hover:underline"
                  >
                    <ArrowLeft size={12} aria-hidden="true" /> Your tunnels
                  </button>
                </div>
                <table className="manager-requests">
                  <caption className="sr-only">
                    Example requests for the {tunnel.name.toLowerCase()} tunnel
                    on port {tunnel.port}
                  </caption>
                  <thead>
                    <tr>
                      <th scope="col" className="hidden w-24 md:table-cell">
                        Time
                      </th>
                      <th scope="col" className="w-14 sm:w-20">
                        Method
                      </th>
                      <th scope="col">Path</th>
                      <th scope="col" className="hidden w-24 sm:table-cell">
                        Size
                      </th>
                      <th scope="col" className="w-16 text-right">
                        Duration
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {tunnel.requests.map((request) => (
                      <tr key={`${request.time}-${request.path}`}>
                        <td className="hidden text-terminal-muted md:table-cell">
                          {request.time}
                        </td>
                        <td>
                          <span
                            className={
                              request.method === "POST"
                                ? "text-[#e7c792]"
                                : "text-lime"
                            }
                          >
                            {request.method}
                          </span>
                        </td>
                        <td className="break-all pr-3">{request.path}</td>
                        <td className="hidden text-terminal-muted sm:table-cell">
                          {request.size}
                        </td>
                        <td className="text-right text-terminal-muted">
                          {request.duration}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <p className="mt-4 font-mono text-[10px] text-terminal-muted">
                  4 requests, example traffic
                </p>
              </div>
            )}
          </div>
          <div className="manager-actions">
            <div className="flex min-w-0 items-center gap-2 text-terminal-muted">
              <span className="text-lime" aria-hidden="true">
                $
              </span>
              <code className="min-w-0 break-all text-[10px] leading-5 sm:text-[11px]">
                {command}
              </code>
              <CopyButton
                key={command}
                text={command}
                label={`Copy command for port ${tunnel.port}`}
              />
            </div>
            <button
              type="button"
              onClick={() => setView(view === 0 ? 1 : 0)}
              className="inline-flex min-h-10 shrink-0 items-center gap-2 rounded-md border border-white/15 px-3 text-[11px] text-lime transition-colors hover:border-lime/50 hover:bg-white/5"
            >
              {view === 0 ? "View requests" : "Back to tunnels"}
              <ArrowRight size={13} aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
      <figcaption className="mt-4 flex flex-wrap justify-between gap-2 text-[11px] text-white/60">
        <span>Built for the terminal. Recreated for the web.</span>
        <span className="font-mono text-[10px]">
          Interactive preview with example data
        </span>
      </figcaption>
    </figure>
  );
}
