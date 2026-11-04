import Link from "next/link";
import type { ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeSlug from "rehype-slug";
import { documentation, documentationHref } from "@/content/documentation";
import { Navigation } from "@/components/navigation";
import { SiteFooter } from "@/components/site-footer";
import { site } from "@/lib/site";
import styles from "./documentation-page.module.css";

export function DocumentationPage({
  title,
  description,
  slug,
  children,
}: {
  title: string;
  description: string;
  slug?: string;
  children: ReactNode;
}) {
  const links = (
    <>
      <Link href="/#install">Install Burrow</Link>
      {documentation.map((item) => (
        <Link
          key={item.slug}
          href={`/docs/${item.slug}`}
          aria-current={slug === item.slug ? "page" : undefined}
        >
          {item.title}
        </Link>
      ))}
      <Link href="/support">Get support</Link>
    </>
  );
  return (
    <>
      <Navigation github={site.github} />
      <main id="main" className={`page-container ${styles.page}`}>
        <Link href="/docs" className={styles.back}>
          Burrow documentation
        </Link>
        <header className={styles.heading}>
          <h1>{title}</h1>
          <p>{description}</p>
        </header>
        <div className={styles.layout}>
          <div className={styles.sidebar}>
            <details className={styles.mobileNavigation}>
              <summary>Guides and reference</summary>
              <nav className={styles.navigation} aria-label="Documentation">
                {links}
              </nav>
            </details>
            <nav
              className={`${styles.navigation} ${styles.desktopNavigation}`}
              aria-label="Documentation"
            >
              {links}
            </nav>
          </div>
          <article className={styles.body}>{children}</article>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}

export function DocumentationMarkdown({
  markdown,
  source,
}: {
  markdown: string;
  source: string;
}) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeSlug]}
      skipHtml
      components={{
        a: ({ href, children }) => (
          <a href={href ? documentationHref(href, source) : undefined}>
            {children}
          </a>
        ),
        pre: ({ children }) => (
          <pre tabIndex={0} aria-label="Code example">
            {children}
          </pre>
        ),
        table: ({ children }) => (
          <div
            className={styles.table}
            role="region"
            aria-label="Reference table"
            tabIndex={0}
          >
            <table>{children}</table>
          </div>
        ),
      }}
    >
      {markdown}
    </ReactMarkdown>
  );
}
