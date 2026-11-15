CREATE TABLE IF NOT EXISTS melt_queue_items (
	item_id INT PRIMARY KEY,
	queued_date DATE NOT NULL,
	note VARCHAR(200) NOT NULL DEFAULT '',
	queued_by INT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_melt_queue_date (queued_date),
	FOREIGN KEY (item_id) REFERENCES items(id),
	FOREIGN KEY (queued_by) REFERENCES users(id)
);
