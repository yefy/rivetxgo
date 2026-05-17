package rivetxsql

import (
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestSelete() error {
	err := TestSelectWithWherePoint()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestSelectWithWhere()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestSelectWithWhereJoin()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}

func TestSelectWithWherePoint() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	err = testDataClearTable(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	if testDataCountRows(rivetxsql, "test_data") != 0 {
		return ee.New(nil, "testDataCountRows(rivetxsql, \"test_data\") != 0")
	}

	// insert test data
	testData := []*TestData{
		{Index: 1, Key: "hex", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "abc", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "def", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "ghi", NameId: 103, NameIndex: 1003, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "xyz", NameId: 104, NameIndex: 1004, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "kyl", NameId: 105, NameIndex: 1005, CurrTime: time.Now().Truncate(time.Second)},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	if testDataCountRows(rivetxsql, "test_data") != len(testData) {
		return ee.New(nil, "testDataCountRows(rivetxsql, \"test_data\") != len(testData)")
	}

	{
		res1, err := NewSelect[*TestData]("test_data").WhereEq("1", 1).Order("order by index_col, key_col").Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != len(testData) {
			return ee.New(err, "len(res1) != len(testData)")
		}

		// verify result
		rows, err := TestDataQueryAll(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}

		if len(rows) != len(res1) {
			return ee.New(err, "len(rows) != len(res1)")
		}

		for i, row := range rows {
			if row != *res1[i] {
				return ee.New(err, "row %d mismatch: got %+v, want %+v", i, row, res1[i])
			}
		}
	}

	for i := 0; i < 20; i++ {
		limit := i
		log4.Info("OrderFieldSelect true index:%v", i)
		res1, err := NewSelect[TestData]("test_data").OrderFieldSelect("id", true, i).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}

		if limit > len(testData) {
			limit = len(testData)
		}

		if len(res1) != limit {
			return ee.New(err, "expected %v rows, got %d", limit, len(res1))
		}

		rows, err := TestDataQueryAllById(rivetxsql, true, i)
		if err != nil {
			return ee.New(err, "")
		}

		if len(rows) != len(res1) {
			return ee.New(err, "len(rows) != len(res1)")
		}

		for index, row := range rows {
			log4.Info("index:%v, res:%+v, row:%+v", index, res1[index], row)
		}

		for index, row := range rows {
			if row != res1[index] {
				return ee.New(err, "i:%v, row %d mismatch: got %+v, want %+v", i, index, row, res1[index])
			}
		}
	}

	for i := 0; i < 20; i++ {
		limit := i
		log4.Info("OrderFieldSelect false index:%v", i)
		res1, err := NewSelect[*TestData]("test_data").OrderFieldSelect("id", false, i).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}

		if limit > len(testData) {
			limit = len(testData)
		}

		if len(res1) != limit {
			return ee.New(err, "expected %v rows, got %d", limit, len(res1))
		}

		rows, err := TestDataQueryAllById(rivetxsql, false, i)
		if err != nil {
			return ee.New(err, "")
		}

		if len(rows) != len(res1) {
			return ee.New(err, "len(rows) != len(res1)")
		}

		for index, row := range rows {
			log4.Info("index:%v, res:%+v, row:%+v", index, res1[index], row)
		}

		for index, row := range rows {
			if row != *res1[index] {
				return ee.New(err, "i:%v, row %d mismatch: got %+v, want %+v", i, index, row, res1[index])
			}
		}
	}

	{
		res1, err := NewSelect[*TestDataNoExport]("test_data").WhereEq("index_col", 1).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 3 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	{
		LogRivetxsql().Info("test WhereIn")
		res1, err := NewSelect[*TestData]("test_data").WhereEq("index_col", 1).WhereIn([]string{"key_col"}, [][]interface{}{{"yyy"}, {"xxx"}}).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 0 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	{
		_, err := NewSelect[*TestData]("test_data").WhereEq("index_col", 1).WhereIn([]string{"key_col"}, [][]interface{}{}).Timeout(10 * time.Second).Exec(rivetxsql)
		if err == nil {
			return ee.New(err, "")
		} else {
			log4.Info("err:%v", err)
		}
	}

	{
		// -----------------------------
		// 1️⃣ single-table query, no IN
		// -----------------------------
		res1, err := SelectRaw[*TestData](rivetxsql, "test_data", "", QueryCond{
			FixedCols: []string{"index_col"},
			FixedVals: []interface{}{1},
		}, "", nil, "", 0, 0, 0, 10*time.Second)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 3 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	{
		res1, err := NewSelect[*TestData]("test_data").WhereEq("index_col", 1).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 3 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	{
		res1, err := NewSelect[*TestData]("test_data").Where("index_col = ?", []interface{}{1}).Where("and key_col = ?", []interface{}{"abc"}).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 1 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	{
		res1, err := NewSelect[*TestData]("test_data").Where("index_col = ? and key_col = ?", []interface{}{1, "abc"}).Timeout(10 * time.Second).Exec(rivetxsql)
		if err != nil {
			return ee.New(err, "")
		}
		log4.Info("res1:%+v", res1)
		for _, res := range res1 {
			log4.Info("res:%+v", res)
		}
		if len(res1) != 1 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	return nil
}

func TestSelectWithWhere() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	err = testDataClearTable(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	// insert test data
	testData := []TestData{
		{Index: 1, Key: "hex", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "abc", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "def", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "ghi", NameId: 103, NameIndex: 1003, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "xyz", NameId: 104, NameIndex: 1004, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "kyl", NameId: 105, NameIndex: 1005, CurrTime: time.Now().Truncate(time.Second)},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	{
		// -----------------------------
		// 1️⃣ single-table query, no IN
		// -----------------------------
		res1, err := SelectRaw[*TestData](rivetxsql, "test_data", "", QueryCond{
			FixedCols: []string{"index_col"},
			FixedVals: []interface{}{1},
		}, "", nil, "", 0, 0, 0, 10*time.Second)
		if err != nil {
			return ee.New(err, "")
		}
		if len(res1) != 3 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}
	}

	// -----------------------------
	// 1️⃣ single-table query, no IN
	// -----------------------------
	res1, err := SelectRaw[TestData](rivetxsql, "test_data", "", QueryCond{
		FixedCols: []string{"index_col"},
		FixedVals: []interface{}{1},
	}, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res1) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res1))
	}

	// -----------------------------
	// 2️⃣ single-table query, fixed columns + IN conditions
	// -----------------------------
	cond := QueryCond{
		FixedCols: []string{"index_col"},
		FixedVals: []interface{}{2},
		InCols:    []string{"key_col"},
		InVals:    [][]interface{}{{"ghi"}, {"xyz"}, {"kyl"}},
	}
	res2, err := SelectRaw[TestData](rivetxsql, "test_data", "", cond, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res2) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res2))
	}

	// -----------------------------
	// 3️⃣ query with struct conditions
	// -----------------------------
	type Fixed struct {
		Index int `db:"index_col"`
	}

	type InStruct struct {
		Key string `db:"key_col"`
	}
	condStruct := QueryStruct[Fixed, InStruct]{
		Fixed:  &Fixed{Index: 1},
		InVals: []InStruct{{Key: "abc"}, {Key: "def"}, {Key: "hex"}},
	}
	res3, err := Select[TestData](rivetxsql, "test_data", "", condStruct, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res3) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res3))
	}

	// -----------------------------
	// 4️⃣ verify paged reads
	// -----------------------------
	res4, err := SelectRaw[TestData](rivetxsql, "test_data", "", QueryCond{}, "1=1", nil, "", 0, 0, 0, 10*time.Second) // batchSize = 2
	if err != nil {
		return ee.New(err, "")
	}
	if len(res4) != 6 {
		return ee.New(err, "expected 6 rows, got %d", len(res4))
	}

	log4.Info("✅ rivetxsql select tests passed")

	return nil
}

