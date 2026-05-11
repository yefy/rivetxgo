package rivetxsql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestCreate_ExecutesAndIsIdempotent(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		t.Fatalf("setup test_data table failed: %v", err)
	}
	testDataDropTable(rivetxsql)

	if err := Create[TestData](rivetxsql, "test_data", 0); err != nil {
		t.Fatalf("Create[TestData] failed: %v", err)
	}

	if err := Create[TestData](rivetxsql, "test_data", 3*time.Second); err != nil {
		t.Fatalf("Create[TestData] with explicit timeout failed: %v", err)
	}

	if err := testKeyCreateTable(rivetxsql); err != nil {
		t.Fatalf("setup test_key table failed: %v", err)
	}
	testKeyDropTable(rivetxsql)

	if err := Create[Testkey](rivetxsql, "test_key", 0); err != nil {
		t.Fatalf("Create[Testkey] failed: %v", err)
	}
}

func TestCreate_InvalidStructTypeReturnsError(t *testing.T) {
	err := Create[int](nil, "invalid", 0)
	if err == nil {
		t.Fatal("expected error for non-struct type, got nil")
	}
}
