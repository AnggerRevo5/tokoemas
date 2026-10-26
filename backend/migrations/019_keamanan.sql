-- Migration 019: sesi login di server, log keamanan, dan kolom audit tambahan.
-- Aditif: hanya membuat tabel baru dan menambah kolom NULL. Tidak mengubah atau menghapus data lama.
-- Aman dijalankan berulang. Backend juga mencoba menjalankan ini otomatis saat start;
-- jalankan manual bila user database aplikasi tidak punya hak CREATE/ALTER.

CREATE TABLE IF NOT EXISTS user_sessions (
  id CHAR(32) NOT NULL PRIMARY KEY,
  user_id INT NOT NULL,
  created_at DATETIME NOT NULL,
  last_seen DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  revoked_at DATETIME NULL,
  ip VARCHAR(45) NULL,
  user_agent VARCHAR(255) NULL,
  INDEX idx_sesi_user (user_id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS security_events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  event VARCHAR(40) NOT NULL,
  username VARCHAR(50) NULL,
  user_id INT NULL,
  location_id INT NULL,
  ip VARCHAR(45) NULL,
  user_agent VARCHAR(255) NULL,
  detail VARCHAR(255) NULL,
  INDEX idx_event_waktu (created_at),
  INDEX idx_event_username (username)
);

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'audit_logs' AND COLUMN_NAME = 'username');
SET @s := IF(@c = 0, 'ALTER TABLE audit_logs ADD COLUMN username VARCHAR(50) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'audit_logs' AND COLUMN_NAME = 'location_id');
SET @s := IF(@c = 0, 'ALTER TABLE audit_logs ADD COLUMN location_id INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'audit_logs' AND COLUMN_NAME = 'ip');
SET @s := IF(@c = 0, 'ALTER TABLE audit_logs ADD COLUMN ip VARCHAR(45) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;
