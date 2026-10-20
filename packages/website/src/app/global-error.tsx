"use client";

import { useEffect } from "react";
import * as Sentry from "@sentry/nextjs";

export default function GlobalError({
  error,
  retry,
}: {
  error: unknown;
  retry: () => void;
}) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);
  return (
    <html lang="en">
      <body>
        <main>
          <h1>Something went wrong.</h1>
          <p>Please try loading Burrow again.</p>
          <button onClick={retry}>Try again</button>
        </main>
      </body>
    </html>
  );
}
