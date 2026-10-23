ALTER TABLE users MODIFY COLUMN role ENUM('super_admin','owner','cabang','pj') NOT NULL;

ALTER TABLE items MODIFY COLUMN status ENUM('tersedia','dikirim','cuci','lebur','terjual','hasil') NOT NULL DEFAULT 'tersedia';

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'sale_id');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN sale_id INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

CREATE TABLE IF NOT EXISTS melt_batches (
  id INT AUTO_INCREMENT PRIMARY KEY,
  start_date DATE NOT NULL,
  status ENUM('proses','leburan_kotor','cukim','terjual') NOT NULL DEFAULT 'proses',
  item_count INT NOT NULL,
  source_weight DECIMAL(10,3) NOT NULL,
  modal DECIMAL(15,2) NOT NULL DEFAULT 0,
  cut_weight DECIMAL(10,3) NULL,
  cut_note VARCHAR(255) NULL,
  leburan_weight DECIMAL(10,3) NULL,
  leburan_date DATE NULL,
  leburan_by INT NULL,
  leburan_item_id INT NULL,
  cukim_weight DECIMAL(10,3) NULL,
  cukim_karat VARCHAR(10) NULL,
  cukim_date DATE NULL,
  cukim_by INT NULL,
  cukim_item_id INT NULL,
  note VARCHAR(255) NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS penadah_sales (
  id INT AUTO_INCREMENT PRIMARY KEY,
  sale_date DATE NOT NULL,
  penadah_name VARCHAR(100) NOT NULL,
  sale_total DECIMAL(15,2) NOT NULL,
  modal DECIMAL(15,2) NOT NULL DEFAULT 0,
  weight_gram DECIMAL(10,3) NOT NULL,
  item_count INT NOT NULL,
  note VARCHAR(255) NULL,
  cash_id INT NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);