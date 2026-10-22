// Tombol ganti tema berupa ikon lampu: menyala (emas) di mode terang, padam (garis) di mode gelap.
export default function TombolTema({ tema, onUbah, className = '' }) {
  const terang = tema !== 'gelap'
  const label = terang ? 'Ganti ke mode gelap' : 'Ganti ke mode terang'

  return (
    <button
      type="button"
      className={`ghost tema-ikon ${className}`.trim()}
      onClick={onUbah}
      aria-label={label}
      title={label}
    >
      <svg
        className={terang ? 'bohlam nyala' : 'bohlam'}
        viewBox="-2 -2 28 28"
        width="24"
        height="24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5" />
        <path d="M9 18h6" />
        <path d="M10 22h4" />
        <g className="sinar">
          <path d="M12 -1.5v1.2" />
          <path d="M4.6 1.2l.9.9" />
          <path d="M19.4 1.2l-.9.9" />
          <path d="M1 8h1.2" />
          <path d="M21.8 8H23" />
        </g>
      </svg>
    </button>
  )
}
