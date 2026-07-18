import { useEffect, useState } from 'react'
import { apiClient } from '../api/client'

const Dashboard = () => {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState('')

  useEffect(() => {
    apiClient.get('/api/')
      .then(res => setData(res.data as Record<string, unknown>))
      .catch(e => setErr(String(e)))
  }, [])

  return (
    <main style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>Dashboard</h1>
      {err && <p style={{ color: 'red' }}>{err}</p>}
      {data && <pre>{JSON.stringify(data, null, 2)}</pre>}
      <a href="/">Back home</a>
    </main>
  )
}

export default Dashboard
