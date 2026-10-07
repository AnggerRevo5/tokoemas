CREATE TABLE locations (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  type ENUM('cabang','back_office') NOT NULL
);

CREATE TABLE users (
  id INT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role ENUM('super_admin','owner','cabang') NOT NULL,
  location_id INT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (location_id) REFERENCES locations(id)
);

INSERT INTO locations (name, type) VALUES
('Kepanjen', 'cabang'),
('Mall Dinoyo', 'cabang'),
('Back Office', 'back_office');

CREATE TABLE stock_movements (
  id INT AUTO_INCREMENT PRIMARY KEY,
  location_id INT NOT NULL,
  direction ENUM('in','out') NOT NULL,
  category VARCHAR(30) NOT NULL,
  item_name VARCHAR(100) NOT NULL,
  weight_gram DECIMAL(10,3) NOT NULL,
  karat VARCHAR(10) NULL,
  qty INT NOT NULL DEFAULT 1,
  note VARCHAR(255) NULL,
  transaction_date DATE NOT NULL,
  related_id INT NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (location_id) REFERENCES locations(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE cash_transactions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  location_id INT NOT NULL,
  direction ENUM('in','out') NOT NULL,
  category VARCHAR(30) NOT NULL,
  amount DECIMAL(15,2) NOT NULL,
  note VARCHAR(255) NULL,
  transaction_date DATE NOT NULL,
  related_id INT NULL,
  is_void BOOLEAN NOT NULL DEFAULT FALSE,
  void_reason VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (location_id) REFERENCES locations(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE debts (
  id INT AUTO_INCREMENT PRIMARY KEY,
  creditor_type ENUM('investor','perorangan','pabrik','bank_pegadaian') NOT NULL,
  creditor_name VARCHAR(100) NOT NULL,
  principal DECIMAL(15,2) NOT NULL,
  note VARCHAR(255) NULL,
  start_date DATE NOT NULL,
  due_date DATE NULL,
  status ENUM('aktif','lunas') NOT NULL DEFAULT 'aktif',
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE debt_payments (
  id INT AUTO_INCREMENT PRIMARY KEY,
  debt_id INT NOT NULL,
  amount DECIMAL(15,2) NOT NULL,
  paid_date DATE NOT NULL,
  note VARCHAR(255) NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (debt_id) REFERENCES debts(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE audit_logs (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  action VARCHAR(30) NOT NULL,
  table_name VARCHAR(50) NOT NULL,
  record_id INT NULL,
  old_value JSON NULL,
  new_value JSON NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);