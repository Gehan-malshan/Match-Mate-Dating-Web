# MatchMate Base Design System — Midnight Chemistry

This document is the canonical visual-design source of truth for MatchMate applications and shared UI packages. It incorporates the approved **Midnight Chemistry** design reference into the repository while remaining subordinate to `AGENTS.md`, accepted ADRs, accessibility requirements, and MatchMate's privacy and safety boundaries.

## 1. Status and authority

- **DECISION:** Midnight Chemistry is the base visual language for the member web application, organizer/admin interfaces, and future shared design-system components.
- **DECISION:** New frontend work must use the semantic tokens and component rules in this guide instead of introducing isolated colors, spacing scales, radii, shadows, or typography without review.
- **DECISION:** Product security, privacy, moderation, and accessibility rules override decorative examples.
- **DECISION:** Existing UI is migrated through focused, tested changes; this document does not imply that every existing component is already fully tokenized.

When implementation and this guide disagree, record the mismatch in the change history and update the implementation or this guide in the same pull request.

## 2. Brand and style

Midnight Chemistry presents MatchMate as a premium, energetic dating experience that balances nighttime mystery with the warmth of a new connection. It combines restrained minimalism and glassmorphism on an obsidian backdrop, allowing the magenta-to-orange brand gradient to provide energy and direction.

The experience should feel:

- Private and trustworthy.
- Modern and editorial.
- Warm, intentional, and optimistic.
- Premium without becoming inaccessible or visually noisy.
- Energetic at moments of action, calm during forms, safety decisions, and sensitive workflows.

Deep neutral surfaces create depth. Translucent layers, purposeful typography, restrained glow, and original human imagery support the brand. Decorative intensity must never obscure content, state, consent, price, policy, validation, or safety information.

## 3. Canonical color tokens

Use semantic token names in components. Do not bind reusable components directly to raw hexadecimal values.

### Approved logo

The approved MatchMate logo is the rounded double-arch lowercase `m` mark supplied by the project owner. Its color travels from vivid magenta on the left through coral to sunset orange/gold on the right, with a restrained glow.

Canonical member-web assets:

| Asset | Purpose |
|---|---|
| `frontend/apps/web/public/brand/matchmate-logo-source.png` | Preserved supplied source artwork; do not load in routine navigation |
| `frontend/apps/web/public/brand/matchmate-logo-mark.png` | Transparent high-resolution web master |
| `frontend/apps/web/public/brand/matchmate-logo-nav.png` | Optimized navigation/footer mark |
| `frontend/apps/web/public/brand/matchmate-favicon.png` | Browser/favicon mark |

Logo rules:

- Preserve the double-arch silhouette, rounded geometry, gradient direction, and proportions.
- Keep sufficient clear space so the glow and outer edges are not cropped.
- Use the transparent mark on dark or approved contrasting surfaces; do not place the black-backed source artwork directly in compact UI.
- Do not recolor, rotate, stretch, outline, add a heart, add effects beyond the approved restrained glow, or recreate it with a text glyph.
- Pair the mark with the textual name `MatchMate` when context does not already provide the accessible brand name.
- Decorative logo images use empty alternative text inside a link or container that already has an accessible name. Standalone informational use requires `alt="MatchMate"`.
- New formats or geometry changes require owner approval and the documented design-change workflow.

### 3.1 Surfaces and content

| Token | Value | Primary use |
|---|---:|---|
| `background` | `#131316` | Default application background |
| `surface` / `surface-dim` | `#131316` | Base surface |
| `surface-container-lowest` | `#0E0E11` | Deepest structural surface |
| `surface-container-low` | `#1B1B1E` | Low-emphasis container |
| `surface-container` | `#1F1F22` | Standard card/container |
| `surface-container-high` | `#2A2A2D` | Raised container |
| `surface-container-highest` | `#353438` | Highest tonal container |
| `surface-bright` | `#39393C` | Brightest dark surface |
| `surface-variant` | `#353438` | Alternate raised surface |
| `on-background` / `on-surface` | `#E4E1E6` | Primary content on dark surfaces |
| `on-surface-variant` | `#E4BDC2` | Warm secondary content |
| `outline` | `#AB888C` | High-emphasis outline |
| `outline-variant` | `#5B3F43` | Subtle outline/divider |

`#0F0F12` is retained as the **Void** decorative backdrop for hero gradients, image overlays, and other deepest visual regions. It does not replace the canonical `background` token.

### 3.2 Brand and supporting colors

