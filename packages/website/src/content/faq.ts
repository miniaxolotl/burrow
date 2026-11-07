export const faqs = [
  {
    question: "Does Burrow need an account?",
    answer:
      "No. The CLI creates a private owner credential automatically on first use, one per server, with no signup or login. Keep ~/.config/burrow/owners/ private and backed up; a public tunnel URL never grants management access.",
  },
  {
    question: "Where is my owner credential stored?",
    answer:
      "Under ~/.config/burrow/owners/, in a separate private file for each server. Keep that directory private and backed up. Deleting it is not a cleanup step and may leave you unable to manage existing tunnels.",
  },
  {
    question: "How do I turn off telemetry?",
    answer:
      "Run burrowctl telemetry disable, or set DO_NOT_TRACK=1 or BURROW_TELEMETRY=0. Official releases collect anonymous command outcomes and sanitized diagnostics only, with no account identity and no command arguments, URLs, credentials or tunnel content.",
  },
  {
    question: "What if the package-manager install fails?",
    answer:
      "Check that Node.js and npm or pnpm are available in your terminal. For a global-directory permission error, configure a writable package-manager prefix or use the native binary for your operating system and architecture. Avoid changing system permissions just to install the CLI.",
  },
  {
    question: "What should I include in a support request?",
    answer:
      "Include your operating system, client version, the command shape with private values removed and whether the same request works locally. Do not include your owner credential, authorization headers or private request bodies.",
  },
] as const;
