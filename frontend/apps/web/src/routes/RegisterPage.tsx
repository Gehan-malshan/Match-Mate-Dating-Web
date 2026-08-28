import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { ApiProblem, register, verifyEmail } from '../lib/account-api'

const schema = z.object({ email: z.email(), password: z.string().min(12), nickname: z.string().min(2).max(40), dateOfBirth: z.string().min(10), consent: z.boolean().refine(Boolean) })
const steps = ['Identity', 'The veil', 'Visibility', 'Vow']
type FormShape = { email: string; password: string; nickname: string; dateOfBirth: string; consent: boolean }

export function RegisterPage() {
  const [step, setStep] = useState(0)
  const [message, setMessage] = useState('')
  const [token, setToken] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const form = useForm({
    defaultValues: { email: '', password: '', nickname: '', dateOfBirth: '', consent: false },
    onSubmit: async ({ value }) => {
      setMessage('')
      const error = stepError(step, value)
      if (error) { setMessage(error); return }
      if (step < 3) { setStep(step + 1); return }
      if (!schema.safeParse(value).success) { setMessage('Check every field and use a password of at least 12 characters.'); return }
      try {
        const result = await register(toRegistrationRequest(value))
        setToken(result.verificationToken ?? '')
        setMessage('Your private profile was created. Verify your email before signing in.')
      } catch (error) { setMessage((error as ApiProblem).detail ?? 'Registration failed.') }
    },
  })

  return <main className="register-page">
    <div className="register-atmosphere" aria-hidden="true" />
    <header className="register-header">
      <Link className="register-brand" to="/"><img src="/brand/matchmate-logo-nav.png" alt="" /><span>MatchMate</span></Link>
      <nav><a href="/#about">How it works</a><a href="/#trust">Privacy</a><Link to="/login">Sign in</Link></nav>
    </header>
    <section className="register-stage" aria-labelledby="register-title">
      <div className="register-intro"><p>Begin your journey</p><h1 id="register-title">Create your <em>private profile.</em></h1><span>Start with the essentials. Your identity and matching preferences remain private by default.</span></div>
      <div className="register-card">
        <ol className="register-steps" aria-label="Profile creation steps">{steps.map((label, index) => <li key={label} className={index === step ? 'current' : index < step ? 'complete' : ''}><b>{index < step ? '✓' : index + 1}</b><span>{label}</span></li>)}</ol>
        <form onSubmit={(event) => { event.preventDefault(); void form.handleSubmit() }}>
          {step === 0 && <><h2>Essential origins</h2><p>Use a nickname for the profile other members may see. Your exact date of birth only confirms adulthood.</p><div className="register-fields"><Field form={form} name="nickname" label="Nickname" placeholder="Choose a name to be known by" /><Field form={form} name="dateOfBirth" label="Date of birth" type="date" /></div></>}
          {step === 1 && <><h2>Behind the veil</h2><p>Your email is for account security and verification. It is never shown to other members.</p><div className="register-fields"><Field form={form} name="email" label="Email address" type="email" placeholder="name@example.com" /><form.Field name="password">{(field: any) => <label><span>Password</span><div className="register-password"><input autoComplete="new-password" type={showPassword ? 'text' : 'password'} placeholder="At least 12 characters" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} /><button type="button" onClick={() => setShowPassword(!showPassword)}>{showPassword ? 'Hide' : 'Show'}</button></div></label>}</form.Field></div></>}
          {step === 2 && <><h2>You control the reveal</h2><p>After verification, you can complete the community profile at your own pace. Nothing is visible until you choose community visibility and moderation approves it.</p><div className="visibility-rules"><p><b>Always private</b>Email, exact date of birth, matching preferences, and deal-breakers.</p><p><b>Only with your approval</b>Nickname, broad location, bio, age band, and interests.</p></div></>}
          {step === 3 && <><h2>Your privacy vow</h2><p>MatchMate is for adults seeking respectful, consent-led connections. Confirm the privacy terms to create your account.</p><form.Field name="consent">{(field: any) => <label className="register-consent"><input type="checkbox" checked={field.state.value} onChange={(event) => field.handleChange(event.target.checked)} /><span>I am 18 or older and accept the privacy terms. I understand my profile is private until I choose otherwise.</span></label>}</form.Field></>}
          <div className="register-actions">{step > 0 && <button type="button" onClick={() => { setMessage(''); setStep(step - 1) }}>Back</button>}<button className="button" type="submit">{step === 3 ? 'Create private profile' : 'Continue journey'} →</button></div>
        </form>
        {message && <p className="form-message" role="status">{message}</p>}
        {token && <button className="text-action" type="button" onClick={async () => { try { await verifyEmail(token); setMessage('Email verified. You can now sign in.') } catch (error) { setMessage((error as ApiProblem).detail) } }}>Verify development account</button>}
      </div>
      <p className="register-existing">Already have an account? <Link to="/login">Sign in</Link></p>
    </section>
  </main>
}

function Field({ form, name, label, type = 'text', placeholder }: { form: any; name: keyof FormShape; label: string; type?: string; placeholder?: string }) { return <form.Field name={name}>{(field: any) => <label><span>{label}</span><input type={type} placeholder={placeholder} value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} /></label>}</form.Field> }
function stepError(step: number, value: FormShape) { if (step === 0 && (!value.nickname.trim() || value.dateOfBirth.length < 10)) return 'Add a nickname and your date of birth to continue.'; if (step === 1 && (!z.email().safeParse(value.email).success || value.password.length < 12)) return 'Use a valid email address and a password with at least 12 characters.'; if (step === 3 && !value.consent) return 'Please confirm that you are 18 or older and accept the privacy terms.'; return '' }
export function toRegistrationRequest(value: FormShape) { return { email: value.email, password: value.password, nickname: value.nickname, dateOfBirth: value.dateOfBirth, consentVersion: 'privacy-2026-08' } }
