import { useRef, useState, type ReactNode } from 'react'
import { useGSAP } from '@gsap/react'
import gsap from 'gsap'
import { ScrollTrigger } from 'gsap/ScrollTrigger'

gsap.registerPlugin(useGSAP, ScrollTrigger)

type IconName =
  | 'arrow'
  | 'calendar'
  | 'check'
  | 'heart'
  | 'lock'
  | 'profile'
  | 'shield'
  | 'spark'
  | 'ticket'
  | 'users'

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
  users: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 1-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/></>,
}

function Icon({ name }: { name: IconName }) {
  return (
    <svg aria-hidden="true" className="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      {paths[name]}
    </svg>
  )
}

const journey = [
  {
    icon: 'profile' as const,
    title: 'Begin privately',
    lead: 'Create a verified account and decide what is visible.',
    detail: 'Your legal identity, contact details, and private preferences never become community profile fields. A preview shows the exact approved public view.',
  },
  {
    icon: 'calendar' as const,
    title: 'Choose the room',
    lead: 'Discover a curated event with clear policies.',
    detail: 'Dates, broad location, format, eligibility, price, and safety rules appear only after organizer approval. No invented inventory or vague promises.',
  },
  {
    icon: 'ticket' as const,
    title: 'Confirm your place',
    lead: 'A server-confirmed booking unlocks eligibility.',
    detail: 'Capacity is reserved before payment, price and currency are snapshotted by the server, and PayHere confirmation comes from verified backend state.',
  },
  {
    icon: 'users' as const,
    title: 'Meet with context',
    lead: 'Understand the suggestion, then meet in person.',
    detail: 'Versioned rules create reproducible pairings. Organizers review the run, and any later identity reveal still requires approved policy and explicit consent.',
  },
]

const revealSentence = 'No black box. No inferred traits. Just approved rules you can understand, reviewed before anyone meets.'

