package main

import "testing"

func TestSeedEnvironmentGuard(t *testing.T) {
	for _, environment := range []string{"development", "test"} {
		if err := ensureDevelopment(environment); err != nil {
			t.Fatalf("%s should be allowed: %v", environment, err)
		}
	}
	for _, environment := range []string{"production", "staging", ""} {
		if err := ensureDevelopment(environment); err == nil {
			t.Fatalf("%q should be rejected", environment)
		}
	}
}

func TestHardCodedAccountsUseReservedTestDomain(t *testing.T) {
	for _, account := range developmentAccounts {
		if len(account.email) < len(".test") || account.email[len(account.email)-len(".test"):] != ".test" {
			t.Fatalf("seed email is not reserved test data: %s", account.email)
		}
	}
}
