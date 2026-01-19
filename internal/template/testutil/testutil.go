// Package testutil provides shared test helpers for the template package tests.
package testutil

import (
	"strings"
	"testing"
)

// ContainsSubstring checks if haystack contains needle (case-insensitive).
func ContainsSubstring(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// Error represents any error type with a GetError method.
type Error interface {
	GetError() string
}

// AssertNoErrors fails the test if errors slice is not empty.
func AssertNoErrors[E Error](t *testing.T, errs []E) {
	t.Helper()
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}
}

// AssertErrorCount fails if error count doesn't match expected.
func AssertErrorCount[E Error](t *testing.T, errs []E, expected int) {
	t.Helper()
	if len(errs) != expected {
		t.Fatalf("Expected %d errors, got %d: %v", expected, len(errs), errs)
	}
}

// AssertErrorContains fails if no error contains the expected substring.
func AssertErrorContains[E Error](t *testing.T, errs []E, expected string) {
	t.Helper()
	for _, err := range errs {
		if ContainsSubstring(err.GetError(), expected) {
			return
		}
	}
	t.Errorf("Expected error containing %q, got: %v", expected, errs)
}

// RequireErrorContains is like AssertErrorContains but calls t.Fatal.
func RequireErrorContains[E Error](t *testing.T, errs []E, expected string) {
	t.Helper()
	for _, err := range errs {
		if ContainsSubstring(err.GetError(), expected) {
			return
		}
	}
	t.Fatalf("Expected error containing %q, got: %v", expected, errs)
}

// HasErrorContaining returns true if any error contains the expected substring.
func HasErrorContaining[E Error](errs []E, expected string) bool {
	for _, err := range errs {
		if ContainsSubstring(err.GetError(), expected) {
			return true
		}
	}
	return false
}
