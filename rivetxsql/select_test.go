package rivetxsql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const selectTestTable = "select_test_data"

func setupSelectTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, err := rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + selectTestTable)
	if err != nil {
		t.Fatalf("drop select test table failed: %v", err)
	}

	query := `
CREATE TABLE ` + selectTestTable + ` (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	index_col INT NOT NULL,
	key_col VARCHAR(64) NOT NULL,
	name_id INT UNSIGNED NOT NULL,
	name_index INT UNSIGNED NOT NULL,
	curr_time DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE INDEX u_std_key (index_col, key_col)
);`
	_, err = rivetxsql.Pool.Exec(query)
	if err != nil {
		t.Fatalf("create select test table failed: %v", err)
	}
}

func teardownSelectTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, _ = rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + selectTestTable)
}

func countSelectTestRows(t *testing.T, rivetxsql *RivetxSql) int {
	t.Helper()
	var count int
	if err := rivetxsql.Pool.QueryRow("SELECT COUNT(*) FROM " + selectTestTable).Scan(&count); err != nil {
		t.Fatalf("count select test rows failed: %v", err)
	}
	return count
}

func insertSelectTestRows(t *testing.T, rivetxsql *RivetxSql, rows []TestData) {
	t.Helper()
	if _, err := Insert(rivetxsql, selectTestTable, rows, 2, "", false, 10*time.Second); err != nil {
		t.Fatalf("insert select test rows failed: %v", err)
	}
}

func TestSelectRaw_FixedConditionReturnsMatchingRows(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupSelectTestTable(t, rivetxsql)
	defer teardownSelectTestTable(t, rivetxsql)

	testData := []TestData{
		{Index: 1, Key: "abc", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "def", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "ghi", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
	}
	insertSelectTestRows(t, rivetxsql, testData)

	res, err := SelectRaw[*TestData](rivetxsql, selectTestTable, "", QueryCond{
		FixedCols: []string{"index_col"},
		FixedVals: []interface{}{1},
	}, "", nil, "ORDER BY key_col", 0, 0, 0, 10*time.Second)
	if err != nil {
		t.Fatalf("SelectRaw failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(res))
	}
	for _, row := range res {
		if row.Index != 1 {
			t.Fatalf("expected index_col 1, got %d", row.Index)
		}
	}
	if count := countSelectTestRows(t, rivetxsql); count != 3 {
		t.Fatalf("expected 3 total rows, got %d", count)
	}
}

func TestSelect_StructQueryWithInValues(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupSelectTestTable(t, rivetxsql)
	defer teardownSelectTestTable(t, rivetxsql)

	testData := []TestData{
		{Index: 1, Key: "abc", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "xyz", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "kyl", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
	}
	insertSelectTestRows(t, rivetxsql, testData)

	queryStruct := QueryStruct[struct {
		Index int `db:"index_col"`
	}, struct {
		Key string `db:"key_col"`
	}]{
		Fixed: &struct {
			Index int `db:"index_col"`
		}{Index: 2},
		InVals: []struct {
			Key string `db:"key_col"`
		}{
			{Key: "xyz"},
			{Key: "kyl"},
		},
	}

	res, err := Select[*TestData](rivetxsql, selectTestTable, "", queryStruct, "", nil, "ORDER BY key_col", 0, 0, 0, 10*time.Second)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(res))
	}
	for _, row := range res {
		if row.Index != 2 {
			t.Fatalf("expected index_col 2, got %d", row.Index)
		}
	}
}

func TestSelectBuilder_OrderFieldSelect(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Close()

	setupSelectTestTable(t, rivetxsql)
	defer teardownSelectTestTable(t, rivetxsql)

	testData := []TestData{
		{Index: 1, Key: "abc", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "def", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "ghi", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
	}
	insertSelectTestRows(t, rivetxsql, testData)

	res, err := NewSelect[TestData](selectTestTable).OrderFieldSelect("id", false, 2).Timeout(10 * time.Second).Exec(rivetxsql)
	if err != nil {
		t.Fatalf("SelectBuilder Exec failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(res))
	}
	if res[0].Id >= res[1].Id {
		t.Fatalf("expected ascending ids, got %d and %d", res[0].Id, res[1].Id)
	}
}
