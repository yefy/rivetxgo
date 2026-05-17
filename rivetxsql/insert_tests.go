package rivetxsql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"time"
)

func TestInsert() error {
	err := TestBatchNewInsertStructPoint()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchNewInsertStruct()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchInsert()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchInsertStruct()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchInsert_NoDuplicateUpdate()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchInsertStruct_NoDuplicateUpdate()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}

// ----------------------
// test InsertRaw with manual columns+values
// ----------------------
func TestBatchInsert() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	cols := []string{"index_col", "key_col", "name_id", "name_index", "curr_time"}
	vals := [][]interface{}{
		{0, "abc", 1, 1001, time.Now().Truncate(time.Second)},
		{1, "abc", 2, 1002, time.Now().Truncate(time.Second)},
		{2, "abc", 3, 1003, time.Now().Truncate(time.Second)},
		{3, "xyz", 4, 1004, time.Now().Truncate(time.Second)},
		{4, "xyz", 5, 1005, time.Now().Truncate(time.Second)},
		{5, "xyz", 6, 1006, time.Now().Truncate(time.Second)},
		{6, "xyz", 7, 1007, time.Now().Truncate(time.Second)},
		{7, "xyz", 8, 1008, time.Now().Truncate(time.Second)},
		{8, "xyz", 9, 1009, time.Now().Truncate(time.Second)},
		{9, "xyz", 10, 1010, time.Now().Truncate(time.Second)},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"

	_, err = InsertRaw(rivetxsql, "test_data", cols, vals, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	// insert duplicate index_col,key_col again to trigger ON DUPLICATE KEY UPDATE
	valsDup := [][]interface{}{
		{0, "abc", 11, 11, time.Now().Truncate(time.Second)}, // name_id = 11, name_index += 11
	}
	_, err = InsertRaw(rivetxsql, "test_data", cols, valsDup, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	var nameId, nameIndex int
	_ = rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex)

	if nameId != 11 || nameIndex != 1012 { // 1001 + 10 = 1011
		return ee.New(err, "ON DUPLICATE KEY UPDATE failed, got name_id=%d, name_index=%d", nameId, nameIndex)
	}

	log4.Info("BatchInsert test passed ✅")
	return nil
}

// ----------------------
// test Insert
// ----------------------
func TestBatchInsertStruct() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	testData := []TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "abc", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 2, "abc", 3, 1003, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 3, "xyz", 4, 1004, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 4, "xyz", 5, 1005, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 5, "xyz", 6, 1006, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 6, "xyz", 7, 1007, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 7, "xyz", 8, 1008, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 8, "xyz", 9, 1009, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 9, "xyz", 10, 1010, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"

	_, err = Insert(rivetxsql, "test_data", testData, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	// test duplicate struct insert triggers ON DUPLICATE KEY UPDATE
	dataDup := []TestData{
		{0, 0, "abc", 11, 11, time.Now().Truncate(time.Second), time.Time{}, time.Time{}}, // name_id = 11, name_index += 11
	}
	_, err = Insert(rivetxsql, "test_data", dataDup, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	var nameId, nameIndex int
	_ = rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex)

	if nameId != 11 || nameIndex != 1012 {
		return ee.New(err, "ON DUPLICATE KEY UPDATE failed for struct, got name_id=%d, name_index=%d", nameId, nameIndex)
	}

	log4.Info("BatchInsertStruct test passed ✅")
	return nil
}

// ----------------------
// test InsertRaw without ON DUPLICATE KEY UPDATE
// ----------------------
func TestBatchInsert_NoDuplicateUpdate() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	cols := []string{"index_col", "key_col", "name_id", "name_index", "curr_time"}
	vals := [][]interface{}{
		{0, "abc", 1, 1001, time.Now().Truncate(time.Second)},
		{1, "abc", 2, 1002, time.Now().Truncate(time.Second)},
		{2, "abc", 3, 1003, time.Now().Truncate(time.Second)},
		{3, "xyz", 4, 1004, time.Now().Truncate(time.Second)},
		{4, "xyz", 5, 1005, time.Now().Truncate(time.Second)},
		{5, "xyz", 6, 1006, time.Now().Truncate(time.Second)},
		{6, "xyz", 7, 1007, time.Now().Truncate(time.Second)},
		{7, "xyz", 8, 1008, time.Now().Truncate(time.Second)},
		{8, "xyz", 9, 1009, time.Now().Truncate(time.Second)},
		{9, "xyz", 10, 1010, time.Now().Truncate(time.Second)},
	}

	onDuplicate := "" // omit ON DUPLICATE KEY UPDATE

	_, err = InsertRaw(rivetxsql, "test_data", cols, vals, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	log4.Info("BatchInsert without ON DUPLICATE KEY UPDATE passed ✅")
	return nil
}

// ----------------------
// test Insert without ON DUPLICATE KEY UPDATE
// ----------------------
func TestBatchInsertStruct_NoDuplicateUpdate() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	testData := []TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "abc", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 2, "abc", 3, 1003, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 3, "xyz", 4, 1004, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 4, "xyz", 5, 1005, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 5, "xyz", 6, 1006, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 6, "xyz", 7, 1007, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 7, "xyz", 8, 1008, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 8, "xyz", 9, 1009, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 9, "xyz", 10, 1010, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	onDuplicate := "" // omit ON DUPLICATE KEY UPDATE

	_, err = Insert(rivetxsql, "test_data", testData, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	log4.Info("BatchInsertStruct without ON DUPLICATE KEY UPDATE passed ✅")
	return nil
}

// ----------------------
// test NewInsert
// ----------------------
func TestBatchNewInsertStruct() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	testData := []TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "abc", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 2, "abc", 3, 1003, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 3, "xyz", 4, 1004, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 4, "xyz", 5, 1005, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 5, "xyz", 6, 1006, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 6, "xyz", 7, 1007, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 7, "xyz", 8, 1008, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 8, "xyz", 9, 1009, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 9, "xyz", 10, 1010, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"
	result, err := NewInsert("test_data", testData).BatchSize(2).OnDuplicateUpdate(onDuplicate).Timeout(10 * time.Second).Exec(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}
	log4.Info("BatchInsertStruct test result:%+v", result)

	var id int64
	_ = rivetxsql.Pool.QueryRow("SELECT id FROM test_data order by id DESC limit 1").Scan(&id)

	if result.LastInsertID != id {
		return ee.New(err, "result.LastInsertID:%v != id:%v ", result.LastInsertID, id)
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 || count != int(result.TotalAffected) {
		return ee.New(err, "expected 10 rows, got %d|%d", count, result.TotalAffected)
	}

	// verify result
	rows, err := TestDataQueryAllNoId(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	if len(rows) != len(testData) {
		return ee.New(err, "len(rows) != len(testData)")
	}

	for i, row := range rows {
		if row != testData[i] {
			return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, testData[i])
		}
	}

	// test duplicate struct insert triggers ON DUPLICATE KEY UPDATE
	dataDup := []TestData{
		{0, 0, "abc", 11, 11, time.Now().Truncate(time.Second), time.Time{}, time.Time{}}, // name_id = 11, name_index += 11
	}
	_, err = Insert(rivetxsql, "test_data", dataDup, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	var nameId, nameIndex int
	_ = rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex)

	if nameId != 11 || nameIndex != 1012 {
		return ee.New(err, "ON DUPLICATE KEY UPDATE failed for struct, got name_id=%d, name_index=%d", nameId, nameIndex)
	}

	log4.Info("BatchInsertStruct test passed ✅")
	return nil
}

// ----------------------
// test NewInsert
// ----------------------
func TestBatchNewInsertStructPoint() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	testData := []*TestData{
		{0, 0, "abc", 1, 1001, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 1, "abc", 2, 1002, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 2, "abc", 3, 1003, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 3, "xyz", 4, 1004, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 4, "xyz", 5, 1005, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 5, "xyz", 6, 1006, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 6, "xyz", 7, 1007, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 7, "xyz", 8, 1008, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 8, "xyz", 9, 1009, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
		{0, 9, "xyz", 10, 1010, time.Now().Truncate(time.Second), time.Time{}, time.Time{}},
	}

	onDuplicate := "name_id = VALUES(name_id), name_index = name_index + VALUES(name_index)"
	result, err := NewInsert("test_data", testData).BatchSize(2).OnDuplicateUpdate(onDuplicate).Timeout(10 * time.Second).Exec(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}
	log4.Info("BatchInsertStruct test result:%+v", result)

	var id int64
	_ = rivetxsql.Pool.QueryRow("SELECT id FROM test_data order by id DESC limit 1").Scan(&id)

	if result.LastInsertID != id {
		return ee.New(err, "result.LastInsertID:%v != id:%v ", result.LastInsertID, id)
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 || count != int(result.TotalAffected) {
		return ee.New(err, "expected 10 rows, got %d|%d", count, result.TotalAffected)
	}

	// verify result
	rows, err := TestDataQueryAllNoId(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	if len(rows) != len(testData) {
		return ee.New(err, "len(rows) != len(testData)")
	}

	for i, row := range rows {
		row.CurrTime = row.CurrTime.Truncate(time.Second)
		want := *testData[i]
		want.CurrTime = want.CurrTime.Truncate(time.Second)
		if row != want {
			return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, want)
		}
	}

	// test duplicate struct insert triggers ON DUPLICATE KEY UPDATE
	dataDup := []TestData{
		{0, 0, "abc", 11, 11, time.Now().Truncate(time.Second), time.Time{}, time.Time{}}, // name_id = 11, name_index += 11
	}
	_, err = Insert(rivetxsql, "test_data", dataDup, 2, onDuplicate, false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	var nameId, nameIndex int
	_ = rivetxsql.Pool.QueryRow("SELECT name_id, name_index FROM test_data WHERE index_col = 0 AND key_col = 'abc'").Scan(&nameId, &nameIndex)

	if nameId != 11 || nameIndex != 1012 {
		return ee.New(err, "ON DUPLICATE KEY UPDATE failed for struct, got name_id=%d, name_index=%d", nameId, nameIndex)
	}

	log4.Info("BatchInsertStruct test passed ✅")
	return nil
}
