# @script/release

Release admission and retained publication tooling. Follow
[the deployment guide](../../documentation/deploy.md); a source release is not a
delivery approval.

`promotion.mjs` binds an immutable Release Please source to independent full CI
and a frozen promotion prepared by Source Release. Site-only selection compares retained
delivery inputs against the verified production anchor in `policy.json`, repeats
the comparison during admission/deployment, and records the exact identities and
input digest. Unknown or changed service/install inputs fail closed. There is no
automatic full-publication fallback, npm bump, installer approval or baseline
advance.

The production workflow retains a combined publisher (`src/index.ts`,
`upload-assets.mjs` and image publication), but site-only skips it and the dependent
service rollout. Do not invoke publishing commands to recover a refused candidate.
A targeted delivery needs a separate reviewed decision and verification.

Local validation, with no release credentials or remote writes:

```bash
node --test scripts/release/*.test.mjs
```
