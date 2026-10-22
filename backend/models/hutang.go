package models

import "time"

// Debt mencerminkan tabel debts.
type Debt struct {
	ID           int        `json:"id"`
	CreditorType string     `json:"creditor_type"` // investor | perorangan | pabrik | bank_pegadaian
	CreditorName string     `json:"creditor_name"`
	Principal    float64    `json:"principal"`
	Note         *string    `json:"note"`
	StartDate    time.Time  `json:"start_date"`
	DueDate      *time.Time `json:"due_date"`
	Status       string     `json:"status"` // aktif | lunas
	CashID       *int       `json:"cash_id"`
	IsVoid       bool       `json:"is_void"`
	VoidReason   *string    `json:"void_reason"`
	CreatedBy    int        `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DebtPayment mencerminkan tabel debt_payments.
type DebtPayment struct {
	ID        int       `json:"id"`
	DebtID    int       `json:"debt_id"`
	Amount    float64   `json:"amount"`
	PaidDate  time.Time `json:"paid_date"`
	Note      *string   `json:"note"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}
