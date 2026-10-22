ALTER TABLE service_jobs MODIFY COLUMN status ENUM('di_toko','dikirim_bo','di_bo','diproses','kembali_bo','selesai','dikirim_toko','siap_diambil','diambil') NOT NULL DEFAULT 'di_toko';

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'service_jobs' AND COLUMN_NAME = 'returned_bo_date');
SET @s := IF(@c = 0, 'ALTER TABLE service_jobs ADD COLUMN returned_bo_date DATE NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;
