import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useForm } from '@tanstack/react-form'
import { ApiProblem, login } from '../lib/account-api'
import { AccountShell } from './RegisterPage'

export function LoginPage(){const [message,setMessage]=useState('');const navigate=useNavigate();const form=useForm({defaultValues:{email:'',password:''},onSubmit:async({value})=>{try{await login(value.email,value.password);await navigate({to:'/app/profile'})}catch(error){setMessage((error as ApiProblem).detail??'Sign in failed.')}}});return <AccountShell title="Welcome back" intro="Sign in to manage your private profile and matchmaking preferences."><form onSubmit={e=>{e.preventDefault();void form.handleSubmit()}}><form.Field name="email">{field=><label>Email<input type="email" value={field.state.value} onChange={e=>field.handleChange(e.target.value)}/></label>}</form.Field><form.Field name="password">{field=><label>Password<input type="password" value={field.state.value} onChange={e=>field.handleChange(e.target.value)}/></label>}</form.Field><button className="button" type="submit">Sign in</button></form>{message&&<p className="form-message" role="alert">{message}</p>}<p>New here? <Link to="/register">Create an account</Link></p></AccountShell>}
