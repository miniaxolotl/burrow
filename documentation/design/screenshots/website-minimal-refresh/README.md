# Minimal website refresh

Captured on 22 October 2026 from the local production build of
`feature/website-minimal-refresh`, in Chromium at 1440px and 390px. Reduced motion
is enabled for stable full-page images; the marketing palette remains light in
either system theme.

- [Desktop](desktop.png)
- [Mobile](mobile.png)
- [Example request logs](requests.png)
- [Cookie Settings hover](consent-hover.png)

Open feature rows, quiet dividers and compact clay-accent controls share the
family's layout rhythm. The terminal example types its command, connects the
route and reveals the response once. Replay is keyboard-operable. Selecting a
tunnel and viewing its requests keeps the selected example; transitions respect
reduced motion. The approved 07L / Arc identity and IBM Plex fonts are retained.

Consent buttons use explicit primary, secondary and Settings states. The new
regression test reproduced 1:1 hover contrast on the previous build and checks
normal, hover, press and keyboard focus. Additional tests cover replay under
reduced motion and selected-tunnel request navigation.

Screenshot follow-up: primary actions are 34px high on desktop; header, manager
tabs, replay and consent controls are 32px. Cookie settings actions have no
underline in any state.

Verification includes website lint/types/build, 23 browser cases and all three
Go modules. Responsive review covers WCAG A/AA axe, overflow and console errors
at 320/390/768/1440px. Existing original diagrams and sample requests retain
their example labels; no live tunnel or customer traffic is represented.
