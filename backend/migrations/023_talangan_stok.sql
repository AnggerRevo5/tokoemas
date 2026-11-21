CREATE TABLE IF NOT EXISTS talangan (
  id INT AUTO_INCREMENT PRIMARY KEY,
  location_id INT NOT NULL,
  bp_nama VARCHAR(30) NOT NULL,
  arah VARCHAR(12) NOT NULL,
  item_name VARCHAR(100) NOT NULL,
  weight_gram DECIMAL(10,3) NOT NULL DEFAULT 0,
  karat VARCHAR(10) NULL,
  nilai DECIMAL(15,2) NOT NULL,
  entry_date DATE NOT NULL,
  note VARCHAR(255) NULL,
  konfirmasi BOOLEAN NOT NULL DEFAULT FALSE,
  konfirmasi_note VARCHAR(255) NULL,
  konfirmasi_tgl DATE NULL,
  status VARCHAR(12) NOT NULL DEFAULT 'aktif',
  tukar_id INT NULL,
  batal_alasan VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_talangan (location_id, bp_nama, status)
);

ALTER TABLE items
  ADD COLUMN bp_asal VARCHAR(30) NULL,
  ADD COLUMN bp_nilai DECIMAL(15,2) NULL,
  MODIFY COLUMN status ENUM('tersedia','dikirim','cuci','lebur','terjual','hasil','kembali','digadai','tidak_ditebus','di_bp','tukar_bp') NOT NULL DEFAULT 'tersedia';

ALTER TABLE talangan
  ADD COLUMN item_id INT NULL,
  ADD INDEX idx_talangan_item (item_id);
