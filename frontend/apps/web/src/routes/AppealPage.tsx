import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { appealModerationAction, logout, myModerationActions } from '../lib/account-api'

export function AppealPage() {
  const navigate = useNavigate()
  const actions = useQuery({ queryKey: ['my-moderation-actions'], queryFn: myModerationActions })
  const [reason, setReason] = useState('')
  const [chosen, setChosen] = useState('')
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)
  return <main className="login-page">
    <header className="login-header"><Link className="login-brand" to="/">MatchMate</Link></header>
    <section className="login-stage" aria-labelledby="appeal-title"><div className="login-card">
      <div className="login-welcome"><p className="login-kicker">Account safety</p><h1 id="appeal-title">Request a review</h1><p>Your account has limited access. You can appeal an active action here. Your appeal is private to the moderation team.</p></div>
      {actions.isPending && <p role="status">Loading active actions…</p>}
      {actions.isError && <p role="alert">We could not load your actions. Please try again later.</p>}
      {actions.data && actions.data.length === 0 && <p role="status">No active actions are available to appeal. Sign in again if access has been restored.</p>}
      {actions.data && actions.data.length > 0 && <form onSubmit={async event => {
        event.preventDefault()
        setMessage('')
        setSubmitting(true)
        try {
          await appealModerationAction(chosen, reason.trim())
          setMessage('Your appeal was submitted for review. You can sign in again after a decision.')
          setReason('')
        } catch (error) {
          setMessage(error instanceof Error ? error.message : 'We could not submit your appeal.')
        } finally { setSubmitting(false) }
      }}>
        <label className="login-field"><span>Action to appeal</span><select required value={chosen} onChange={event => setChosen(event.target.value)}><option value="">Choose an action</option>{actions.data.map(action => <option key={action.actionId} value={action.actionId}>{action.actionClass.replaceAll('_', ' ').toLowerCase()} · {action.scope === 'ALL' ? 'all events' : 'event'} · {new Date(action.effectiveAt).toLocaleDateString()}</option>)}</select></label>
        <label className="login-field"><span>Why should we review this action?</span><textarea required minLength={10} maxLength={1000} value={reason} onChange={event => setReason(event.target.value)} /></label>
        <button className="login-submit" type="submit" disabled={submitting}>{submitting ? 'Submitting…' : 'Submit appeal'}</button>
      </form>}
      {message && <p className="login-message" role="status">{message}</p>}
      <button className="login-google" type="button" onClick={async () => { await logout(); await navigate({ to: '/login' }) }}>Sign out</button>
    </div></section>
  </main>
}