| Token | Value |
|---|---:|
| `primary` / `surface-tint` | `#FFB2BE` |
| `on-primary` | `#660025` |
| `primary-container` | `#FF4E7C` |
| `on-primary-container` | `#5A0020` |
| `inverse-primary` | `#BC004B` |
| `secondary` | `#FFBF81` |
| `on-secondary` | `#4A2800` |
| `secondary-container` | `#FF9800` |
| `on-secondary-container` | `#653900` |
| `tertiary` | `#CDBDFF` |
| `on-tertiary` | `#370096` |
| `tertiary-container` | `#9A7BFF` |
| `on-tertiary-container` | `#2F0084` |

The primary brand gradient is:

```css
linear-gradient(115deg, #ff006b 0%, #ff8c00 100%)
```

Use it for primary actions, active states, progress emphasis, selected navigation, and occasional editorial text. It is not a general background fill.

### 3.3 Inverse, fixed, and error colors

| Token | Value |
|---|---:|
| `inverse-surface` | `#E4E1E6` |
| `inverse-on-surface` | `#303033` |
| `primary-fixed` | `#FFD9DE` |
| `primary-fixed-dim` | `#FFB2BE` |
| `on-primary-fixed` | `#400014` |
| `on-primary-fixed-variant` | `#900038` |
| `secondary-fixed` | `#FFDCBE` |
| `secondary-fixed-dim` | `#FFB870` |
| `on-secondary-fixed` | `#2C1600` |
| `on-secondary-fixed-variant` | `#693C00` |
| `tertiary-fixed` | `#E8DEFF` |
| `tertiary-fixed-dim` | `#CDBDFF` |
| `on-tertiary-fixed` | `#20005F` |
| `on-tertiary-fixed-variant` | `#4F00D0` |
| `error` | `#FFB4AB` |
| `on-error` | `#690005` |
| `error-container` | `#93000A` |
| `on-error-container` | `#FFDAD6` |

Error, warning, success, pending, selected, disabled, and focus states require a non-color cue such as text, iconography, pattern, or shape.

## 4. Typography

Use **Plus Jakarta Sans** for display and headline text and **Manrope** for body, label, form, and functional text. Font files must be loaded through an approved, performant, privacy-aware method with appropriate fallbacks. The interface must remain usable if custom fonts fail.

| Token | Family | Size | Weight | Line height | Letter spacing |
|---|---|---:|---:|---:|---:|
| `display-lg` | Plus Jakarta Sans | 48px | 700 | 56px | `-0.02em` |
| `headline-lg` | Plus Jakarta Sans | 32px | 700 | 40px | `-0.01em` |
| `headline-lg-mobile` | Plus Jakarta Sans | 28px | 700 | 36px | normal |
| `headline-md` | Plus Jakarta Sans | 24px | 600 | 32px | normal |
| `body-lg` | Manrope | 18px | 400 | 28px | normal |
| `body-md` | Manrope | 16px | 400 | 24px | normal |
| `label-md` | Manrope | 14px | 600 | 20px | `0.05em` |
| `label-sm` | Manrope | 12px | 500 | 16px | normal |

Guidance:

- Tight headline spacing creates an impactful, branded tone.
- Body typography remains calm and highly readable on dark surfaces.
- Use gradient text only for short, decorative emphasis; retain a readable solid-color fallback.
- Mobile headings scale down approximately 15–20% and must avoid clipping or excessive line lengths.
- Do not use all-caps styling for long text, errors, instructions, or consent language.

## 5. Layout and spacing

The atomic spacing unit is **4px**. The primary visual rhythm uses multiples of **8px**, while 4px increments are permitted for fine alignment.

| Token | Value |
|---|---:|
| `space-xs` | 4px |
| `space-sm` | 8px |
| `space-md` | 16px |
| `space-lg` | 24px |
| `space-xl` | 40px |
| `container-max` | 1200px |

Responsive layout:

- Desktop: 12-column fluid grid, 24px gutters, centered content up to 1200px.
- Tablet/intermediate layouts: 20px page gutters unless a component specification requires more.
- Mobile: 4-column grid with 16px page margins.
- Use generous negative space around imagery and discovery surfaces, but keep task flows compact enough to scan.
- Prefer 8px, 16px, 24px, 32px, and 40px vertical relationships.
- Avoid fixed heights for user-generated copy, translated content, validation errors, or accessibility text scaling.

## 6. Elevation and depth

Use tonal layering and backdrop blur instead of heavy shadows.

