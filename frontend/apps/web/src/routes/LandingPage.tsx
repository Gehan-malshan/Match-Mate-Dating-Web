import type { CSSProperties, ReactNode } from 'react'

type IconName = 'arrow' | 'calendar' | 'check' | 'heart' | 'lock' | 'profile' | 'shield' | 'spark' | 'ticket' | 'users'

const paths: Record<IconName, ReactNode> = {
  arrow: <><path d="M5 12h14"/><path d="m13 6 6 6-6 6"/></>,
  calendar: <><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/></>,
  check: <path d="m5 12 4 4L19 6"/>,
  heart: <path d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8l1.1 1.1L12 21l7.7-7.5 1.1-1.1a5.5 5.5 0 0 0 0-7.8Z"/>,
  lock: <><rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></>,
  profile: <><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></>,
  shield: <><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/><path d="m9 12 2 2 4-4"/></>,
  spark: <><path d="m12 3-1.5 4.5L6 9l4.5 1.5L12 15l1.5-4.5L18 9l-4.5-1.5L12 3Z"/><path d="m5 15-.8 2.2L2 18l2.2.8L5 21l.8-2.2L8 18l-2.2-.8L5 15Z"/></>,
  ticket: <><path d="M2 9a3 3 0 0 0 0 6v3h20v-3a3 3 0 0 0 0-6V6H2v3Z"/><path d="M13 6v12"/></>,
  users: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/></>,
}

function Icon({ name }: { name: IconName }) {
  return <svg aria-hidden="true" className="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">{paths[name]}</svg>
}

const principles = [
  { value: 'Private', label: 'Sensitive preferences stay out of public profiles', icon: 'lock' as const },
  { value: 'Explainable', label: 'Clear, versioned compatibility rules—not ML', icon: 'spark' as const },
  { value: 'Real-world', label: 'Curated events designed for respectful connection', icon: 'users' as const },
]

const safeguards = [
  { icon: 'shield' as const, title: 'Privacy from the first profile', text: 'Only approved, non-sensitive profile details are shown. Contact details and private preferences stay protected.' },
  { icon: 'spark' as const, title: 'Transparent compatibility scoring', text: 'Hard preferences and weighted answers create reproducible suggestions. No hidden ML model, no inferred sensitive traits.' },
  { icon: 'heart' as const, title: 'Safety built into every step', text: 'Verification, blocking, reporting, organizer review, moderation, and consent controls shape the complete experience.' },
]

const steps = [
  { number: '01', icon: 'profile' as const, title: 'Create a private profile', text: 'Share only what helps us understand you. Preview exactly what other approved members may see.' },
  { number: '02', icon: 'calendar' as const, title: 'Choose a curated event', text: 'Explore eligible events, understand the format and policies, then reserve your place when registration opens.' },
  { number: '03', icon: 'ticket' as const, title: 'Confirm your booking', text: 'Complete the secure PayHere flow. A server-confirmed booking makes you eligible for that event’s matching pool.' },
  { number: '04', icon: 'users' as const, title: 'Meet with intention', text: 'Rule-based suggestions are reviewed before the event. You meet safely, respond privately, and control any reveal.' },
]

