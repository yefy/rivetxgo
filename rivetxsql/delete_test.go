package rivetxsql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const deleteTestTable = "delete_test_data"

func setupDeleteTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()

	_, err := rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + deleteTestTable)
	if err != nil {
		t.Fatalf("drop table failed: %v", err)
	}

	query := `
CREATE TABLE ` + deleteTestTable + ` (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	index_col INT NOT NULL,
	key_col VARCHAR(64) NOT NULL,
	name_id INT UNSIGNED NOT NULL,
	name_index INT UNSIGNED NOT NULL,
	curr_time DATETIME NOT NULL,
	PRIMARY KEY (id)
);`
	_, err = rivetxsql.Pool.Exec(query)
	if err != nil {
		t.Fatalf("create delete test table failed: %v", err)
	}
}

func teardownDeleteTestTable(t *testing.T, rivetxsql *RivetxSql) {
	t.Helper()
	_, _ = rivetxsql.Pool.Exec("DROP TABLE IF EXISTS " + deleteTestTable)
}

func countDeleteTestRows(t *testing.T, rivetxsql *RivetxSql) int {
	t.Helper()
	var count int
	err := rivetxsql.Pool.QueryRow("SELECT COUNT(*) FROM " + deleteTestTable).Scan(&count)
	if err != nil {
		t.Fatalf("count rows failed: %v", err)
	}
	return count
}

func TestDeleteRaw_ErrorCases(t *testing.T) {
	_, err := DeleteRaw(nil, deleteTestTable, QueryCond{
		FixedCols: []string{"index_col"},
		FixedVals: []interface{}{},
	}, "", nil, 0, 0)
	if err == nil {
		t.Fatal("expected error when fixed cols and fixed vals length mismatch")
	}

	_, err = DeleteRaw(nil, deleteTestTable, QueryCond{
		InCols: []string{"name_id", "name_index"},
		InVals: [][]interface{}{{1}},
	}, "", nil, 0, 0)
	if err == nil {
		t.Fatal("expected error when InVals length does not match InCols length")
	}

	_, err = DeleteRaw(nil, deleteTestTable, QueryCond{}, "", nil, 0, 0)
	if err == nil {
		t.Fatal("expected error when fixed cols, in cols, and cond are all empty")
	}

	_, err = DeleteRaw(nil, deleteTestTable, QueryCond{
		InCols:      []string{"name_id"},
		InVals:      [][]interface{}{{1}, {2}},
		InBatchSize: 1,
	}, "", nil, 1, 0)
	if err == nil {
		t.Fatal("expected error when len(InVals) > InBatchSize and limit > 0")
	}
}

func TestDeleteRaw_FixedInConditionsDeletesRows(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupDeleteTestTable(t, rivetxsql)
	defer teardownDeleteTestTable(t, rivetxsql)

	_, err = rivetxsql.Pool.Exec(
		"INSERT INTO "+deleteTestTable+" (index_col, key_col, name_id, name_index, curr_time) VALUES (?,?,?,?,?),(?,?,?,?,?)",
		0, "abc", 1, 1001, time.Now().Truncate(time.Second),
		0, "def", 2, 1002, time.Now().Truncate(time.Second),
	)
	if err != nil {
		t.Fatalf("insert test data failed: %v", err)
	}

	group := QueryCond{
		FixedCols: []string{"index_col"},
		FixedVals: []interface{}{0},
		InCols:    []string{"name_id", "name_index"},
		InVals: [][]interface{}{
			{1, 1001},
			{2, 1002},
		},
	}

	res, err := DeleteRaw(rivetxsql, deleteTestTable, group, "", nil, 0, 0)
	if err != nil {
		t.Fatalf("DeleteRaw failed: %v", err)
	}
	if res.TotalAffected != 2 {
		t.Fatalf("expected TotalAffected 2, got %d", res.TotalAffected)
	}

	count := countDeleteTestRows(t, rivetxsql)
	if count != 0 {
		t.Fatalf("expected 0 rows left, got %d", count)
	}
}

func TestDelete_UsingQueryStruct(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupDeleteTestTable(t, rivetxsql)
	defer teardownDeleteTestTable(t, rivetxsql)

	_, err = rivetxsql.Pool.Exec(
		"INSERT INTO "+deleteTestTable+" (index_col, key_col, name_id, name_index, curr_time) VALUES (?,?,?,?,?),(?,?,?,?,?)",
		0, "abc", 1, 1001, time.Now().Truncate(time.Second),
		0, "def", 2, 1002, time.Now().Truncate(time.Second),
	)
	if err != nil {
		t.Fatalf("insert test data failed: %v", err)
	}

	type Fixed struct {
		Index int    `db:"index_col"`
		Key   string `db:"key_col"`
	}
	type In struct {
		NameId    int `db:"name_id"`
		NameIndex int `db:"name_index"`
	}

	res, err := Delete(rivetxsql, deleteTestTable, QueryStruct[Fixed, In]{
		Fixed:  &Fixed{Index: 0, Key: "abc"},
		InVals: []In{{NameId: 1, NameIndex: 1001}},
	}, "", nil, 0)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if res.TotalAffected != 1 {
		t.Fatalf("expected TotalAffected 1, got %d", res.TotalAffected)
	}

	count := countDeleteTestRows(t, rivetxsql)
	if count != 1 {
		t.Fatalf("expected 1 row left, got %d", count)
	}
}

func TestDeleteBuilder_Limit(t *testing.T) {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		t.Skipf("skipping test because test DB is unavailable: %v", err)
	}
	defer rivetxsql.Pool.Close()

	setupDeleteTestTable(t, rivetxsql)
	defer teardownDeleteTestTable(t, rivetxsql)

	_, err = rivetxsql.Pool.Exec(
		"INSERT INTO "+deleteTestTable+" (index_col, key_col, name_id, name_index, curr_time) VALUES (?,?,?,?,?),(?,?,?,?,?)",
		0, "abc", 1, 1001, time.Now().Truncate(time.Second),
		0, "def", 2, 1002, time.Now().Truncate(time.Second),
	)
	if err != nil {
		t.Fatalf("insert test data failed: %v", err)
	}

	res, err := NewDelete(deleteTestTable).
		WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{1, 1001}, {2, 1002}}).
		Limit(1).
		Timeout(5 * time.Second).
		Exec(rivetxsql)
	if err != nil {
		t.Fatalf("NewDelete Exec failed: %v", err)
	}
	if res.TotalAffected != 1 {
		t.Fatalf("expected TotalAffected 1, got %d", res.TotalAffected)
	}

	count := countDeleteTestRows(t, rivetxsql)
	if count != 1 {
		t.Fatalf("expected 1 row left, got %d", count)
	}
}
