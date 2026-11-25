import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { ApiProblem, blockMember, CommunityProfile, getCommunityProfile } from '../lib/account-api'
import { MemberNavigation } from '../components/MemberNavigation'
import './CommunityPage.css'

const initials = (nickname: string) => nickname.split(/\s+|_/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'M'

export function CommunityProfilePage() {
  const { profileId } = useParams({ from: '/member-access/community/$profileId' })
  const navigate = useNavigate()
  const [profile, setProfile] = useState<CommunityProfile>()
  const [message, setMessage] = useState('Loading this community profile…')
  const [failed, setFailed] = useState(false)
  const [confirmingBlock, setConfirmingBlock] = useState(false)

  useEffect(() => {
    getCommunityProfile(profileId).then((value) => { setProfile(value); setMessage(''); setFailed(false) }).catch((error: ApiProblem) => { setMessage(error.detail ?? 'This profile is not available.'); setFailed(true) })
  }, [profileId])

  return <main className="community-page">
    <MemberNavigation active="community" />
    <section className="community-detail-shell">
      <Link className="community-back" to="/community">← Back to community</Link>
      {!profile ? <section className="community-state" role={failed ? 'alert' : 'status'}>{!failed && <div className="community-loader" aria-hidden="true" />}<p>{message}</p>{failed && <Link className="button button-ghost" to="/community">Return to community</Link>}</section> : <article className="community-detail">
        <aside>
          <div className="community-detail-avatar" aria-hidden="true"><span>{initials(profile.nickname)}</span></div>
          <p className="community-kicker">Approved community profile</p>
          <h1>{profile.nickname}</h1>
          <p className="community-detail-meta">{profile.ageBand} · {profile.broadLocation || 'Broad location private'}</p>
          <div className="community-detail-interests">{profile.interests.map((interest) => <span key={interest}>{interest}</span>)}</div>
        </aside>
        <section>
          <p className="community-kicker">Their introduction</p>
          <h2>A little of what they chose to share.</h2>
          <p className="community-detail-bio">{profile.bio || 'This member has not added a longer introduction yet.'}</p>
          <div className="community-privacy-note"><strong>Privacy by design</strong><p>This profile never exposes email, exact birth date, exact address, private preferences, deal-breakers, or moderation information. MatchMate does not provide member-to-member chat.</p></div>
          <div className="community-safety-actions">
            {!confirmingBlock ? <button type="button" onClick={() => setConfirmingBlock(true)}>Block this profile</button> : <div role="alert"><p>Blocking removes both members from each other’s discovery results and future matching eligibility.</p><button className="is-danger" type="button" onClick={async () => { try { await blockMember(profile.profileId); await navigate({ to: '/community' }) } catch (error) { setMessage((error as ApiProblem).detail ?? 'Unable to block this profile.') } }}>Confirm block</button><button type="button" onClick={() => setConfirmingBlock(false)}>Cancel</button></div>}
          </div>
          {message && <p className="community-detail-message" role="status">{message}</p>}
        </section>
      </article>}
    </section>
  </main>
}
