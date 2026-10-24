# Website support and contact

Current website source uses **support@useburrow.app** for general, support,
privacy, and private security inquiries. The public-only contact module at
`packages/website/src/lib/contact.ts` is shared by policies, footer, and recovery.
Keep unrelated author attribution and dated release evidence unchanged.

`packages/website/tests/support-contact.spec.ts` guards the public mailbox and
absence of obsolete current website contact links. Browser checks use a real
production build. No CLI, protocol, tunnel server, provider, or deployment
behavior is changed by this contact migration. Never email credentials or
private tunnel traffic as part of a support report.

The contact regression first failed on the obsolete links; after migration all
**24 Chromium cases**, root/site lint, typecheck, build, Go vet and Go tests pass.
The optional live tunnel target was not configured; no live tunnel acceptance is
claimed. [Reviewed contact capture](design/screenshots/website-support-contact/README.md).
