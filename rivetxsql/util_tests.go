package rivetxsql

import (
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"reflect"
	"time"
)

// TINYTEXT
// TEXT
// MEDIUMTEXT
// LONGTEXT
// size:"64" size:"TEXT"
type TestData struct {
	Id        uint64    `db:"id" attr:"auto,primary"`
	Index     int       `db:"index_col"  attr:"unique:u_td_ik,unique:u_td_in"`
	Key       string    `db:"key_col"    attr:"unique:u_td_ik"  size:"64"`
	NameId    int       `db:"name_id"    attr:"unique:u_td_in"`
	NameIndex int       `db:"name_index" attr:"index:i_td_name_index"`
	CurrTime  time.Time `db:"curr_time"`
	CreatedAt time.Time `db:"created_at" attr:"DEFAULT CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `db:"updated_at" attr:"DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (obj TestData) OrderFieldSelectValue() interface{} {
	return obj.Id
}

type TestDataNoExport struct {
	index     int
	Key       string `db:"key_col"`
	NameId    int    `db:"name_id"`
	nameIndex int    `db:"-"`
}

type TestDataByD struct {
	Index     int    `db:"d.index_col"`
	Key       string `db:"d.key_col"`
	NameId    int    `db:"d.name_id"`
	NameIndex int    `db:"d.name_index"`
}

type TestDataByAs struct {
	Index     int    `db:"d.index_col"`
	Key       string `db:"d.key_col"`
	NameId    int    `db:"d.name_id"`
	NameId2   int    `db:"d.name_id as name_id_2"`
	NameIndex int    `db:"d.name_index"`
}

type Testkey struct {
	Id        uint64    `db:"id" attr:"auto,primary"`
	Index     int       `db:"index_col"`
	Key       string    `db:"key_col"`
	CreatedAt time.Time `db:"created_at" attr:"DEFAULT CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `db:"updated_at" attr:"DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func testOpenRivetxSql() (*RivetxSql, error) {
	config := &Config{
		Url: "root:Yfygz@389@tcp(192.168.80.139:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local",
		//Url:             "root:Yfygz@389@tcp(192.168.192.139:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 100000,
		ConnMaxIdleTime: 100000,
	}
	return CreateRivetxSql(config)
}

// 删除表
func testKeyDropTable(rivetxsql *RivetxSql) {
	_, _ = rivetxsql.Pool.Exec("DROP TABLE test_key;")
}

// 创建测试表
func testKeyCreateTable(rivetxsql *RivetxSql) error {
	query := `
	CREATE TABLE IF NOT EXISTS test_key (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		index_col INT NOT NULL,
		key_col VARCHAR(64) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE INDEX u_tk_key (index_col, key_col)
	);`
	_, err := rivetxsql.Pool.Exec(query)
	return err
}

func testKeyClearTable(rivetxsql *RivetxSql) error {
	//_, err := NewDelete("test_key").WhereEq("1", 1).Exec(rivetxsql)
	//if err != nil {
	//	log4.Error("clearTestKeyTable, err:%v", err)
	//	return err
	//}
	//return nil
	_, err := rivetxsql.Pool.Exec("DELETE FROM test_key")
	return err
}

// 创建测试表
func testDataCreateTable(rivetxsql *RivetxSql) error {
	query := `
	CREATE TABLE IF NOT EXISTS test_data (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		index_col INT NOT NULL,
		key_col VARCHAR(64) NOT NULL,
		name_id INT UNSIGNED NOT NULL,
		name_index INT UNSIGNED NOT NULL,
		PRIMARY KEY (id),
		UNIQUE INDEX u_td_key (index_col, key_col),
        INDEX i_td_name_id (name_id)
	);`
	_, err := rivetxsql.Pool.Exec(query)
	return err
}

func testDataClearTable(rivetxsql *RivetxSql) error {
	//_, err := NewDelete("test_data").WhereEq("1", 1).Exec(rivetxsql)
	//if err != nil {
	//	log4.Error("testDataClearTable, err:%v", err)
	//	return err
	//}
	//return nil
	testDataTruncateTable(rivetxsql)
	return nil
}

// 清空表
func testDataTruncateTable(rivetxsql *RivetxSql) {
	_, _ = rivetxsql.Pool.Exec("TRUNCATE TABLE test_data;")
}

// 删除表
func testDataDropTable(rivetxsql *RivetxSql) {
	_, _ = rivetxsql.Pool.Exec("DROP TABLE test_data;")
}

//type TableCount struct {
//	TotalCount int `db:"COUNT(*) as total_count"`
//}

