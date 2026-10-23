ALTER TABLE washing_jobs MODIFY COLUMN status ENUM('menunggu','proses','selesai') NOT NULL DEFAULT 'menunggu';

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'washer_sent_date');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN washer_sent_date DATE NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'sent_by');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN sent_by INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;