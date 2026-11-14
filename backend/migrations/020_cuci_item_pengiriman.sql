ALTER TABLE washing_job_items
	ADD COLUMN washer_name VARCHAR(100) NOT NULL DEFAULT '',
	ADD COLUMN sent_date DATE NULL;
