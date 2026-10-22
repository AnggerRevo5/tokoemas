package models

import "time"

// StockMovement mencerminkan tabel stock_movements (mutasi stok gram).
type StockMovement struct {
	ID              int       `json:"id"`
	LocationID      int       `json:"location_id"`
	Direction       string    `json:"direction"` // in | out
	Category        string    `json:"category"`
	ItemName        string    `json:"item_name"`
	WeightGram      float64   `json:"weight_gram"`
	Karat           *string   `json:"karat"`
	Qty             int       `json:"qty"`
	Note            *string   `json:"note"`
	TransactionDate time.Time `json:"transaction_date"`
	RelatedID       *int      `json:"related_id"`
	ItemID          *int      `json:"item_id"`
	IsVoid          bool      `json:"is_void"`
	VoidReason      *string   `json:"void_reason"`
	CreatedBy       int       `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// CashTransaction mencerminkan tabel cash_transactions (mutasi kas rupiah).
type CashTransaction struct {
	ID              int       `json:"id"`
	LocationID      int       `json:"location_id"`
	Direction       string    `json:"direction"` // in | out
	Category        string    `json:"category"`
	Amount          float64   `json:"amount"`
	Note            *string   `json:"note"`
	TransactionDate time.Time `json:"transaction_date"`
	RelatedID       *int      `json:"related_id"`
	IsVoid          bool      `json:"is_void"`
	VoidReason      *string   `json:"void_reason"`
	CreatedBy       int       `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLog mencerminkan tabel audit_logs.
type AuditLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Action    string    `json:"action"`
	TableName string    `json:"table_name"`
	RecordID  *int      `json:"record_id"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}
