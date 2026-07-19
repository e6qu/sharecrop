//go:build integration

package integration_test

import (
	"testing"

	"github.com/google/uuid"
)

func uniqueIntegrationEmail(t *testing.T, prefix string) string {
	t.Helper()
	return prefix + "-" + uuid.NewString() + "@example.com"
}
