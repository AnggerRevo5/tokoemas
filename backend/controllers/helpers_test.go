package controllers

import "testing"

func TestIDLokasiBackOfficeSesuai(t *testing.T) {
	tests := []struct {
		name        string
		requested   string
		backOffice  int
		wantAllowed bool
	}{
		{name: "back office", requested: "3", backOffice: 3, wantAllowed: true},
		{name: "other location", requested: "2", backOffice: 3},
		{name: "missing location", backOffice: 3},
		{name: "invalid location", requested: "abc", backOffice: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := idLokasiBackOfficeSesuai(tt.requested, tt.backOffice); got != tt.wantAllowed {
				t.Fatalf("idLokasiBackOfficeSesuai(%q, %d) = %t, want %t",
					tt.requested, tt.backOffice, got, tt.wantAllowed)
			}
		})
	}
}