// 查询表行数
func testDataCountRows(rivetxsql *RivetxSql, tableName string) int {
	/*
		rets, err := NewSelect[TableCount](tableName).Where("1=1", nil).Exec(rivetxsql)
		if err != nil {
			log4.Error("testDataCountRows tableName:%v, err:%v", tableName, err)
			return 0
		}
		if len(rets) <= 0 {
			return 0
		}
		return rets[0].TotalCount

	*/
	var count int
	err := rivetxsql.Pool.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
	if err != nil {
		log4.Error("testDataCountRows tableName:%v, err:%v", tableName, err)
		return 0
	}
	return count
}

// 查询表内容
func TestDataQueryAll(rivetxsql *RivetxSql) ([]TestData, error) {
	sql := "SELECT id, index_col, key_col, name_id, name_index, curr_time, created_at, updated_at FROM test_data ORDER BY index_col, key_col"
	log4.Info("sql:%v", sql)
	rows, err := rivetxsql.Pool.Query(sql)
	if err != nil {
		return nil, ee.New(err, "")
	}
	defer rows.Close()

	var result []TestData
	for rows.Next() {
		var td TestData
		if err := rows.Scan(&td.Id, &td.Index, &td.Key, &td.NameId, &td.NameIndex, &td.CurrTime, &td.CreatedAt, &td.UpdatedAt); err != nil {
			return nil, ee.New(err, "")
		}
		result = append(result, td)
	}

	return result, nil
}

func TestDataQueryAllNoId(rivetxsql *RivetxSql) ([]TestData, error) {
	sql := "SELECT index_col, key_col, name_id, name_index, curr_time FROM test_data ORDER BY index_col, key_col"
	log4.Info("sql:%v", sql)
	rows, err := rivetxsql.Pool.Query(sql)
	if err != nil {
		return nil, ee.New(err, "")
	}
	defer rows.Close()

	var result []TestData
	for rows.Next() {
		var td TestData
		if err := rows.Scan(&td.Index, &td.Key, &td.NameId, &td.NameIndex, &td.CurrTime); err != nil {
			return nil, ee.New(err, "")
		}
		result = append(result, td)
	}

	return result, nil
}

func TestDataQueryAllById(rivetxsql *RivetxSql, isDesc bool, limit int) ([]TestData, error) {
	order := ""
	if isDesc {
		order = "DESC"
	}
	sql := fmt.Sprintf("SELECT id, index_col, key_col, name_id, name_index, curr_time, created_at, updated_at  FROM test_data ORDER BY id %v limit %v", order, limit)
	log4.Info("sql:%v", sql)
	rows, err := rivetxsql.Pool.Query(sql)
	if err != nil {
		return nil, ee.New(err, "")
	}
	defer rows.Close()

	var result []TestData
	for rows.Next() {
		var td TestData
		if err := rows.Scan(&td.Id, &td.Index, &td.Key, &td.NameId, &td.NameIndex, &td.CurrTime, &td.CreatedAt, &td.UpdatedAt); err != nil {
			return nil, ee.New(err, "")
		}
		result = append(result, td)
	}

	return result, nil
}

func StructFieldsAndValues2(v any) ([]string, []interface{}, error) {
	if v == nil {
		return nil, nil, nil
	}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil, nil
		}

		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil, ee.New(nil, "err:input must be a struct or pointer to struct, got %v", val.Kind())
	}

	fields := make([]string, 0, typ.NumField())
	values := make([]interface{}, 0, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		// 获取db标签
		tag := f.Tag.Get("rivetxsql")

		// 如果标签为空，使用字段名的蛇形命名
		if tag == "" {
			if !f.IsExported() {
				continue
			}
			tag = ToSnakeCase(f.Name)
		} else {
			if !f.IsExported() {
				return nil, nil, ee.New(nil, "err:structName:%v, field:%v !f.IsExported()", typ.Name(), f.Name)
			}
		}

		// 跳过带有"-"标签的字段
		if tag == "-" {
			continue
		}

		fields = append(fields, tag)
		values = append(values, val.Field(i).Interface())
	}
	return fields, values, nil
}

func StructFields1(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, ee.New(nil, "must be struct")
	}

	meta, err := getStructMeta(typ)
	if err != nil {
		return nil, err
	}

	return meta.cols, nil
}

func StructFields2(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}

	typ := reflect.TypeOf(v)

	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, ee.New(nil, "must be struct")
	}

	meta, err := getStructMeta(typ)
	if err != nil {
		return nil, err
	}

	return meta.cols, nil
}

func StructMeta1(v any) (*structMeta, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, ee.New(nil, "must be struct")
	}

	meta, err := getStructMeta(typ)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
