package httpapi

import (
	"context"
	"os"
	"testing"

	"github.com/elvove/photography/backend-go/internal/store"
)

// testPool opens the database named by ELVOVE_TEST_DATABASE, or skips the test.
// Only tests that cannot state their claim without a real database use it: the
// queries in syncBookingPhotos are SQL, so a fake would test the fake.
//
//	ELVOVE_TEST_DATABASE="postgres://postgres:postgres@127.0.0.1:5435/photographer_portfolio" go test ./internal/httpapi/
func testPool(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("ELVOVE_TEST_DATABASE")
	if url == "" {
		t.Skip("ELVOVE_TEST_DATABASE is not set")
	}
	db, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open the test database: %v", err)
	}
	t.Cleanup(func() { db.Pool.Close() })
	return db
}
