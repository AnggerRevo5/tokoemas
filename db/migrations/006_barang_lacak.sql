SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'jenis');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN jenis ENUM(''logam_mulia'',''perhiasan'',''hasil_lebur'') NOT NULL DEFAULT ''perhiasan''', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'brand');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN brand VARCHAR(50) NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'kondisi');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN kondisi ENUM(''kotor'',''bersih'') NOT NULL DEFAULT ''bersih''', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'harga_beli');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN harga_beli DECIMAL(15,2) NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'purchase_cash_id');
SET @s := IF(@c = 0, 'ALTER TABLE items ADD COLUMN purchase_cash_id INT NULL', 'SELECT 1');
PREPARE p FROM @s;
EXECUTE p;
DEALLOCATE PREPARE p;

ALTER TABLE items MODIFY COLUMN status ENUM('tersedia','dikirim','cuci','lebur','terjual') NOT NULL DEFAULT 'tersedia';