1. **Floor — level 0:** Void/background region using `#0F0F12` or `background`.
2. **Surface — level 1:** Dark cards around `#1A1A1E`/`surface-container-low` with a subtle white 5% border.
3. **Float — level 2:** Glass overlays with up to 20px backdrop blur and a white 10% border.
4. **Pop — level 3:** Modals and critical temporary surfaces with a soft 30px primary-magenta glow at approximately 15% opacity.

Backdrop blur must have an opaque fallback. Text contrast must be verified against the composited background, not only the nominal token.

## 7. Shape language

| Token | Value | Typical use |
|---|---:|---|
| `radius-sm` | 4px | Compact controls and tags |
| `radius-default` | 8px | Buttons and input fields |
| `radius-md` | 12px | Medium containers |
| `radius-lg` | 16px | Cards and modules |
| `radius-xl` | 24px | Profile imagery and discovery cards |
| `radius-full` | 9999px | Pills and circular controls |

Icons use rounded caps and corners. Rounded shapes should support hierarchy and friendliness without making every container visually identical.

## 8. Base component specifications

### Buttons

- **Primary:** brand-gradient fill, white text, clear disabled/loading states, and a restrained magenta glow on hover.
- **Secondary:** transparent or dark surface with a visible gradient/primary outline and high-contrast text.
- **Ghost:** minimal surface treatment; text cannot begin below required contrast and then become readable only on hover.
- Interactive targets must be at least 44×44 CSS pixels where practical.

### Input fields

- Use `surface-container-low` fill and a visible 1px outline.
- Focus uses a high-contrast primary ring or border; do not rely on a low-contrast gradient alone.
- Visible labels use `label-md`; placeholders are examples, not labels.
- Error, help, pending, and success content remains associated programmatically with the field.

### Discovery cards

- Use large approved imagery with a bottom-up dark gradient to protect text contrast.
- The bottom-left metadata area may contain only privacy-approved community-profile fields, such as an approved nickname/display name and approved age or age band.
- Never display legal names, date of birth, exact address, contact details, social handles, private preferences, verification evidence, or moderation data.
- Cards require keyboard access, descriptive alternatives for meaningful imagery, loading/failure states, and clear block/report entry points where applicable.

Event discovery cards use the same tonal hierarchy without member imagery. They show only approved catalog fields: event name/description, broad location, schedule, fixed-precision price/currency, configured capacity, lifecycle label, and a detail action. Exact venue and organizer identifiers remain operational. Configured capacity always carries a text disclaimer that Booking is authoritative for availability.

### Organizer event operations

- Use compact, calm forms on `surface` containers; privileged work must favor legibility over decorative motion.
- Separate broad public location from exact operational venue with explicit labels.
- Show lifecycle state and optimistic version together, and provide only transitions valid for the current state.
- Cancellation requires an explicit reason and visually distinct destructive treatment; confirmation cannot rely on color alone.
- Capacity fields state that they configure policy and do not represent consumed or available seats.
- Loading, empty, error, stale-conflict, and successful-save messages use live status semantics and remain visible without animation.

### Chips and tags

- Use `radius-full` or `radius-xl`, a translucent white surface near 10%, and `label-sm` typography.
- Use only for approved interests, traits, filters, or statuses.
- Selection requires more than color alone and must be announced accessibly.

### Navigation

- Desktop uses a minimal top-aligned navigation bar.
- Mobile application routes may use a bottom-fixed glass navigation bar; public marketing pages may retain an accessible compact top navigation when more appropriate.
- Active items may use the brand gradient but also require a shape, label, weight, or indicator.
- Fixed navigation must respect safe-area insets and must not cover focused content, toasts, forms, or payment controls.

### Cards, dialogs, and overlays

- Standard cards use tonal surfaces and restrained borders.
- Dialogs require focus trapping, labelled titles, Escape behavior where safe, and focus restoration.
- Confirmation wording must clearly distinguish destructive, payment, privacy, reveal-consent, and moderation actions.
- Glass surfaces are decorative hierarchy, not a substitute for readable structure.

## 9. Imagery and motion

- Prefer original, inclusive imagery showing consenting adults in respectful social settings.
- Avoid staged swipe-app clichés, explicit intimacy, public figures, visible contact details, identity documents, or unapproved venue information.
- Decorative images use empty alternative text; meaningful images use concise descriptions.
- Motion should reinforce navigation, state change, and hierarchy—not create pressure in consent, payment, or safety decisions.
- Support `prefers-reduced-motion`; essential state changes must remain understandable without animation.

### 9.1 Public landing-page expression

The member landing page applies Midnight Chemistry as a cinematic AIDA narrative rather than a repeated card template:

