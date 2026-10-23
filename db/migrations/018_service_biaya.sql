SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'service_jobs' AND COLUMN_NAME = 'service_fee');
SET @s := IF(@c = 0, 'ALTER TABLE service_jobs ADD COLUMN service_fee DECIMAL(15,2) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'service_jobs' AND COLUMN_NAME = 'fee_cash_id');
SET @s := IF(@c = 0, 'ALTER TABLE service_jobs ADD COLUMN fee_cash_id INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;
