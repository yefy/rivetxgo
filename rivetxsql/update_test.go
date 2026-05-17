package rivetxsql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const updateTestTable = "update_test_data"

type UpdateTestData struct {
	Id        uint64 `db:"id" attr:"auto,primary"`
	Index     int    `db:"index_col"`
	Key       string `db:"key_col" size:"64"`
	NameId    int    `db:"name_id"`
	NameIndex int    `db:"name_index"`
}

func setupUpdateTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, err := rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + updateTestTable)
	if err != nil {
		t.Fatalf("drop update test table failed: %v", err)
	}

	query := `
CREATE TABLE ` + updateTestTable + ` (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	index_col INT NOT NULL,
	key_col VARCHAR(64) NOT NULL,
	name_id INT UNSIGNED NOT NULL,
	name_index INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	UNIQUE INDEX u_utd_key (index_col, key_col)
);`
	_, err = rivetxsql.Pool.Exec(query)
	if err != nil {
		t.Fatalf("create update test table failed: %v", err)
	}
}

func teardownUpdateTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, _ = rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + updateTestTable)
}

func queryUpdateTestRows(t *testing.T, rivetxsql *RivetxSql) []UpdateTestData {
	t.Helper()
	rows, err := rivetxsql.Pool.Query("SELECT index_col, key_col, name_id, name_index FROM " + updateTestTable + " ORDER BY index_col, key_col")
	if err != nil {
		t.Fatalf("query update test rows failed: %v", err)
	}
	defer rows.Close()

	result := make([]UpdateTestData, 0)
	for rows.Next() {
		var row UpdateTestData
		if err := rows.Scan(&row.Index, &row.Key, &row.NameId, &row.NameIndex); err != nil {
			t.Fatalf("scan update test row failed: %v", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration failed: %v", err)
	}
	return result
}

func insertUpdateTestRows(t *testing.T, rivetxsql *RivetxSql, rows []UpdateTestData) {
	t.Helper()
	if _, err := Insert(rivetxsql, updateTestTable, rows, 2, "", false, 10*time.Second); err != nil {
		t.Fatalf("insert update test rows failed: %v", err)
	}
}

func TestUpdateRaw_ErrorCases(t *testing.T) {
	_, err := UpdateRaw(nil, updateTestTable, nil, nil, nil, nil, 0, 0)
	if err == nil {
		t.Fatal("expected error when required params are empty")
	}

	_, err = UpdateRaw(nil, updateTestTable, []string{"index_col"}, [][]interface{}{{1}}, nil, []string{"u.name_id = v.name_id"}, 0, 0)
	if err == nil {
		t.Fatal("expected error when joinOn is empty")
	}

	_, err = UpdateRaw(nil, updateTestTable, []string{"index_col"}, [][]interface{}{}, []string{"index_col"}, []string{"u.name_id = v.name_id"}, 0, 0)
	if err == nil {
		t.Fatal("expected error when vals is empty")
	}

	_, err = UpdateRaw(nil, updateTestTable, []string{"index_col", "key_col"}, [][]interface{}{{1}}, []string{"index_col"}, []string{"u.name_id = v.name_id"}, 0, 0)
	if err == nil {
		t.Fatal("expected error when val row length does not match cols length")
	}
}

func TestUpdateRaw_BatchUpdatesRows(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupUpdateTestTable(t, rivetxsql)
	defer teardownUpdateTestTable(t, rivetxsql)

	initial := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 1, NameIndex: 1000},
		{Index: 0, Key: "xyz", NameId: 2, NameIndex: 2000},
		{Index: 1, Key: "def", NameId: 3, NameIndex: 3000},
	}
	insertUpdateTestRows(t, rivetxsql, initial)

	cols := []string{"index_col", "key_col", "name_id", "name_index"}
	vals := [][]interface{}{
		{0, "abc", 10, 10},
		{0, "xyz", 20, 20},
		{1, "def", 30, 30},
	}
	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	res, err := UpdateRaw(rivetxsql, updateTestTable, cols, vals, joinOn, setExpr, 2, 10*time.Second)
	if err != nil {
		t.Fatalf("UpdateRaw failed: %v", err)
	}
	if res.TotalAffected != int64(len(vals)) {
		t.Fatalf("expected TotalAffected %d, got %d", len(vals), res.TotalAffected)
	}

	rows := queryUpdateTestRows(t, rivetxsql)
	if len(rows) != len(initial) {
		t.Fatalf("expected %d rows, got %d", len(initial), len(rows))
	}

	expected := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 10, NameIndex: 1010},
		{Index: 0, Key: "xyz", NameId: 20, NameIndex: 2020},
		{Index: 1, Key: "def", NameId: 30, NameIndex: 3030},
	}

	for i, row := range rows {
		if row != expected[i] {
			t.Fatalf("row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}
}

func TestUpdate_WithStructSlice(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupUpdateTestTable(t, rivetxsql)
	defer teardownUpdateTestTable(t, rivetxsql)

	initial := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 1, NameIndex: 1000},
		{Index: 1, Key: "def", NameId: 2, NameIndex: 2000},
	}
	insertUpdateTestRows(t, rivetxsql, initial)

	updates := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 10, NameIndex: 10},
		{Index: 1, Key: "def", NameId: 20, NameIndex: 20},
	}
	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	res, err := Update(rivetxsql, updateTestTable, updates, joinOn, setExpr, 0, 10*time.Second)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if res.TotalAffected != int64(len(updates)) {
		t.Fatalf("expected TotalAffected %d, got %d", len(updates), res.TotalAffected)
	}

	rows := queryUpdateTestRows(t, rivetxsql)

	expected := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 10, NameIndex: 1010},
		{Index: 1, Key: "def", NameId: 20, NameIndex: 2020},
	}

	for i, row := range rows {
		if row != expected[i] {
			t.Fatalf("row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}
}

func TestUpdateBuilder_ExecWithPointerSlice(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupUpdateTestTable(t, rivetxsql)
	defer teardownUpdateTestTable(t, rivetxsql)

	initial := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 1, NameIndex: 1000},
		{Index: 1, Key: "def", NameId: 2, NameIndex: 2000},
	}
	insertUpdateTestRows(t, rivetxsql, initial)

	updates := []*UpdateTestData{
		{Index: 0, Key: "abc", NameId: 10, NameIndex: 10},
		{Index: 1, Key: "def", NameId: 20, NameIndex: 20},
	}
	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	res, err := NewUpdate(updateTestTable, updates).JoinOn(joinOn).SetExpr(setExpr).BatchSize(2).Timeout(10 * time.Second).Exec(rivetxsql)
	if err != nil {
		t.Fatalf("Update builder Exec failed: %v", err)
	}
	if res.TotalAffected != int64(len(updates)) {
		t.Fatalf("expected TotalAffected %d, got %d", len(updates), res.TotalAffected)
	}

	rows := queryUpdateTestRows(t, rivetxsql)

	expected := []UpdateTestData{
		{Index: 0, Key: "abc", NameId: 10, NameIndex: 1010},
		{Index: 1, Key: "def", NameId: 20, NameIndex: 2020},
	}

	for i, row := range rows {
		if row != expected[i] {
			t.Fatalf("row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}
}
