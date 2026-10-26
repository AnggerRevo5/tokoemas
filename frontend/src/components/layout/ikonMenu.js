import {
  ArrowLeftRight,
  Droplets,
  Factory,
  FileText,
  Flame,
  Gem,
  Landmark,
  LayoutDashboard,
  Settings,
  TrendingUp,
  Users,
  Wallet,
  Wrench,
  HandCoins,
} from 'lucide-react'

// ikon tiap menu dan kelompok menu di sidebar (kode sama dengan constants/roles)
export const IKON_MENU = {
  dashboard: LayoutDashboard,
  kas: Wallet,
  stok: Gem,
  cuci: Droplets,
  lebur: Flame,
  service: Wrench,
  laporan: FileText,
  hutang: HandCoins,
  labarugi: TrendingUp,
  akun: Users,
}

export const IKON_GRUP = {
  dashboard: LayoutDashboard,
  transaksi: ArrowLeftRight,
  produksi: Factory,
  keuangan: Landmark,
  pengaturan: Settings,
}
