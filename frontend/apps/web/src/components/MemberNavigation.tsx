import { Link } from '@tanstack/react-router'
import {NotificationCenter} from './NotificationCenter'
import './MemberNavigation.css'

type Section = 'events' | 'community' | 'bookings' | 'notifications' | 'profile'

export function MemberNavigation({ active }: { active: Section }) {
  const item = (section: Section, to: '/events' | '/community' | '/app/bookings' | '/app/notifications' | '/app/profile', label: string) => (
    <Link className={active === section ? 'is-active' : ''} to={to} aria-current={active === section ? 'page' : undefined}>{label}</Link>
  )

  return <>
    <header className="member-header">
      <Link className="member-brand" to="/" aria-label="MatchMate home"><img src="/brand/matchmate-logo-nav.png" alt="" /><span>MatchMate</span></Link>
      <nav aria-label="Member navigation">{item('events', '/events', 'Events')}{item('community', '/community', 'Community')}{item('bookings', '/app/bookings', 'Bookings')}{item('profile', '/app/profile', 'Profile')}</nav>
      <div className="member-header-actions"><NotificationCenter/><Link className="member-account" to="/app/profile" aria-label="Open my profile"><span aria-hidden="true">M</span></Link></div>
    </header>
    <nav className="member-mobile-nav" aria-label="Mobile member navigation">
      <Link className={active === 'bookings' ? 'is-active' : ''} to="/app/bookings"><span aria-hidden="true">◇</span>Bookings</Link>
      <Link className={active === 'events' ? 'is-active' : ''} to="/events"><span aria-hidden="true">◇</span>Events</Link>
      <Link className={active === 'community' ? 'is-active' : ''} to="/community"><span aria-hidden="true">●</span>Community</Link>
      <Link className={active === 'notifications' ? 'is-active' : ''} to="/app/notifications"><span aria-hidden="true">♢</span>Updates</Link>
      <Link className={active === 'profile' ? 'is-active' : ''} to="/app/profile"><span aria-hidden="true">○</span>Profile</Link>
    </nav>
  </>
}