- Attention uses the approved full-bleed event image, a strong left editorial gradient, a two-line headline, a restrained privacy note, and a glass navigation surface. Do not replace it with a detached portrait or partially visible decorative image.
- Interest introduces the product through a concise image-led About composition, then pairs one large visual with one open editorial column containing the privacy and mutual-choice principles instead of stacking separate cards. Sections use warm tonal changes rather than decorative grid lines or large empty gaps.
- Desire places local event imagery beside a fully visible five-stage process list with short, understandable headings and descriptions. A simple matching visual shows preferences moving through approved rules and human review, followed by event imagery that scales into view; decorative marquees and dashboard-like dividers are avoided.
- Action uses a truthful pre-launch state instead of an operational registration form or invented event inventory.

The public footer provides only real in-page navigation, trust guidance, development status, copyright, and a back-to-top action. Do not add placeholder social, contact, policy, or legal destinations before those routes and owners exist.

GSAP motion is progressive enhancement. Entrance, text-reveal, and image-scale effects must be scoped to the landing page, cleaned up with the owning React lifecycle, and skipped when reduced motion is requested. CSS animation must also stop under the same preference. Content cannot begin hidden in the reduced-motion or no-animation experience.

Use project-local approved imagery rather than third-party random-image services. Hover scaling stays inside clipped image containers, and decorative motion must not introduce horizontal scrolling.

## 10. Accessibility requirements

WCAG 2.2 AA is the minimum implementation target unless an approved policy requires a stronger target.

- Verify text, icon, border, focus, and interactive-state contrast on actual composited surfaces.
- Provide visible keyboard focus and logical focus order.
- Support keyboard-only interaction and screen-reader names, roles, values, errors, and status announcements.
- Do not communicate status, compatibility, availability, safety, or errors through color alone.
- Support text zoom/reflow and responsive layouts without loss of information or controls.
- Keep touch targets, labels, instructions, timeout behavior, and error recovery accessible.
- Run automated checks and manual keyboard/screen-reader review for critical journeys.

## 11. Privacy, safety, and product boundaries

This design system does not authorize new product behavior. All interfaces must continue to enforce:

- No member-to-member chat in the initial product.
- No publication of contact information or exact addresses.
- Private matchmaking preferences remain private.
- Matchmaking is deterministic and explainable without machine learning.
- Blocking, reporting, moderation, organizer review, consent, and audit states are first-class UI requirements.
- Payment success is confirmed from server state, not browser redirects alone.
- Unapproved events, metrics, availability, verification claims, or safety promises must not be presented as facts.

## 12. Implementation guidance

The target implementation location for reusable tokens and components is `frontend/packages/ui`; application-specific composition remains within the owning app.

When implementation begins or changes:

1. Represent colors, typography, spacing, radii, elevation, and motion as semantic tokens.
2. Provide dark-theme tokens first; do not infer or ship a light theme without an approved design.
3. Keep raw palette values in the token layer rather than repeated across components.
4. Build accessible primitives before feature-specific variants.
5. Add component examples/stories when the selected UI documentation tool is approved.
6. Add visual-regression coverage for stable shared components and critical responsive layouts.
7. Test hover, focus, active, disabled, loading, empty, error, success, pending, and restricted states.
8. Update this guide, affected app/package READMEs, tests, and change history in the same pull request.

## 13. Design change workflow

A change to a canonical token, typography role, spacing scale, radius, component state, responsive navigation rule, accessibility behavior, or privacy-sensitive presentation requires:

- A before/after pull-request summary following `docs/change-management/README.md`.
- Updates to this guide and affected app/package documentation.
- A migration note identifying affected components and screenshots/visual tests.
- Accessibility and privacy impact review.
- Focused implementation and verification; do not combine a broad redesign with unrelated product behavior.
- An ADR when the change introduces a major frontend framework, rendering model, theming architecture, or cross-application ownership decision.

## 14. Base-design review checklist

- [ ] Semantic tokens are used instead of isolated raw values.
- [ ] Plus Jakarta Sans and Manrope roles are followed with safe fallbacks.
- [ ] Layout follows the responsive grid and spacing rhythm.
- [ ] Elevation uses tonal layering and restrained glass effects.
- [ ] Focus, contrast, reduced motion, text scaling, and keyboard use are verified.
- [ ] Profile/event/payment/safety content respects the approved visibility policy.
- [ ] Loading, empty, pending, failure, disabled, and restricted states are designed.
- [ ] Mobile fixed navigation does not obscure content or controls.
- [ ] Relevant tests, screenshots, documentation, and change history are updated.
