// Dropdown daftar lokasi (opsi tambahan di awal lewat children, misalnya "Semua lokasi").
export default function PilihanLokasi({ locations, value, onChange, children }) {
  return (
    <select value={value} onChange={onChange}>
      {children}
      {locations.map((l) => (
        <option key={l.id} value={l.id}>{l.name}</option>
      ))}
    </select>
  )
}
