package rivetxsql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const insertTestTable = "insert_test_data"

func setupInsertTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, err := rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + insertTestTable)
	if err != nil {
		t.Fatalf("drop insert test table failed: %v", err)
	}

	query := `
CREATE TABLE ` + insertTestTable + ` (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	index_col INT NOT NULL,
	key_col VARCHAR(64) NOT NULL,
	name_id INT UNSIGNED NOT NULL,
	name_index INT UNSIGNED NOT NULL,
	curr_time DATETIME NOT NULL,
	PRIMARY KEY (id),
	UNIQUE INDEX u_it_key (index_col, key_col)
);`
	_, err = rivetxsql.Pool.Exec(query)
	if err != nil {
		t.Fatalf("create insert test table failed: %v", err)
	}
}

func teardownInsertTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, _ = rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + insertTestTable)
}

func countInsertTestRows(t *testing.T, rivetxsql *RivetxSql) int {
	t.Helper()
	var count int
	if err := rivetxsql.Pool.QueryRow("SELECT COUNT(*) FROM " + insertTestTable).Scan(&count); err != nil {
		t.Fatalf("count rows failed: %v", err)
	}
	return count
}

func TestInsertRaw_ErrorCases(t *testing.T) {
	_, err := InsertRaw(nil, "insert_test_data", nil, [][]interface{}{{1}}, 1, "", false, 0)
	if err == nil {
		t.Fatal("expected error when cols is empty")
	}

	_, err = InsertRaw(nil, "insert_test_data", []string{"index_col"}, nil, 1, "", false, 0)
	if err == nil {
		t.Fatal("expected error when vals is empty")
	}

	_, err = InsertRaw(nil, "insert_test_data", []string{"index_col", "key_col"}, [][]interface{}{{1}}, 1, "", false, 0)
	if err == nil {
		t.Fatal("expected error when row length does not match cols length")
	}
}

func TestInsertRaw_DuplicateUpdate(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupInsertTestTable(t, rivetxsql)
	defer testDataDropTable(rivetxsql)

	cols := []string{"index_col", "key_col", "name_id", "name_index", "curr_time"}
	vals := [][]interface{}{
		{0, "abc", 1, 1001, time.Now().Truncate(time.Second)},
		{1, "xyz", 2, 1002, time.Now().Truncate(time.Second)},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"
	result, err := InsertRaw(rivetxsql, "insert_test_data", cols, vals, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		t.Fatalf("InsertRaw failed: %v", err)
	}
	if result.TotalAffected != 2 {
		t.Fatalf("expected TotalAffected 2, got %d", result.TotalAffected)
	}

	if count := countInsertTestRows(t, rivetxsql); count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	_, err = InsertRaw(rivetxsql, "insert_test_data", cols, [][]interface{}{{0, "abc", 11, 11, time.Now().Truncate(time.Second)}}, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		t.Fatalf("InsertRaw duplicate update failed: %v", err)
	}

	var nameId, nameIndex int
	if err := rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM insert_test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex); err != nil {
		t.Fatalf("query row failed: %v", err)
	}
	if nameId != 11 || nameIndex != 1012 {
		t.Fatalf("expected duplicate update to set name_id=11 and name_index=1012, got %d/%d", nameId, nameIndex)
	}
}

func TestInsertRaw_IgnoreDuplicate(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupInsertTestTable(t, rivetxsql)
	defer teardownInsertTestTable(t, rivetxsql)

	cols := []string{"index_col", "key_col", "name_id", "name_index", "curr_time"}
	vals := [][]interface{}{
		{0, "abc", 1, 1001, time.Now().Truncate(time.Second)},
	}

	if _, err := InsertRaw(rivetxsql, "insert_test_data", cols, vals, 2, "", true, 10*time.Second); err != nil {
		t.Fatalf("InsertRaw ignored duplicate failed: %v", err)
	}
	if _, err := InsertRaw(rivetxsql, "insert_test_data", cols, vals, 2, "", true, 10*time.Second); err != nil {
		t.Fatalf("InsertRaw ignore duplicate second insert failed: %v", err)
	}

	if count := countInsertTestRows(t, rivetxsql); count != 1 {
		t.Fatalf("expected 1 row after INSERT IGNORE duplicate, got %d", count)
	}
}

func TestInsertStruct_DuplicateUpdate(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupInsertTestTable(t, rivetxsql)
	defer teardownInsertTestTable(t, rivetxsql)

	testData := []TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "xyz", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"
	result, err := Insert(rivetxsql, "insert_test_data", testData, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if result.TotalAffected != 2 {
		t.Fatalf("expected TotalAffected 2, got %d", result.TotalAffected)
	}

	if count := countInsertTestRows(t, rivetxsql); count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	dup := []TestData{{0, 0, "abc", 11, 11, time.Now().Truncate(time.Second), time.Time{}, time.Time{}}}
	if _, err := Insert(rivetxsql, "insert_test_data", dup, 2, onDuplicate, false, 10*time.Second); err != nil {
		t.Fatalf("Insert duplicate struct failed: %v", err)
	}

	var nameId, nameIndex int
	if err := rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM insert_test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex); err != nil {
		t.Fatalf("query row failed: %v", err)
	}
	if nameId != 11 || nameIndex != 1012 {
		t.Fatalf("expected duplicate update to set name_id=11 and name_index=1012, got %d/%d", nameId, nameIndex)
	}
}

func TestInsertBuilder_Exec(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupInsertTestTable(t, rivetxsql)
	defer teardownInsertTestTable(t, rivetxsql)

	testData := []TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "xyz", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	result, err := NewInsert("insert_test_data", testData).BatchSize(2).OnDuplicateUpdate("name_id = VALUES(name_id)").Timeout(10 * time.Second).Exec(rivetxsql)
	if err != nil {
		t.Fatalf("NewInsert Exec failed: %v", err)
	}
	if result.TotalAffected != 2 {
		t.Fatalf("expected TotalAffected 2, got %d", result.TotalAffected)
	}

	if count := countInsertTestRows(t, rivetxsql); count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}
