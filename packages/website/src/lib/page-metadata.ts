import type { Metadata } from "next";
export function pageMetadata(
  title: string,
  description: string,
  path: string,
): Metadata {
  return {
    title,
    description,
    alternates: { canonical: path },
    openGraph: { title: `${title} | Burrow`, description, url: path },
    twitter: { title: `${title} | Burrow`, description },
  };
}