func TestSelectWithWhereJoin() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	err = testDataClearTable(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	if err := testKeyCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	err = testKeyClearTable(rivetxsql)
	if err != nil {
		return ee.New(err, "")
	}

	// insert test data
	testData := []TestData{
		{Index: 1, Key: "hex", NameId: 100, NameIndex: 1000, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "abc", NameId: 101, NameIndex: 1001, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 1, Key: "def", NameId: 102, NameIndex: 1002, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "ghi", NameId: 103, NameIndex: 1003, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "xyz", NameId: 104, NameIndex: 1004, CurrTime: time.Now().Truncate(time.Second)},
		{Index: 2, Key: "kyl", NameId: 105, NameIndex: 1005, CurrTime: time.Now().Truncate(time.Second)},
	}
	_, err = Insert(rivetxsql, "test_data", testData, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	// insert test data
	testKey := []Testkey{
		{Index: 1, Key: "hex"},
		{Index: 1, Key: "abc"},
		{Index: 1, Key: "def"},
		{Index: 2, Key: "ghi"},
		{Index: 2, Key: "xyz"},
		{Index: 2, Key: "kyl"},
	}
	_, err = Insert(rivetxsql, "test_key", testKey, 2, "", false, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}

	// -----------------------------
	// 1️⃣ single-table query, no IN
	// -----------------------------
	join := "JOIN test_key k ON d.index_col = k.index_col AND d.key_col = k.key_col"
	res1, err := SelectRaw[TestDataByD](rivetxsql, "test_data d", join, QueryCond{
		FixedCols: []string{"d.index_col"},
		FixedVals: []interface{}{1},
	}, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res1) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res1))
	}

	log4.Info("res1:%+v", res1)

	{
		join := "JOIN test_key k ON d.index_col = k.index_col AND d.key_col = k.key_col"
		res1, err := SelectRaw[TestDataByAs](rivetxsql, "test_data d", join, QueryCond{
			FixedCols: []string{"d.index_col"},
			FixedVals: []interface{}{1},
		}, "", nil, "", 0, 0, 0, 10*time.Second)
		if err != nil {
			return ee.New(err, "")
		}
		if len(res1) != 3 {
			return ee.New(err, "expected 3 rows, got %d", len(res1))
		}

		log4.Info("res1:%+v", res1)
	}

	// -----------------------------
	// 2️⃣ single-table query, fixed columns + IN conditions
	// -----------------------------
	cond := QueryCond{
		FixedCols: []string{"d.index_col"},
		FixedVals: []interface{}{2},
		InCols:    []string{"d.key_col"},
		InVals:    [][]interface{}{{"ghi"}, {"xyz"}, {"kyl"}},
	}
	res2, err := SelectRaw[TestDataByD](rivetxsql, "test_data d", join, cond, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res2) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res2))
	}
	log4.Info("res2:%+v", res2)

	// -----------------------------
	// 3️⃣ query with struct conditions
	// -----------------------------

	type Fixed struct {
		Index int `db:"d.index_col"`
	}

	type InStruct struct {
		Key string `db:"d.key_col"`
	}
	condStruct := QueryStruct[Fixed, InStruct]{
		Fixed:  &Fixed{Index: 1},
		InVals: []InStruct{{Key: "abc"}, {Key: "def"}, {Key: "hex"}},
	}
	res3, err := Select[TestDataByD](rivetxsql, "test_data d", join, condStruct, "", nil, "", 0, 0, 0, 10*time.Second)
	if err != nil {
		return ee.New(err, "")
	}
	if len(res3) != 3 {
		return ee.New(err, "expected 3 rows, got %d", len(res3))
	}
	log4.Info("res3:%+v", res3)

	// -----------------------------
	// 4️⃣ verify paged reads
	// -----------------------------
	res4, err := SelectRaw[TestDataByD](rivetxsql, "test_data d", join, QueryCond{}, "1=1", nil, "", 0, 0, 0, 10*time.Second) // batchSize = 2
	if err != nil {
		return ee.New(err, "")
	}
	if len(res4) != 6 {
		return ee.New(err, "expected 6 rows, got %d", len(res4))
	}
	log4.Info("res4:%+v", res4)

	log4.Info("✅ rivetxsql select tests passed")

	return nil
}
