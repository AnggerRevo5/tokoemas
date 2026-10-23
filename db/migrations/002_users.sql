-- Migration: akun login sesuai role
-- Ganti setiap <HASH_...> dengan hasil bcrypt (lihat panduan di bawah)
-- Aman dijalankan berulang: akun yang sudah ada tidak ditimpa

INSERT IGNORE INTO users (username, password_hash, role, location_id) VALUES
('admin',    '<HASH_ADMIN>',    'super_admin', NULL),
('owner',    '<HASH_OWNER>',    'owner',       NULL),
('kepanjen', '<HASH_KEPANJEN>', 'cabang',      1),
('dinoyo',   '<HASH_DINOYO>',   'cabang',      2);