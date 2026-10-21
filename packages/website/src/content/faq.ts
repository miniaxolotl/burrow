export const faqs = [
  {
    question: "Do I need an account?",
    answer:
      "No. Install the CLI and run burrowctl tunnel 3000. It connects to brmbl.dev and creates a private owner credential automatically, stored on your machine. There is no login or manual key setup.",
  },
  {
    question: "Can other users see my tunnel sessions?",
    answer:
      "Your tunnel list, connection count and request logs require your private owner credential. Sharing a public URL does not let someone manage your tunnel or claim its ID.",
  },
  {
    question: "Is the app I share private?",
    answer:
      "Your tunnel URL is public. Use your app's own authentication if access should be restricted. The tunnel server handles the traffic; owner-only management does not hide HTTP contents from the hosting infrastructure.",
  },
  {
    question: "Does my URL stay the same?",
    answer:
      "Reconnects keep the same URL while the CLI is running. Another owner cannot claim your reserved ID. Closing the tunnel retires its URL; a new tunnel gets a new one.",
  },
  {
    question: "Can I use my own server and domain?",
    answer:
      "Yes. Run burrowd with Redis, configure your domain with wildcard DNS and TLS, then point the same CLI at your server. Burrow is MIT-licensed.",
  },
] as const;
