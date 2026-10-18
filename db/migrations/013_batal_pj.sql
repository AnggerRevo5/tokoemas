ALTER TABLE washing_jobs MODIFY COLUMN status ENUM('menunggu','proses','selesai','batal') NOT NULL DEFAULT 'menunggu';
ALTER TABLE melt_batches MODIFY COLUMN status ENUM('proses','leburan_kotor','cukim','terjual','batal') NOT NULL DEFAULT 'proses';

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'cancel_reason');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN cancel_reason VARCHAR(255) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'cancelled_by');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN cancelled_by INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'cancelled_date');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN cancelled_date DATE NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'washing_jobs' AND COLUMN_NAME = 'cancel_returned');
SET @s := IF(@c = 0, 'ALTER TABLE washing_jobs ADD COLUMN cancel_returned INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'melt_batches' AND COLUMN_NAME = 'cancel_reason');
SET @s := IF(@c = 0, 'ALTER TABLE melt_batches ADD COLUMN cancel_reason VARCHAR(255) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'melt_batches' AND COLUMN_NAME = 'cancelled_by');
SET @s := IF(@c = 0, 'ALTER TABLE melt_batches ADD COLUMN cancelled_by INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'melt_batches' AND COLUMN_NAME = 'cancelled_date');
SET @s := IF(@c = 0, 'ALTER TABLE melt_batches ADD COLUMN cancelled_date DATE NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;