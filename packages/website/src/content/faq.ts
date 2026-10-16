export const faqs = [
  {
    question: "Do I need an account?",
    answer:
      "No. Burrow connects to brmbl.dev by default. Install the CLI and create a tunnel — no account or API key needed.",
  },
  {
    question: "Does my URL stay the same?",
    answer:
      "Each new tunnel gets a generated URL. Quick reconnects can preserve it, but longer interruptions may create a new one. Keep burrowctl running while you share your app.",
  },
  {
    question: "Can I use my own server and domain?",
    answer:
      "Yes. Run burrowd with Redis, configure your domain with wildcard DNS and TLS, then point the same CLI at your server. Burrow is MIT-licensed.",
  },
] as const;
