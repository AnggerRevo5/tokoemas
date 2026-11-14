import { KARAT } from '../../constants/barang'

export const FILTER_PRODUKSI_KOSONG = { q: '', karat: '', berat_min: '', berat_max: '', asal: '', dari: '', sampai: '', orang: '' }

// susun parameter query untuk server (nama field sama dengan yang dibaca backend)
export function paramProduksi(f) {
  const p = new URLSearchParams()
  const peta = {
    q: f.q.trim(), karat: f.karat, berat_min: f.berat_min, berat_max: f.berat_max,
    asal: f.asal, from: f.dari, to: f.sampai, orang: f.orang.trim(),
  }
  Object.entries(peta).forEach(([k, v]) => v && p.set(k, v))
  return p
}

// Filter bersama untuk menu Cuci, Lebur, dan Service. children = filter status milik halaman.
// labelOrang: nama kolom orangnya (Pencuci, Penyervice, Dicatat oleh); tampilAsal=false untuk akun cabang.
export default function FilterProduksi({ f, onUbah, locations, tampilAsal = true, labelNama, labelOrang, children, aksiTambahan }) {
  const set = (bagian) => onUbah({ ...f, ...bagian })
  const aktif = JSON.stringify(f) !== JSON.stringify(FILTER_PRODUKSI_KOSONG)
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
          <select value={f.karat} onChange={(e) => set({ karat: e.target.value })}>
            <option value="">Semua</option>
            {KARAT.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
          </select>
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
