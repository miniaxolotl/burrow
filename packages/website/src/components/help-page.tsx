import Link from "next/link";
import type { ReactNode } from "react";
import { Navigation } from "@/components/navigation";
import { SiteFooter } from "@/components/site-footer";
import { contact } from "@/lib/contact";
import { site } from "@/lib/site";
import styles from "./help-page.module.css";

export function HelpPage({
  title,
  description,
  page,
  children,
}: {
  title: string;
  description: string;
  page: "support" | "contact";
  children: ReactNode;
}) {
  return (
    <>
      <Navigation github={site.github} />
      <main id="main" className={`page-container ${styles.page}`}>
        <Link href="/" className={styles.back}>
          Back to Burrow
        </Link>
        <header className={styles.heading}>
          <h1>{title}</h1>
          <p>{description}</p>
        </header>
        <div className={styles.layout}>
          <aside className={styles.contact} aria-label="Email Burrow">
            <h2>Write to us.</h2>
            <a className={styles.email} href={contact.href}>
              {contact.email}
            </a>
            <p>
              One address for help, feedback, and privacy or security questions.
            </p>
            <nav aria-label="Help and contact">
              <Link
                href="/support"
                aria-current={page === "support" ? "page" : undefined}
              >
                Support
              </Link>
              <Link
                href="/contact"
                aria-current={page === "contact" ? "page" : undefined}
              >
                Contact
              </Link>
              <Link href="/privacy">Privacy policy</Link>
            </nav>
          </aside>
          <div className={styles.content}>{children}</div>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
