import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ApiProblem, CommunityProfile, listCommunityProfiles } from '../lib/account-api'
import { MemberNavigation } from '../components/MemberNavigation'
import './CommunityPage.css'

const initials = (nickname: string) => nickname.split(/\s+|_/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'M'
const tone = (profileId: string) => [...profileId].reduce((value, character) => value + character.charCodeAt(0), 0) % 4

export function CommunityPage() {
  const [profiles, setProfiles] = useState<CommunityProfile[]>([])
  const [cursor, setCursor] = useState('')
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState('')

  async function load(nextCursor = '') {
    setLoading(true)
    setMessage('')
    try {
      const page = await listCommunityProfiles(nextCursor)
      setProfiles((current) => nextCursor ? [...current, ...page.items] : page.items)
      setCursor(page.nextCursor ?? '')
    } catch (error) {
      const problem = error as ApiProblem
      setMessage(problem.code === 'AUTHENTICATION_REQUIRED' || problem.code === 'INVALID_ACCESS_TOKEN'
        ? 'Sign in to browse approved community profiles.'
        : problem.detail ?? 'Community profiles are temporarily unavailable.')
    } finally { setLoading(false) }
  }

  useEffect(() => { void load() }, [])

  return <main className="community-page">
    <MemberNavigation active="community" />
    <section className="community-shell" aria-labelledby="community-title">
      <header className="community-intro">
        <div><p className="community-kicker">The directory</p><h1 id="community-title">Discovery through <em>privacy.</em></h1><p>Meet approved members through what they choose to share. Identity, contact details, and private matching preferences remain protected.</p></div>
        <div className="community-standards" aria-label="Community discovery safeguards"><span>Approved profiles only</span><span>Block-aware discovery</span></div>
      </header>

      {message && <section className="community-state" role="alert"><h2>Community unavailable</h2><p>{message}</p><Link className="button button-ghost" to="/login">Member sign in</Link></section>}
      {loading && profiles.length === 0 && <section className="community-state" aria-live="polite"><div className="community-loader" aria-hidden="true" /><p>Opening the approved community directory…</p></section>}
      {!loading && !message && profiles.length === 0 && <section className="community-state"><h2>No community profiles yet</h2><p>Profiles appear only after members choose Community visibility and a moderator approves them.</p><Link className="button button-ghost" to="/app/profile">Review my profile</Link></section>}

      {profiles.length > 0 && <div className="community-grid" aria-label="Approved community profiles">
        {profiles.map((profile) => <article className="community-card" key={profile.profileId}>
          <div className={`community-avatar tone-${tone(profile.profileId)}`} aria-hidden="true"><span>{initials(profile.nickname)}</span></div>
          <p className="community-card-location">{profile.ageBand} · {profile.broadLocation || 'Broad location private'}</p>
          <h2>{profile.nickname}</h2>
          <p className="community-card-bio">{profile.bio || 'This member is keeping their introduction brief for now.'}</p>
          <div className="community-interests">{profile.interests.slice(0, 3).map((interest) => <span key={interest}>{interest}</span>)}{profile.interests.length === 0 && <span className="is-empty">Introduction first</span>}</div>
          <Link className="community-profile-link" to="/community/$profileId" params={{ profileId: profile.profileId }}>View profile <span aria-hidden="true">→</span></Link>
        </article>)}
      </div>}

      {cursor && <button className="community-more" type="button" disabled={loading} onClick={() => void load(cursor)}><span>{loading ? 'Seeking…' : 'Seek further'}</span><b aria-hidden="true">⌄</b></button>}
    </section>
  </main>
}
