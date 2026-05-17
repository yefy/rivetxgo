package rivetxsql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"time"
)

func TestDetele() error {
	err := TestBatchDeletePerGroup()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchDeletePerGroupStruct()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchDeletePerGroupStruct2()
	if err != nil {
		return ee.New(err, "")
	}
	err = TestBatchDeletePerGroupStruct2Point()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchDeletePerGroupStruct2PointLimit()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestBatchDeletePerGroupStruct2PointReserve()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}

func TestBatchDeletePerGroup() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert data
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
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	// build delete group
	groups := []QueryCond{
		{ //1
			FixedCols: []string{"index_col", "key_col"},
			FixedVals: []interface{}{0, "abc"},
			InCols:    []string{"name_id", "name_index"},
			InVals: [][]interface{}{
				{1, 1001}, {2, 1002}, {3, 1003}, {4, 1004}, {5, 1005},
			},
		},
		{ //5
			FixedCols: nil,
			FixedVals: nil,
			InCols:    []string{"name_id", "name_index"},
			InVals: [][]interface{}{
				{6, 1006}, {7, 1007}, {8, 1008}, {9, 1009}, {10, 1010},
			},
		},
		{ //0
			FixedCols: []string{"index_col", "key_col"},
			FixedVals: []interface{}{1, "xyz"},
			InCols:    nil,
			InVals:    nil,
		},
	}

	for _, group := range groups {
		_, err = DeleteRaw(rivetxsql, "test_data", group, "", nil, 0, 0)
		if err != nil {
			return ee.New(err, "")
		}
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 4 {
		return ee.New(err, "expected 4 rows left, got %d", count)
	}

	log4.Info("BatchDeletePerGroup test passed ✅")
	return nil
}

func TestBatchDeletePerGroupStruct() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert data
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
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	// define struct
	type Fixed struct {
		Index int    `db:"index_col"`
		Key   string `db:"key_col"`
	}
	type In struct {
		NameId    int `db:"name_id"`
		NameIndex int `db:"name_index"`
	}

	// build delete group
	groups := []QueryStruct[Fixed, In]{
		{ //1
			Fixed: &Fixed{Index: 0, Key: "abc"},
			InVals: []In{
				{NameId: 1, NameIndex: 1001},
				{NameId: 2, NameIndex: 1002},
			},
		},
		{ //2
			Fixed: nil,
			InVals: []In{
				{NameId: 4, NameIndex: 1004},
				{NameId: 5, NameIndex: 1005},
			},
		},
		{ //0
			Fixed:  &Fixed{Index: 1, Key: "xyz"},
			InVals: nil,
		},
	}
	for _, group := range groups {
		_, err = Delete(rivetxsql, "test_data", group, "", nil, 0)
		if err != nil {
			return ee.New(err, "")
		}
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 7 {
		return ee.New(err, "expected 7 rows left, got %d", count)
	}

	log4.Info("BatchDeletePerGroupStruct test passed ✅")

	return nil
}

func TestBatchDeletePerGroupStruct2() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert data
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
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	{
		res, err := NewDelete("test_data").WhereEq("index_col", 0).
			WhereEq("key_col", "abc").
			WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{1, 1001}, {2, 1002}}).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 1 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 9 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	{
		res, err := NewDelete("test_data").WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{4, 1004}, {5, 1005}}).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 2 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 7 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	{
		res, err := NewDelete("test_data").WhereEq("index_col", 1).WhereEq("key_col", "xyz").Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 0 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 7 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	log4.Info("BatchDeletePerGroupStruct2 test passed ✅")

	return nil
}

func TestBatchDeletePerGroupStruct2Point() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert data
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
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	{
		res, err := NewDelete("test_data").WhereEq("index_col", 0).
			WhereEq("key_col", "abc").
			WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{1, 1001}, {2, 1002}}).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 1 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 9 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	{
		res, err := NewDelete("test_data").WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{4, 1004}, {5, 1005}}).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 2 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 7 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	{
		res, err := NewDelete("test_data").WhereEq("index_col", 1).WhereEq("key_col", "xyz").Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 0 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 7 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	log4.Info("BatchDeletePerGroupStruct2 test passed ✅")

	return nil
}

func TestBatchDeletePerGroupStruct2PointLimit() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataTruncateTable(rivetxsql)

	// insert data
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
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
		return ee.New(err, "expected 10 rows, got %d", count)
	}

	{
		res, err := NewDelete("test_data").WhereEq("index_col", 0).
			WhereEq("key_col", "abc").
			WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{1, 1001}, {2, 1002}}).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 1 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 9 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	{
		res, err := NewDelete("test_data").WhereIn([]string{"name_id", "name_index"}, [][]interface{}{{4, 1004}, {5, 1005}}).Limit(1).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		if res.TotalAffected != 1 {
			return ee.New(err, "res.TotalAffected != 1")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 8 {
			return ee.New(err, "expected 7 rows left, got %d", count)
		}
	}

	log4.Info("TestBatchDeletePerGroupStruct2PointLimit test passed ✅")

	return nil
}

func TestBatchDeletePerGroupStruct2PointReserve() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	for i := 0; i < 20; i++ {
		if err := testDataCreateTable(rivetxsql); err != nil {
			return ee.New(err, "")
		}
		testDataTruncateTable(rivetxsql)

		// insert data
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
		_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
		if err != nil {
			return ee.New(err, "")
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != 10 {
			return ee.New(err, "expected 10 rows, got %d", count)
		}

		reserveSize := i
		res, err := NewDelete("test_data").ReserveSize("id", reserveSize, 10*time.Millisecond).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}

		if reserveSize > len(testData) {
			reserveSize = len(testData)
		}

		if res.TotalAffected != int64(len(testData)-reserveSize) {
			return ee.New(err, "res.TotalAffected:%v != int64(len(testData):%v - reserveSize:%v)", res.TotalAffected, len(testData), reserveSize)
		}

		if count := testDataCountRows(rivetxsql, "test_data"); count != reserveSize {
			return ee.New(err, "expected %v rows left, got %d", reserveSize, count)
		}

		testData = testData[len(testData)-reserveSize : len(testData)]
		// verify result
		rows, err := TestDataQueryAllNoId(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}

		if len(rows) != len(testData) {
			return ee.New(err, "len(rows) != len(testData)")
		}

		for i, row := range rows {
			if row != *testData[i] {
				return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, testData[i])
			}
		}
	}

	log4.Info("TestBatchDeletePerGroupStruct2PointReserve test passed ✅")

	return nil
}
