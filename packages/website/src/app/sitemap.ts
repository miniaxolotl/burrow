import type { MetadataRoute } from "next";
import { site } from "@/lib/site";

export default function sitemap(): MetadataRoute.Sitemap {
  return site.indexable
    ? [
        { url: site.url, changeFrequency: "monthly", priority: 1 },
        {
          url: `${site.url}/privacy`,
          changeFrequency: "yearly",
          priority: 0.2,
        },
        { url: `${site.url}/terms`, changeFrequency: "yearly", priority: 0.2 },
      ]
    : [];
}
