import { useRef, type ReactNode } from 'react'
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
    title: 'Create your private profile',
    detail: 'Share your values, lifestyle, interests, and relationship goals.',
  },
  {
    icon: 'calendar' as const,
    title: 'Reserve an event place',
    detail: 'Choose an approved event and confirm your booking securely.',
  },
  {
    icon: 'spark' as const,
    title: 'Discover compatible introductions',
    detail: 'Clear rules compare preferences, deal-breakers, and shared values.',
  },
  {
    icon: 'users' as const,
    title: 'Meet in guided conversations',
    detail: 'Meet suitable people through a comfortable, organized event.',
  },
  {
    icon: 'heart' as const,
    title: 'Connect by mutual choice',
    detail: 'Contact details stay private unless both people choose to continue.',
  },
]

const revealSentence = 'Clear rules. Better introductions. Nothing hidden.'

export function LandingPage() {
  const pageRef = useRef<HTMLElement>(null)

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
          <a href="#about">About</a>
          <a href="#how-it-works">How it works</a>
          <a href="#events">Events</a>
        </nav>
        <a className="button button-small" href="/register">Create your profile</a>
      </header>

      <section className="hero" id="top">
        <div className="hero-backdrop" aria-hidden="true" />
        <div className="hero-content">
          <p className="eyebrow"><span /> Sri Lanka's privacy-first dating events</p>
          <h1>Less swiping.<br />More <em>meaningful meetings.</em></h1>
          <p className="hero-copy">Meet compatible people through thoughtfully organized real-world events.</p>
          <div className="hero-actions">
            <a className="button" href="#events">Find your next match <Icon name="arrow" /></a>
            <a className="button button-ghost" href="#how-it-works">See how it works</a>
          </div>
        </div>
        <a className="scroll-cue" href="#trust"><span /> Discover the experience</a>
      </section>

      <section className="about section-shell" id="about" aria-labelledby="about-title">
        <div className="about-intro">
          <h2 id="about-title">A more human path from <em>compatibility to conversation.</em></h2>
          <p className="about-lead">MatchMate turns private compatibility into safe, face-to-face introductions.</p>
        </div>
        <figure className="about-visual">
          <img src="/images/matchmate-about-cafe.png" alt="Two young professionals sharing a comfortable conversation in a refined cafe" />
          <figcaption><strong>No member-to-member chat.</strong><span>Meet through safe, organized events.</span></figcaption>
        </figure>
      </section>

      <section className="trust section-shell" id="trust">
        <header className="section-heading section-heading-wide">
          <h2>Keep the mystery.<br /><em>Lose the uncertainty.</em></h2>
          <p>MatchMate is designed around one idea: meeting someone new can feel electric without asking you to surrender privacy, context, or control.</p>
        </header>

        <div className="trust-grid trust-grid-simple">
          <article className="trust-tile trust-visual">
            <img src="/images/matchmate-event.png" alt="Guests sharing a respectful conversation at an organized event" />
            <div><Icon name="shield" /><span><strong>Safe, organized meetings</strong>Guided events with coordinators, reporting, and support.</span></div>
          </article>
          <aside className="trust-principles" aria-label="MatchMate privacy principles">
            <div>
              <span className="trust-principle-icon"><Icon name="lock" /></span>
              <span className="trust-principle-number">01</span>
              <h3>Your private details stay private.</h3>
              <p>Only approved profile information can be seen by others.</p>
            </div>
            <div>
              <span className="trust-principle-icon"><Icon name="spark" /></span>
              <span className="trust-principle-number">02</span>
              <h3>Clear matching. Mutual choice.</h3>
              <p>No black box and no contact reveal without consent.</p>
            </div>
          </aside>
        </div>
      </section>

      <section className="journey section-shell" id="how-it-works">
        <header className="section-heading journey-heading">
          <h2>From your profile to a <em>real introduction.</em></h2>
          <p>Five clear steps. No endless swiping and no pressure to share contact details.</p>
        </header>

        <div className="journey-story">
          <div className="journey-visual">
            <img src="/images/matchmate-event-checkin.png" alt="Young professionals arriving and checking in at an organized MatchMate event" />
            <span>Profile <Icon name="arrow" /> Event <Icon name="arrow" /> Real meeting</span>
          </div>
          <ol className="journey-list" aria-label="How MatchMate works">
            {journey.map((step, index) => (
              <li key={step.title}>
                <span className="journey-index">{String(index + 1).padStart(2, '0')}</span>
                <span className="journey-icon"><Icon name={step.icon} /></span>
                <div><h3>{step.title}</h3><p>{step.detail}</p></div>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section className="matching section-shell" aria-labelledby="matching-title">
        <div className="matching-layout">
          <div className="matching-copy">
            <p className="reveal-statement" id="matching-title">
              {revealSentence.split(' ').map((word, index) => <span className="reveal-word" key={`${word}-${index}`}>{word}{' '}</span>)}
            </p>
            <p>Approved, repeatable rules—not inferred traits or machine learning.</p>
          </div>
          <div className="rules-orbit" aria-label="Preferences move through approved rules and human review">
            <span className="orbit-ring orbit-ring-outer" aria-hidden="true" />
            <span className="orbit-ring orbit-ring-inner" aria-hidden="true" />
            <div className="orbit-core"><img src="/brand/matchmate-logo-nav.png" alt="" /><strong>MatchMate</strong><small>Explainable matching</small></div>
            <span className="orbit-node orbit-preferences"><Icon name="check" /> Preferences</span>
            <span className="orbit-node orbit-rules"><Icon name="spark" /> Rules</span>
            <span className="orbit-node orbit-review"><Icon name="shield" /> Review</span>
          </div>
        </div>
        <div className="matching-proof">
          <div><span className="matching-number">01</span><h3>Preferences first</h3><p>Eligibility, deal-breakers, and boundaries are respected.</p></div>
          <div><span className="matching-number">02</span><h3>Explainable results</h3><p>Clear, repeatable rules guide every introduction.</p></div>
          <div><span className="matching-number">03</span><h3>Human oversight</h3><p>Organizer decisions are reviewed and recorded.</p></div>
        </div>
      </section>

      <section className="event-stage" id="events">
        <img className="event-image" src="/images/matchmate-rooftop-event.png" alt="Young professionals having guided conversations at a warmly lit rooftop event" />
        <div className="event-wash" aria-hidden="true" />
        <div className="event-content section-shell">
          <span className="event-icon"><Icon name="calendar" /></span>
          <h2>Meet compatible people<br /><em>in the real world.</em></h2>
          <p>MatchMate events bring compatible introductions together in comfortable Colombo venues. Dates, prices, eligibility, and safety details are published after every event is confirmed.</p>
          <a className="button event-action" href="#event-updates">
            View event announcements <Icon name="arrow" />
          </a>
        </div>
      </section>

      <footer className="site-footer">
        <div className="footer-content">
          <div className="footer-intro">
            <a className="brand footer-brand" href="#top" aria-label="Back to the top">
              <img className="brand-mark" src="/brand/matchmate-logo-nav.png" alt="" width="44" height="44" />
              <span>MatchMate</span>
            </a>
            <p>Privacy-first matchmaking that leads to safe, real-world conversations.</p>
          </div>
          <nav aria-label="Footer navigation">
            <strong>Explore</strong>
            <a href="#about">About MatchMate</a>
            <a href="#how-it-works">How it works</a>
            <a href="#events">Events</a>
          </nav>
          <nav aria-label="Trust and safety navigation">
            <strong>Trust</strong>
            <a href="#trust">Privacy and safety</a>
            <a href="#matching-title">How matching works</a>
          </nav>
          <div className="footer-status" id="event-updates">
            <strong>Event updates</strong>
            <span><i /> Colombo event announcements</span>
            <p>New dates and booking details will be published here after confirmation.</p>
          </div>
        </div>
        <div className="footer-bottom">
          <small>© 2026 MatchMate. All rights reserved.</small>
          <a href="#top">Back to top <Icon name="arrow" /></a>
        </div>
      </footer>
    </main>
  )
}