export function LandingPage() {
  return (
    <main>
      <header className="site-header">
        <a className="brand" href="#top" aria-label="MatchMate home"><span className="brand-mark" aria-hidden="true">M</span><span>MatchMate</span></a>
        <nav aria-label="Primary navigation"><a href="#how-it-works">How it works</a><a href="#events">Events</a><a href="#about">About</a></nav>
        <div className="header-actions"><a className="text-action" href="#join">Log in</a><a className="button button-small" href="#join">Join MatchMate</a></div>
      </header>

      <section className="hero" id="top">
        <div className="hero-backdrop" aria-hidden="true" />
        <div className="hero-content">
          <p className="eyebrow"><span /> Sri Lanka’s privacy-first dating events</p>
          <h1>Where mystery<br />meets <em>connection.</em></h1>
          <p className="hero-copy">Meet with intention through verified profiles, curated gatherings, and transparent compatibility rules—without endless swiping or chat.</p>
          <div className="hero-actions"><a className="button" href="#events">Explore events <Icon name="arrow" /></a><a className="button button-ghost" href="#how-it-works">See how it works</a></div>
        </div>
        <div className="trust-card" aria-label="MatchMate privacy principle"><span className="trust-dot" /><div><strong>Private by design</strong><small>Your sensitive preferences stay private.</small></div></div>
        <a className="scroll-cue" href="#principles"><span /> Discover the experience</a>
      </section>

      <section className="principles" id="principles" aria-label="What makes MatchMate different">
        {principles.map((principle) => <article key={principle.value}><Icon name={principle.icon} /><div><strong>{principle.value}</strong><span>{principle.label}</span></div></article>)}
      </section>

      <section className="privacy section-shell" id="about">
        <div className="photo-panel">
          <img src="/images/matchmate-event.png" alt="Guests having a relaxed conversation at a curated rooftop event" />
          <div className="photo-note"><Icon name="shield" /><span><strong>Respect first</strong>Every event follows clear safety and consent boundaries.</span></div>
        </div>
        <div className="privacy-copy">
          <p className="section-kicker">Designed around trust</p>
          <h2>Privacy meets<br /><em>chemistry.</em></h2>
          <p className="section-lead">A dating experience should make you feel curious—not exposed. MatchMate keeps sensitive details private while helping compatible people meet in thoughtfully organized spaces.</p>
          <div className="safeguard-list">{safeguards.map((item) => <article key={item.title}><span className="feature-icon"><Icon name={item.icon} /></span><div><h3>{item.title}</h3><p>{item.text}</p></div></article>)}</div>
        </div>
      </section>

      <section className="how section-shell" id="how-it-works">
        <header className="section-heading"><div><p className="section-kicker">The MatchMate journey</p><h2>From profile to<br /><em>real connection.</em></h2></div><p>A guided path from thoughtful discovery to a safe, in-person introduction—without member-to-member chat.</p></header>
        <div className="steps-grid">{steps.map((step) => <article className="step-card" key={step.number}><span className="step-number">{step.number}</span><span className="step-icon"><Icon name={step.icon} /></span><h3>{step.title}</h3><p>{step.text}</p></article>)}</div>
      </section>

      <section className="matching section-shell" aria-labelledby="matching-title">
        <div className="matching-copy">
          <p className="section-kicker">No black box</p>
          <h2 id="matching-title">Compatibility you can <em>understand.</em></h2>
          <p>MatchMate does not use machine learning. Each event uses a versioned set of approved rules: hard eligibility filters, weighted compatibility components, deterministic tie-breaking, and organizer review.</p>
          <ul className="check-list"><li><Icon name="check" /> Private answers are never displayed in public profiles.</li><li><Icon name="check" /> Suggestions can be reproduced and explained safely.</li><li><Icon name="check" /> Blocks, restrictions, and booking status are rechecked.</li></ul>
        </div>
        <div className="score-card" aria-label="Illustration of explainable compatibility components">
          <div className="score-card-header"><span>Compatibility snapshot</span><span className="draft-pill">Illustrative</span></div>
          <div className="score-ring"><strong>Rules</strong><span>not predictions</span></div>
          <div className="score-bars"><div><span>Core preferences</span><i style={{ '--score': '92%' } as CSSProperties} /></div><div><span>Values & intentions</span><i style={{ '--score': '84%' } as CSSProperties} /></div><div><span>Shared interests</span><i style={{ '--score': '68%' } as CSSProperties} /></div></div>
          <p>Actual weights, questions, and explanation wording require approved product policy.</p>
        </div>
      </section>

      <section className="events section-shell" id="events">
        <div className="event-card"><div className="event-visual" aria-hidden="true"><span><Icon name="calendar" /></span></div><div className="event-copy"><p className="section-kicker">Curated in Sri Lanka</p><h2>Events are being thoughtfully prepared.</h2><p>Dates, venues, ticket prices, eligibility, cancellation rules, and safety policies will appear here only after organizer approval. We won’t invent availability before it is confirmed.</p><a className="inline-link" href="#join">Follow the launch <Icon name="arrow" /></a></div></div>
      </section>

      <section className="final-cta" id="join">
        <div className="cta-orbit" aria-hidden="true" /><p className="section-kicker">A more intentional way to meet</p><h2>Ready for something<br /><em>more real?</em></h2><p>MatchMate is currently in development. Registration and login will open when identity, privacy, booking, and safety workflows are ready.</p><span className="button button-muted" aria-disabled="true">Member access coming soon</span>
      </section>

      <footer>
        <a className="brand footer-brand" href="#top" aria-label="Back to the top"><span className="brand-mark" aria-hidden="true">M</span><span>MatchMate</span></a><p>Privacy-first blind-dating events for Sri Lanka.</p><nav aria-label="Footer navigation"><a href="#about">Privacy approach</a><a href="#how-it-works">How it works</a><a href="#events">Events</a></nav><small>© 2026 MatchMate. Product policies are under development.</small>
      </footer>
    </main>
  )
}

