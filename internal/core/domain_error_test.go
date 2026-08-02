package core

import "testing"

// TestAllErrorCodesCarriesBudgetExceeded pins the error-code catalog: eleven
// codes, including budget_exceeded (the distinct quota-refusal code the work
// budgets and the peer-transfer velocity ceiling answer with), each with a
// stable wire value.
func TestAllErrorCodesCarriesBudgetExceeded(t *testing.T) {
	codes := AllErrorCodes()
	if len(codes) != 11 {
		t.Fatalf("AllErrorCodes() has %d codes, want 11", len(codes))
	}
	budgetExceededListed := false
	for index, code := range codes {
		for _, earlier := range codes[:index] {
			if earlier.String() == code.String() {
				t.Fatalf("duplicate error code %q", code.String())
			}
		}
		if code.String() == "budget_exceeded" {
			budgetExceededListed = true
		}
	}
	if !budgetExceededListed {
		t.Fatalf("AllErrorCodes() is missing budget_exceeded: %v", codes)
	}
	if ErrorCodeBudgetExceeded.String() != "budget_exceeded" {
		t.Fatalf("ErrorCodeBudgetExceeded wire value = %q", ErrorCodeBudgetExceeded.String())
	}
}
