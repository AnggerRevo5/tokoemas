-- Migration: fitur cuci (dengan status) dan lebur (dengan harga)
-- Aman dijalankan berulang

CREATE TABLE IF NOT EXISTS washing_jobs (
  id INT AUTO_INCREMENT PRIMARY KEY,
  origin_location_id INT NOT NULL,
  item_name VARCHAR(100) NOT NULL,
  weight_gram DECIMAL(10,3) NOT NULL,
  karat VARCHAR(10) NULL,
  qty INT NOT NULL DEFAULT 1,
  washer_name VARCHAR(100) NOT NULL,
  status ENUM('proses','selesai') NOT NULL DEFAULT 'proses',
  start_date DATE NOT NULL,
  finish_date DATE NULL,
  note VARCHAR(255) NULL,
  out_stock_id INT NULL,
  in_stock_id INT NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (origin_location_id) REFERENCES locations(id),
  FOREIGN KEY (out_stock_id) REFERENCES stock_movements(id),
  FOREIGN KEY (in_stock_id) REFERENCES stock_movements(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS melt_records (
  id INT AUTO_INCREMENT PRIMARY KEY,
  record_date DATE NOT NULL,
  origin_location_id INT NOT NULL,
  item_name VARCHAR(100) NOT NULL,
  karat VARCHAR(10) NULL,
  weight_gram DECIMAL(10,3) NOT NULL,
  qty INT NOT NULL DEFAULT 1,
  melt_total DECIMAL(15,2) NOT NULL,
  sale_total DECIMAL(15,2) NOT NULL,
  note VARCHAR(255) NULL,
  stock_id INT NULL,
  cash_id INT NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (origin_location_id) REFERENCES locations(id),
  FOREIGN KEY (stock_id) REFERENCES stock_movements(id),
  FOREIGN KEY (cash_id) REFERENCES cash_transactions(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);