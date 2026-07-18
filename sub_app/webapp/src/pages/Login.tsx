import { useState } from 'react'
import { apiClient } from '../api/client'

const Login = () => {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [msg, setMsg] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await apiClient.post('/api/auth/login', { email, password })
      setMsg('Login success: ' + JSON.stringify(res.data))
    } catch (err: unknown) {
      const detail = err instanceof Object && 'response' in err
        ? String((err as { response: { data: unknown } }).response.data)
        : String(err)
      setMsg('Error: ' + detail)
    }
  }

  return (
    <main style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>Login</h1>
      <form onSubmit={handleSubmit}>
        <div>
          <input placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} />
        </div>
        <div>
          <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} />
        </div>
        <button type="submit">Login</button>
      </form>
      {msg && <pre>{msg}</pre>}
      <br />
      <a href="/">Back home</a>
    </main>
  )
}

export default Login
