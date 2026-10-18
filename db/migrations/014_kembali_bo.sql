ALTER TABLE items MODIFY COLUMN status ENUM('tersedia','dikirim','cuci','lebur','terjual','hasil','kembali') NOT NULL DEFAULT 'tersedia';

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'kembali_jenis');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN kembali_jenis VARCHAR(20) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

CREATE TABLE IF NOT EXISTS washing_job_items (
  id INT AUTO_INCREMENT PRIMARY KEY,
  job_id INT NOT NULL,
  item_id INT NOT NULL,
  weight_system DECIMAL(10,3) NOT NULL,
  weight_before DECIMAL(10,3) NULL,
  weight_after DECIMAL(10,3) NULL,
  UNIQUE KEY uq_job_item (job_id, item_id),
  FOREIGN KEY (job_id) REFERENCES washing_jobs(id),
  FOREIGN KEY (item_id) REFERENCES items(id)
);