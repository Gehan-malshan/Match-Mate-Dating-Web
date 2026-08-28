import { useEffect, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { ApiProblem, getMe, logout, Me, updatePreferences, updateProfile } from '../lib/account-api'
import './ProfilePage.css'

const splitList = (value: FormDataEntryValue | null) => String(value ?? '').split(',').map((item) => item.trim()).filter(Boolean)
const initials = (nickname: string) => nickname.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'M'
const ageBand = (dateOfBirth: string) => {
  const birth = new Date(`${dateOfBirth}T00:00:00`)
  const today = new Date()
  let age = today.getFullYear() - birth.getFullYear()
  if (today < new Date(today.getFullYear(), birth.getMonth(), birth.getDate())) age -= 1
  const start = Math.max(18, Math.floor(age / 5) * 5)
  return `${start}–${start + 4}`
}

export function ProfilePage() {
  const [me, setMe] = useState<Me>()
  const [message, setMessage] = useState('Loading your private profile…')
  const navigate = useNavigate()

  useEffect(() => {
    getMe().then((value) => { setMe(value); setMessage('') }).catch(() => void navigate({ to: '/login' }))
  }, [navigate])

  if (!me) return <main className="veil-loading"><div className="veil-loader" aria-hidden="true" /><p role="status">{message}</p></main>

  const profile = me.profile
  const preferences = me.preferences
  const communityEnabled = profile.visibility === 'COMMUNITY' && profile.approval === 'APPROVED'

  return <main className="veil-page">
    <header className="veil-header">
      <Link className="veil-brand" to="/" aria-label="MatchMate home"><img src="/brand/matchmate-logo-nav.png" alt="" /><span>MatchMate</span></Link>
      <nav aria-label="Member navigation">
        <Link to="/events">Events</Link>
        <Link to="/community">Community</Link>
        <a className="is-active" href="#profile-introduction" aria-current="page">Profile</a>
      </nav>
      <button className="veil-signout" type="button" onClick={async () => { await logout(); await navigate({ to: '/' }) }}>Sign out</button>
    </header>

    <section className="veil-shell" aria-labelledby="veil-title">
      <div className="veil-intro">
        <p className="eyebrow">Private profile workspace</p>
        <h1 id="veil-title">My <em>veil.</em></h1>
        <p>Shape the introduction members may eventually see while keeping your identity and matching choices protected.</p>
      </div>

      {message && <p className="veil-message" role="status">{message}</p>}

      <div className="veil-grid">
        <aside className="veil-side">
          <section className="veil-card veil-preview" aria-labelledby="preview-title">
            <div className="veil-avatar" aria-hidden="true"><span>{initials(profile.nickname)}</span></div>
            <p className="veil-overline">Community preview</p>
            <h2 id="preview-title">{profile.nickname}</h2>
            <p className="veil-preview-meta">{ageBand(profile.dateOfBirth)} · {profile.broadLocation || 'Location stays private'}</p>
            <p className="veil-preview-bio">{profile.bio || 'Your approved introduction will appear here when you add a bio.'}</p>
            <div className="veil-chips">{profile.interests.length > 0 ? profile.interests.map((interest) => <span key={interest}>{interest}</span>) : <span className="is-empty">No interests added yet</span>}</div>
            <div className="veil-preview-state">
              <span className={communityEnabled ? 'is-ready' : ''}>{communityEnabled ? 'Visible to community' : 'Private preview only'}</span>
              <small>{profile.approval.toLowerCase()} moderation status</small>
            </div>
          </section>

          <section className="veil-card veil-vault">
            <p className="veil-overline">Privacy vault</p>
            <h2>Your private details stay behind the veil.</h2>
            <dl>
              <div><dt>Account</dt><dd>{me.account.verification.toLowerCase()}</dd></div>
              <div><dt>Profile review</dt><dd>{profile.approval.toLowerCase()}</dd></div>
              <div><dt>Visibility</dt><dd>{profile.visibility.toLowerCase()}</dd></div>
            </dl>
            <p>Email, exact birth date, matching preferences, and deal-breakers are never included in community profiles.</p>
          </section>
        </aside>

        <div className="veil-main">
          <form id="profile-introduction" className="veil-card veil-form" onSubmit={async (event) => {
            event.preventDefault()
            const data = new FormData(event.currentTarget)
            setMessage('Saving your profile…')
            try {
              const updated = await updateProfile({
                nickname: data.get('nickname'),
                broadLocation: data.get('location'),
                bio: data.get('bio'),
                visibility: data.get('visibility'),
                interests: splitList(data.get('interests')),
                expectedVersion: profile.version,
              })
              setMe({ ...me, profile: updated })
              setMessage('Your profile was saved.')
            } catch (error) { setMessage((error as ApiProblem).detail ?? 'Profile update failed.') }
          }}>
            <div className="veil-form-heading"><div><p className="veil-overline">The introduction</p><h2>How you appear</h2></div><span>Community-safe fields</span></div>
            <p className="veil-form-copy">Only the approved fields in this section can become visible after you select Community and moderation approves the profile.</p>
            <div className="veil-field-grid">
              <label><span>Nickname</span><input name="nickname" minLength={2} maxLength={40} defaultValue={profile.nickname} required /></label>
              <label><span>Broad location</span><input name="location" maxLength={120} defaultValue={profile.broadLocation} placeholder="Colombo" /></label>
              <label className="is-wide"><span>Your introduction</span><textarea name="bio" maxLength={1000} defaultValue={profile.bio} placeholder="Share enough to start a meaningful conversation without adding contact details." /></label>
              <label className="is-wide"><span>Interests, separated by commas</span><input name="interests" defaultValue={profile.interests.join(', ')} placeholder="Live music, hiking, cooking" /></label>
              <label className="is-wide"><span>Profile visibility</span><select name="visibility" defaultValue={profile.visibility === 'COMMUNITY' ? 'COMMUNITY' : 'PRIVATE'}><option value="PRIVATE">Private — only you can view it</option><option value="COMMUNITY">Community — after moderation approval</option></select></label>
            </div>
            <div className="veil-form-footer"><p>Contact details and social handles are not allowed in public profile text.</p><button className="button" type="submit">Save introduction</button></div>
          </form>

          <form className="veil-card veil-form veil-preferences" onSubmit={async (event) => {
            event.preventDefault()
            const data = new FormData(event.currentTarget)
            setMessage('Saving private matching preferences…')
            try {
              const updated = await updatePreferences({
                minAge: Number(data.get('minAge')),
                maxAge: Number(data.get('maxAge')),
                intentions: splitList(data.get('intentions')),
                interestedIn: splitList(data.get('interestedIn')),
                languages: splitList(data.get('languages')),
                dealBreakers: splitList(data.get('dealBreakers')),
              })
              setMe({ ...me, preferences: updated })
              setMessage('Your private matching preferences were saved.')
            } catch (error) { setMessage((error as ApiProblem).detail ?? 'Preference update failed.') }
          }}>
            <div className="veil-form-heading"><div><p className="veil-overline">Private matching blueprint</p><h2>Your matching boundaries</h2></div><span className="is-private">Never public</span></div>
            <p className="veil-form-copy">These answers are used only by MatchMate’s deterministic matching rules. Other members cannot view them.</p>
            <div className="veil-field-grid">
              <label><span>Minimum age</span><input name="minAge" type="number" min={18} max={120} defaultValue={preferences?.minAge ?? 25} required /></label>
              <label><span>Maximum age</span><input name="maxAge" type="number" min={18} max={120} defaultValue={preferences?.maxAge ?? 40} required /></label>
              <label className="is-wide"><span>Connection intentions</span><input name="intentions" defaultValue={preferences?.intentions.join(', ')} placeholder="Long-term relationship" /></label>
              <label><span>Interested in</span><input name="interestedIn" defaultValue={preferences?.interestedIn.join(', ')} placeholder="Project-approved values" /></label>
              <label><span>Languages</span><input name="languages" defaultValue={preferences?.languages.join(', ')} placeholder="Sinhala, Tamil, English" /></label>
              <label className="is-wide"><span>Deal-breakers</span><textarea name="dealBreakers" defaultValue={preferences?.dealBreakers.join(', ')} placeholder="Private boundaries, separated by commas" /></label>
            </div>
            <div className="veil-form-footer"><p>Private preferences are not part of your community profile.</p><button className="button" type="submit">Save preferences</button></div>
          </form>
        </div>
      </div>
    </section>

    <nav className="veil-mobile-nav" aria-label="Mobile member navigation">
      <Link to="/events"><span aria-hidden="true">◇</span>Events</Link>
      <Link to="/community"><span aria-hidden="true">○</span>Community</Link>
      <a className="is-active" href="#profile-introduction"><span aria-hidden="true">●</span>Profile</a>
    </nav>
  </main>
}
