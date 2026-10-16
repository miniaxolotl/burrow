import type { MetadataRoute } from "next";
import { site } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      ...(site.indexable ? { allow: "/" } : { disallow: "/" }),
    },
    ...(site.indexable ? { sitemap: `${site.url}/sitemap.xml` } : {}),
  };
}
