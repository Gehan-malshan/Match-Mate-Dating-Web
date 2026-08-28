import { Link } from '@tanstack/react-router'
import './MemberNavigation.css'

type Section = 'events' | 'community' | 'profile'

export function MemberNavigation({ active }: { active: Section }) {
  const item = (section: Section, to: '/events' | '/community' | '/app/profile', label: string) => (
    <Link className={active === section ? 'is-active' : ''} to={to} aria-current={active === section ? 'page' : undefined}>{label}</Link>
  )

  return <>
    <header className="member-header">
      <Link className="member-brand" to="/" aria-label="MatchMate home"><img src="/brand/matchmate-logo-nav.png" alt="" /><span>MatchMate</span></Link>
      <nav aria-label="Member navigation">{item('events', '/events', 'Events')}{item('community', '/community', 'Community')}{item('profile', '/app/profile', 'Profile')}</nav>
      <Link className="member-account" to="/app/profile" aria-label="Open my profile"><span aria-hidden="true">M</span></Link>
    </header>
    <nav className="member-mobile-nav" aria-label="Mobile member navigation">
      <Link className={active === 'events' ? 'is-active' : ''} to="/events"><span aria-hidden="true">◇</span>Events</Link>
      <Link className={active === 'community' ? 'is-active' : ''} to="/community"><span aria-hidden="true">●</span>Community</Link>
      <Link className={active === 'profile' ? 'is-active' : ''} to="/app/profile"><span aria-hidden="true">○</span>Profile</Link>
    </nav>
  </>
}
