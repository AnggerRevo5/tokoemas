CREATE TABLE IF NOT EXISTS items (
  id INT AUTO_INCREMENT PRIMARY KEY,
  item_no VARCHAR(20) NULL UNIQUE,
  item_name VARCHAR(100) NOT NULL,
  karat VARCHAR(10) NULL,
  weight_gram DECIMAL(10,3) NOT NULL,
  origin_location_id INT NOT NULL,
  location_id INT NOT NULL,
  status ENUM('tersedia','cuci','lebur','terjual') NOT NULL DEFAULT 'tersedia',
  source_category VARCHAR(30) NOT NULL,
  entry_date DATE NOT NULL,
  note VARCHAR(255) NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (origin_location_id) REFERENCES locations(id),
  FOREIGN KEY (location_id) REFERENCES locations(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

SET @col := (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'stock_movements' AND COLUMN_NAME = 'item_id');
SET @sql := IF(@col = 0, 'ALTER TABLE stock_movements ADD COLUMN item_id INT NULL', 'SELECT 1');
PREPARE s FROM @sql;
EXECUTE s;
DEALLOCATE PREPARE s;