export function LandingPage() {
  const pageRef = useRef<HTMLElement>(null)
  const [activeStep, setActiveStep] = useState(0)

  useGSAP(() => {
    if (typeof window.matchMedia !== 'function' || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

    gsap.from('.hero-content > *', {
      opacity: 0,
      y: 28,
      duration: 0.9,
      stagger: 0.1,
      ease: 'power3.out',
    })

    gsap.fromTo('.hero-backdrop',
      { scale: 1.06, opacity: 0.72 },
      { scale: 1, opacity: 1, duration: 1.35, ease: 'power3.out' },
    )

    gsap.from('.trust-tile', {
      scrollTrigger: { trigger: '.trust-grid', start: 'top 78%' },
      opacity: 0,
      y: 42,
      duration: 0.8,
      stagger: 0.09,
      ease: 'power3.out',
    })

    gsap.fromTo('.reveal-word',
      { opacity: 0.12 },
      {
        opacity: 1,
        stagger: 0.06,
        scrollTrigger: {
          trigger: '.reveal-statement',
          start: 'top 78%',
          end: 'bottom 48%',
          scrub: 0.7,
        },
      },
    )

    gsap.fromTo('.event-image',
      { scale: 0.82, opacity: 0.3 },
      {
        scale: 1,
        opacity: 1,
        ease: 'none',
        scrollTrigger: {
          trigger: '.event-stage',
          start: 'top bottom',
          end: 'center center',
          scrub: 0.8,
        },
      },
    )
  }, { scope: pageRef })

  return (
    <main className="landing-page" ref={pageRef}>
      <header className="site-header">
        <a className="brand" href="#top" aria-label="MatchMate home">
          <img className="brand-mark" src="/brand/matchmate-logo-nav.png" alt="" width="44" height="44" />
          <span>MatchMate</span>
        </a>
        <nav aria-label="Primary navigation">
          <a href="#trust">Why MatchMate</a>
          <a href="#how-it-works">The journey</a>
          <a href="#events">Events</a>
        </nav>
        <a className="button button-small" href="/register">Create your profile</a>
      </header>

      <section className="hero" id="top">
        <div className="hero-backdrop" aria-hidden="true" />
        <div className="hero-content">
          <p className="eyebrow"><span /> Sri Lanka's privacy-first dating events</p>
          <h1>Where mystery<br />meets <em>connection.</em></h1>
          <p className="hero-copy">Meet with intention through verified profiles, curated gatherings, and transparent compatibility rules - without endless swiping or chat.</p>
          <div className="hero-actions">
            <a className="button" href="#events">Explore events <Icon name="arrow" /></a>
            <a className="button button-ghost" href="#how-it-works">See how it works</a>
          </div>
        </div>
        <div className="trust-card" aria-label="MatchMate privacy principle"><span className="trust-dot" /><div><strong>Private by design</strong><small>Your sensitive preferences stay private.</small></div></div>
        <a className="scroll-cue" href="#trust"><span /> Discover the experience</a>
      </section>

      <section className="principle-marquee" aria-label="MatchMate principles">
        <div className="marquee-track">
          <span>Verified community</span><i />
          <span>No member chat</span><i />
          <span>Explainable matching</span><i />
          <span>Curated real-world events</span><i />
          <span>Consent before reveal</span><i />
          <span aria-hidden="true">Verified community</span><i aria-hidden="true" />
          <span aria-hidden="true">No member chat</span><i aria-hidden="true" />
          <span aria-hidden="true">Explainable matching</span><i aria-hidden="true" />
        </div>
      </section>

      <section className="trust section-shell" id="trust">
        <header className="section-heading section-heading-wide">
          <h2>Keep the mystery.<br /><em>Lose the uncertainty.</em></h2>
          <p>MatchMate is designed around one idea: meeting someone new can feel electric without asking you to surrender privacy, context, or control.</p>
        </header>

        <div className="trust-grid">
          <article className="trust-tile trust-visual">
            <img src="/images/matchmate-event.png" alt="Guests sharing a respectful conversation at an organized event" />
            <div><Icon name="shield" /><span><strong>Respect is structural</strong>Blocking, reporting, moderation, and organizer review are part of the experience.</span></div>
          </article>
          <article className="trust-tile trust-manifesto">
            <span className="feature-icon"><Icon name="lock" /></span>
            <h3>A profile is an invitation, not an exposure.</h3>
            <p>Only explicitly approved community fields can appear. Legal identity, contact details, verification evidence, and matching inputs stay restricted.</p>
          </article>
          <article className="trust-tile trust-compact">
            <Icon name="spark" />
            <h3>Rules, not predictions</h3>
            <p>Deterministic and reproducible. No machine-learning model and no inferred sensitive traits.</p>
          </article>
          <article className="trust-tile trust-compact trust-accent">
            <Icon name="users" />
            <h3>Meet in the room</h3>
            <p>No member-to-member chat. The product is designed to move toward safe real-world interaction.</p>
          </article>
          <article className="trust-tile trust-compact">
            <Icon name="heart" />
            <h3>Reveal requires consent</h3>
            <p>Interest is structured and private. Identity or contact reveal requires approved policy and explicit consent.</p>
          </article>
        </div>
      </section>

      <section className="journey section-shell" id="how-it-works">
        <header className="section-heading journey-heading">
          <h2>Four moves.<br /><em>One real meeting.</em></h2>
          <p>Choose each chapter to see how MatchMate protects the path from first profile to event-day introduction.</p>
        </header>

        <div className="journey-accordion" role="group" aria-label="MatchMate journey steps">
          {journey.map((step, index) => {
            const isActive = activeStep === index
            return (
              <article className={isActive ? 'journey-step is-active' : 'journey-step'} key={step.title}>
                <button type="button" aria-expanded={isActive} onClick={() => setActiveStep(index)} onFocus={() => setActiveStep(index)}>
                  <span className="journey-index">{String(index + 1).padStart(2, '0')}</span>
                  <span className="journey-icon"><Icon name={step.icon} /></span>
                  <span className="journey-title">{step.title}</span>
                  <span className="journey-lead">{step.lead}</span>
                </button>
                <div className="journey-detail" aria-hidden={!isActive}><p>{step.detail}</p></div>
              </article>
            )
          })}
        </div>
      </section>

      <section className="matching section-shell" aria-labelledby="matching-title">
        <p className="reveal-statement" id="matching-title">
          {revealSentence.split(' ').map((word, index) => <span className="reveal-word" key={`${word}-${index}`}>{word}{' '}</span>)}
        </p>
        <div className="matching-proof">
          <div><span className="feature-icon"><Icon name="check" /></span><h3>Hard filters first</h3><p>Booking status, event eligibility, blocks, restrictions, and approved preferences determine who can enter a run.</p></div>
          <div><span className="feature-icon"><Icon name="spark" /></span><h3>Versioned scoring</h3><p>Approved components and deterministic tie-breaking make each suggestion reproducible without exposing private answers.</p></div>
          <div><span className="feature-icon"><Icon name="shield" /></span><h3>Human review</h3><p>Organizers can review and override suggestions, but every override requires a reason and remains auditable.</p></div>
        </div>
      </section>

      <section className="event-stage" id="events">
        <img className="event-image" src="/images/matchmate-event.png" alt="A warmly lit rooftop prepared for an organized MatchMate gathering" />
        <div className="event-wash" aria-hidden="true" />
        <div className="event-content section-shell">
          <span className="event-icon"><Icon name="calendar" /></span>
          <h2>The room is being<br /><em>prepared with care.</em></h2>
          <p>Dates, venues, ticket prices, eligibility, cancellation rules, and safety policies will appear only after organizer approval. We will not invent availability before it is confirmed.</p>
          <a className="inline-link" href="#join">Follow the launch <Icon name="arrow" /></a>
        </div>
      </section>

      <section className="final-cta" id="join">
        <img className="cta-mark" src="/brand/matchmate-logo-mark.png" alt="" aria-hidden="true" />
        <p>Registration opens after identity, booking, privacy, and safety workflows are ready.</p>
        <h2>Something more real<br />is worth <em>waiting for.</em></h2>
        <a className="button" href="/register">Create a private profile</a>
      </section>

      <footer>
        <a className="brand footer-brand" href="#top" aria-label="Back to the top">
          <img className="brand-mark" src="/brand/matchmate-logo-nav.png" alt="" width="44" height="44" />
          <span>MatchMate</span>
        </a>
        <p>Privacy-first blind-dating events for Sri Lanka.</p>
        <nav aria-label="Footer navigation"><a href="#trust">Why MatchMate</a><a href="#how-it-works">The journey</a><a href="#events">Events</a></nav>
        <small>© 2026 MatchMate. Product policies are under development.</small>
      </footer>
    </main>
  )
}
