import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { LazyMotion, MotionConfig, domMax } from 'motion/react'
import './styles/index.css'
import App from './App.jsx'
import { temaAwal, terapkanTema } from './utils/tema'

terapkanTema(temaAwal())

// LazyMotion: hanya fitur animasi yang dipakai (komponen m.*) agar bundel lebih ringan.
// reducedMotion="user": semua animasi motion mati bila perangkat meminta prefers-reduced-motion.
createRoot(document.getElementById('root')).render(
  <StrictMode>
    <LazyMotion features={domMax} strict>
      <MotionConfig reducedMotion="user">
        <App />
      </MotionConfig>
    </LazyMotion>
  </StrictMode>,
)
