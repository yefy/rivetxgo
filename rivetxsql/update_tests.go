package rivetxsql

import (
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

func TestUpdate() error {
	err := TestBatchUpdateStruct()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchUpdateStruct2()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchUpdateStruct2Point()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}

// ----------------------
// test Update
// ----------------------
func TestBatchUpdateStruct() error {
	currTime := time.Now().Truncate(time.Second)
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert initial data
	testData := []TestData{
		{0, 0, "abc", 1, 1000, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 2, 2000, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 3, 3000, currTime, time.Time{}, time.Time{}},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	// prepare update data
	updates := []TestData{
		{0, 0, "abc", 10, 10, currTime, time.Time{}, time.Time{}}, // name_id = 10, name_index += 10
		{0, 1, "xyz", 20, 20, currTime, time.Time{}, time.Time{}}, // name_id = 20, name_index += 20
		{0, 2, "def", 30, 30, currTime, time.Time{}, time.Time{}}, // name_id = 30, name_index += 30
	}

	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	// execute batch update
	_, err = Update(rivetxsql, "test_data", updates, joinOn, setExpr, 2, 0) // test batch processing
	if err != nil {
		return ee.New(err, "")
	}

	// verify result
	rows, err := TestDataQueryAllNoId(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	expected := []TestData{
		{0, 0, "abc", 10, 1010, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 20, 2020, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 30, 3030, currTime, time.Time{}, time.Time{}},
	}

	if len(rows) != len(expected) {
		return ee.New(err, "len(rows) != len(expected)")
	}

	for i, row := range rows {
		if row != expected[i] {
			return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}

	log4.Info("BatchUpdateStruct test passed ✅")
	return nil
}

func TestBatchUpdateStruct2() error {
	currTime := time.Now().Truncate(time.Second)
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert initial data
	testData := []TestData{
		{0, 0, "abc", 1, 1000, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 2, 2000, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 3, 3000, currTime, time.Time{}, time.Time{}},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	// prepare update data
	updates := []TestData{
		{0, 0, "abc", 10, 10, currTime, time.Time{}, time.Time{}}, // name_id = 10, name_index += 10
		{0, 1, "xyz", 20, 20, currTime, time.Time{}, time.Time{}}, // name_id = 20, name_index += 20
		{0, 2, "def", 30, 30, currTime, time.Time{}, time.Time{}}, // name_id = 30, name_index += 30
	}

	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	// execute batch update
	res, err := NewUpdate("test_data", updates).JoinOn(joinOn).SetExpr(setExpr).Exec(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}
	if res.TotalAffected != int64(len(updates)) {
		return ee.New(err, "res.TotalAffected != int64(len(updates))")
	}

	// verify result
	rows, err := TestDataQueryAllNoId(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	expected := []TestData{
		{0, 0, "abc", 10, 1010, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 20, 2020, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 30, 3030, currTime, time.Time{}, time.Time{}},
	}

	if len(rows) != len(expected) {
		return ee.New(err, "len(rows) != len(expected)")
	}

	for i, row := range rows {
		if row != expected[i] {
			return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}

	log4.Info("TestBatchUpdateStruct2 test passed ✅")
	return nil
}

func TestBatchUpdateStruct2Point() error {
	currTime := time.Now().Truncate(time.Second)
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert initial data
	testData := []*TestData{
		{0, 0, "abc", 1, 1000, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 2, 2000, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 3, 3000, currTime, time.Time{}, time.Time{}},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	// prepare update data
	updates := []*TestData{
		{0, 0, "abc", 10, 10, currTime, time.Time{}, time.Time{}}, // name_id = 10, name_index += 10
		{0, 1, "xyz", 20, 20, currTime, time.Time{}, time.Time{}}, // name_id = 20, name_index += 20
		{0, 2, "def", 30, 30, currTime, time.Time{}, time.Time{}}, // name_id = 30, name_index += 30
	}

	joinOn := []string{"index_col", "key_col"}
	setExpr := []string{"u.name_id = v.name_id", "u.name_index = u.name_index + v.name_index"}

	// execute batch update
	res, err := NewUpdate("test_data", updates).JoinOn(joinOn).SetExpr(setExpr).Exec(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}
	if res.TotalAffected != int64(len(updates)) {
		return ee.New(err, "res.TotalAffected != int64(len(updates))")
	}

	// verify result
	rows, err := TestDataQueryAllNoId(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	expected := []*TestData{
		{0, 0, "abc", 10, 1010, currTime, time.Time{}, time.Time{}},
		{0, 1, "xyz", 20, 2020, currTime, time.Time{}, time.Time{}},
		{0, 2, "def", 30, 3030, currTime, time.Time{}, time.Time{}},
	}

	if len(rows) != len(expected) {
		return ee.New(err, "len(rows) != len(expected)")
	}

	for i, row := range rows {
		if row != *expected[i] {
			return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, expected[i])
		}
	}

	log4.Info("TestBatchUpdateStruct2Point test passed ✅")
	return nil
}
