import { useState } from 'react'
import { getToken, clearToken } from './services/api'
import Login from './components/auth/Login'
import AppShell from './components/layout/AppShell'

export default function App() {
  const [loggedIn, setLoggedIn] = useState(!!getToken())

  function logout() {
    clearToken()
    setLoggedIn(false)
  }

  return loggedIn ? <AppShell onLogout={logout} /> : <Login onSuccess={() => setLoggedIn(true)} />
}
