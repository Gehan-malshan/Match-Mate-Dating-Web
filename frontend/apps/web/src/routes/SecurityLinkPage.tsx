import {useState} from 'react'
import {Link} from '@tanstack/react-router'
import {requestPasswordReset,resendVerification,resetPassword,verifyEmail} from '../lib/account-api'

export function SecurityLinkPage({mode}:{mode:'verify'|'forgot'|'reset'}) {
  const [message,setMessage]=useState('')
  const [token]=useState(()=>{const value=new URLSearchParams(window.location.search).get('token')||'';if(value)window.history.replaceState(window.history.state,'',window.location.pathname);return value})
  return <main className="login-page"><section className="login-stage"><div className="login-card">
    <Link className="login-brand" to="/">MatchMate</Link>
    <h1>{mode==='verify'?'Verify your email':mode==='forgot'?'Recover your account':'Choose a new password'}</h1>
    <form onSubmit={async event=>{event.preventDefault();const data=new FormData(event.currentTarget);try{
      if(mode==='verify'){if(token)await verifyEmail(token);else await resendVerification(String(data.get('email')));setMessage(token?'Email verified. You can sign in.':'If this account needs verification, a new link will arrive shortly.')}
      else if(mode==='forgot'){await requestPasswordReset(String(data.get('email')));setMessage('If an eligible account exists, a recovery link will arrive shortly.')}
      else{await resetPassword(token,String(data.get('password')));setMessage('Password changed. Sign in again with your new password.')}
    }catch{setMessage('This request could not be completed. Check the link or try again shortly.')}}}>
      {((mode==='verify'&&!token)||mode==='forgot')&&<label className="login-field">Email address<input name="email" type="email" autoComplete="email" required /></label>}
      {mode==='reset'&&<label className="login-field">New password<input name="password" type="password" autoComplete="new-password" minLength={12} maxLength={128} required /></label>}
      <button className="login-submit" type="submit" disabled={mode==='reset'&&!token}>{mode==='verify'&&token?'Verify email':mode==='verify'?'Send new link':mode==='forgot'?'Send recovery link':'Reset password'}</button>
    </form>
    {message&&<p role="status" className="login-message">{message}</p>}
    <p><Link to="/login">Back to sign in</Link></p>
  </div></section></main>
}
