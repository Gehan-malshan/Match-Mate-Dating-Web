import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useForm } from '@tanstack/react-form'
import { ApiProblem, login } from '../lib/account-api'

export function LoginPage() {
  const [message, setMessage] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const navigate = useNavigate()
  const form = useForm({
    defaultValues: { email: '', password: '' },
    onSubmit: async ({ value }) => {
      setMessage('')
      try {
        const { me } = await login(value.email, value.password)
        if (me.account.roles.includes('admin')) {
          await navigate({ to: '/admin' })
          return
        }
        await navigate({ to: '/app/profile' })
      } catch (error) {
        setMessage((error as ApiProblem).detail ?? 'Sign in failed. Please try again.')
      }
    },
  })

  return (
    <main className="login-page">
      <div className="login-atmosphere" aria-hidden="true" />
      <div className="login-portrait" aria-hidden="true" />

      <header className="login-header">
        <Link className="login-brand" to="/" aria-label="MatchMate home">
          <img src="/brand/matchmate-logo-nav.png" alt="" width="48" height="48" />
          <span>MatchMate</span>
        </Link>
        <nav aria-label="Login page navigation">
          <a href="/#about">The experience</a>
          <a href="/#trust">Privacy and safety</a>
        </nav>
      </header>

      <section className="login-stage" aria-labelledby="login-title">
        <div className="login-card">
          <div className="login-welcome">
            <p className="login-kicker">Member sign in</p>
            <h1 id="login-title">Welcome <em>back.</em></h1>
            <p>Return to your private MatchMate space and pick up where you left off.</p>
          </div>

          <form onSubmit={(event) => { event.preventDefault(); void form.handleSubmit() }}>
            <form.Field name="email">
              {(field) => (
                <label className="login-field">
                  <span>Email address</span>
                  <input autoComplete="email" type="email" placeholder="name@example.com" required value={field.state.value} onBlur={field.handleBlur} onChange={(event) => field.handleChange(event.target.value)} />
                </label>
              )}
            </form.Field>

            <form.Field name="password">
              {(field) => (
                <label className="login-field">
                  <span>Password</span>
                  <div className="password-control">
                    <input autoComplete="current-password" type={showPassword ? 'text' : 'password'} required value={field.state.value} onBlur={field.handleBlur} onChange={(event) => field.handleChange(event.target.value)} />
                    <button type="button" className="password-toggle" aria-label={showPassword ? 'Hide password' : 'Show password'} aria-pressed={showPassword} onClick={() => setShowPassword((visible) => !visible)}>
                      {showPassword ? 'Hide' : 'Show'}
                    </button>
                  </div>
                </label>
              )}
            </form.Field>

            <button className="login-submit" type="submit">Continue your journey <span aria-hidden="true">→</span></button>
          </form>

          {message && <p className="login-message" role="alert">{message}</p>}

          <div className="login-divider"><span>Private by design</span></div>
          <p className="login-privacy-note">Your email, date of birth, and matching preferences stay private.</p>
          <p className="login-register">New to MatchMate? <Link to="/register">Create your private profile</Link></p>
        </div>

        <footer className="login-footer">
          <p>Anonymity is a form of care.</p>
          <span>Privacy and terms are being prepared for launch.</span>
        </footer>
      </section>
    </main>
  )
}
