import { useState } from 'react'
import { api, adaSesi, hapusSesiLokal } from './services/api'
import Login from './components/auth/Login'
import AppShell from './components/layout/AppShell'

export default function App() {
  const [loggedIn, setLoggedIn] = useState(adaSesi)

  async function logout() {
    // sesi dicabut di server; tampilan tetap keluar walau permintaan gagal
    await api('/logout', { method: 'POST' }).catch(() => {})
    hapusSesiLokal()
    setLoggedIn(false)
  }

  return loggedIn ? <AppShell onLogout={logout} /> : <Login onSuccess={() => setLoggedIn(true)} />
}
