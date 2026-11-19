import { KARAT } from '../../constants/barang'

export const FILTER_PRODUKSI_KOSONG = { q: '', karat: '', karats: [], berat_min: '', berat_max: '', asal: '', dari: '', sampai: '', orang: '' }

// susun parameter query untuk server (nama field sama dengan yang dibaca backend)
export function paramProduksi(f) {
  const p = new URLSearchParams()
  if (f.karats?.length) f.karats.forEach((k) => p.append('karat', k))
  const peta = {
    q: f.q.trim(), karat: f.karats?.length ? '' : f.karat, berat_min: f.berat_min, berat_max: f.berat_max,
    asal: f.asal, from: f.dari, to: f.sampai, orang: f.orang.trim(),
  }
  Object.entries(peta).forEach(([k, v]) => v && p.set(k, v))
  return p
}

// Filter bersama untuk menu Cuci, Lebur, dan Service. children = filter status milik halaman.
// labelOrang: nama kolom orangnya (Pencuci, Penyervice, Dicatat oleh); tampilAsal=false untuk akun cabang.
export default function FilterProduksi({ f, onUbah, locations, tampilAsal = true, labelNama, labelOrang, children, aksiTambahan, multipleKarat = false }) {
  const set = (bagian) => onUbah({ ...f, ...bagian })
  const aktif = JSON.stringify(f) !== JSON.stringify(FILTER_PRODUKSI_KOSONG)
  const karats = f.karats || []
  const toggleKarat = (karat) => {
    const next = karats.includes(karat)
      ? karats.filter((k) => k !== karat)
      : [...karats, karat]
    set({ karat: '', karats: next })
  }
  return (
    <div style={{ marginBottom: 12 }}>
      <div className="form-grid">
        {children}
        <div>
          <label>{labelNama}</label>
          <input value={f.q} onChange={(e) => set({ q: e.target.value })} placeholder="Cincin, C-001..." />
        </div>
        <div>
          <label>Kadar</label>
          {multipleKarat ? (
            <details className="filter-kadar-dropdown">
              <summary>{karats.length ? `${karats.length} kadar dipilih` : 'Semua kadar'}</summary>
              <div className="filter-kadar-pilihan">
                <div className="filter-kadar-aksi">
                  <button type="button" className="mini" onClick={() => set({ karat: '', karats: KARAT.map((n) => `${n}K`) })}>
                    Pilih semua
                  </button>
                  {karats.length > 0 && (
                    <button type="button" className="mini" onClick={() => set({ karat: '', karats: [] })}>
                      Hapus pilihan
                    </button>
                  )}
                </div>
                <div className="filter-kadar-grid">
                  {KARAT.map((n) => {
                    const value = `${n}K`
                    return (
                      <label key={n}>
                        <input type="checkbox" checked={karats.includes(value)} onChange={() => toggleKarat(value)} />
                        {value}
                      </label>
                    )
                  })}
                </div>
              </div>
            </details>
          ) : (
            <select value={f.karat} onChange={(e) => set({ karat: e.target.value, karats: [] })}>
              <option value="">Semua</option>
              {KARAT.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
            </select>
          )}
        </div>
        <div>
          <label>Berat dari (g)</label>
          <input type="number" step="0.001" min="0" value={f.berat_min} onChange={(e) => set({ berat_min: e.target.value })} />
        </div>
        <div>
          <label>Berat sampai (g)</label>
          <input type="number" step="0.001" min="0" value={f.berat_max} onChange={(e) => set({ berat_max: e.target.value })} />
        </div>
        {tampilAsal && (
          <div>
            <label>Asal</label>
            <select value={f.asal} onChange={(e) => set({ asal: e.target.value })}>
              <option value="">Semua</option>
              {locations.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select>
          </div>
        )}
        <div>
          <label>Mulai dari tanggal</label>
          <input type="date" value={f.dari} onChange={(e) => set({ dari: e.target.value })} />
        </div>
        <div>
          <label>Mulai sampai tanggal</label>
          <input type="date" value={f.sampai} onChange={(e) => set({ sampai: e.target.value })} />
        </div>
        {labelOrang && (
          <div>
            <label>{labelOrang}</label>
            <input value={f.orang} onChange={(e) => set({ orang: e.target.value })} placeholder="Nama" />
          </div>
        )}
      </div>
      {(aktif || aksiTambahan) && (
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12, marginTop: 8 }}>
          {aktif && (
            <button type="button" className="mini" onClick={() => onUbah(FILTER_PRODUKSI_KOSONG)}>
              Reset filter
            </button>
          )}
          {aksiTambahan}
        </div>
      )}
    </div>
  )
}
