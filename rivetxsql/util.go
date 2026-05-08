package rivetxsql

import (
	"fmt"
	"github.com/yefy/log4go/ee"
	"reflect"
	"rivetxgo/rivetxcore/utilx"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const BatchSize = 1024
const Timeout = 15 * time.Second

// QueryCond 支持固定列 + IN 条件
type QueryCond struct {
	FixedCols   []string        // 固定列名，可为空
	FixedVals   []interface{}   // 固定列值，可为空
	InCols      []string        // IN 子句列名
	InVals      [][]interface{} // IN 条目
	InBatchSize int
}

// QueryStruct 支持结构体映射
type QueryStruct[F any, I any] struct {
	Fixed  *F  // 固定条件结构体，可为空 struct{}
	InVals []I // IN 条件结构体
}

// -----------------------------
// 1. 从结构体提取列和值
// -----------------------------

func GoTypeToSql(t reflect.Type, tagSize string) (string, error) {
	switch t.Kind() {
	case reflect.Uint, reflect.Uintptr, reflect.Uint64:
		return "BIGINT UNSIGNED", nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INT UNSIGNED", nil
	case reflect.Int, reflect.Int64:
		return "BIGINT", nil
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return "INT", nil
	case reflect.String:
		if tagSize == "TINYTEXT" || tagSize == "TEXT" || tagSize == "MEDIUMTEXT" || tagSize == "LONGTEXT" {
			return tagSize, nil
		} else {
			size := 255
			var err error
			if len(tagSize) > 0 {
				size, err = strconv.Atoi(tagSize)
				if err != nil {
					return "", ee.New(err, "")
				}
			}
			return fmt.Sprintf("VARCHAR(%d)", size), nil
		}
	case reflect.Bool:
		return "TINYINT(1)", nil
	case reflect.Struct:
		if strings.Contains(t.String(), "Time") {
			size, err := strconv.Atoi(tagSize)
			if err != nil || len(tagSize) <= 0 {
				return "DATETIME", nil
			} else {
				return fmt.Sprintf("DATETIME(%d)", size), nil
			}
		}
		return "", fmt.Errorf("unsupported struct type: %s", t.String())
	default:
		return "", ee.New(nil, "typ err")
	}
}

type structMeta struct {
	cols       []string
	fieldIndex []int
	sqlTypes   []string
	fixedAttr  []string

	discardAutoCols       []string
	discardAutoFieldIndex []int

	autoColMap map[string]bool
	primary    string
	indexMap   map[string]string
	uniqueMap  map[string][]string
}

var metaCache sync.Map // map[reflect.Type]*structMeta

func getStructMeta(t reflect.Type) (*structMeta, error) {
	if v, ok := metaCache.Load(t); ok {
		LogRivetxsql().Debug("find cache")
		return v.(*structMeta), nil
	}

	meta := &structMeta{
		cols:       make([]string, 0, t.NumField()),
		fieldIndex: make([]int, 0, t.NumField()),
		sqlTypes:   make([]string, 0, t.NumField()),

		discardAutoCols:       make([]string, 0, t.NumField()),
		discardAutoFieldIndex: make([]int, 0, t.NumField()),

		autoColMap: make(map[string]bool, t.NumField()),
		indexMap:   make(map[string]string, t.NumField()),
		uniqueMap:  make(map[string][]string, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		tag := field.Tag.Get("db")
		tag = utilx.StringTrim(tag)

		if tag == "-" {
			continue
		}

		if tag == "" {
			if !field.IsExported() {
				continue
			}
			tag = ToSnakeCase(field.Name)
		} else {
			if !field.IsExported() {
				return nil, ee.New(nil, "struct:%v field:%v not exported", t.Name(), field.Name)
			}
		}

		tag = strings.ReplaceAll(tag, "\n", "")
		tag = strings.ReplaceAll(tag, "\t", " ")

		meta.cols = append(meta.cols, tag)
		meta.fieldIndex = append(meta.fieldIndex, i)

		tagAttr := field.Tag.Get("attr")
		tagAttr = utilx.StringTrim(tagAttr)
		isAuto := false
		fixedAttr := ""
		if len(tagAttr) > 0 {
			attrs := strings.Split(tagAttr, ",")
			for _, attr := range attrs {
				attr = utilx.StringTrim(attr)
				if attr == "auto" {
					isAuto = true
					meta.autoColMap[tag] = true
				} else if attr == "primary" {
					if len(meta.primary) > 0 {
						return nil, ee.New(nil, "duplicate primary, tagAttr:%v", tagAttr)
					}
					meta.primary = tag
				} else {
					subAttrs := strings.Split(attr, ":")
					if len(subAttrs) == 1 {
						fixedAttr = subAttrs[0]
					} else {
						if len(subAttrs) != 2 {
							return nil, ee.New(nil, "tagAttr:%v", tagAttr)
						}
						if subAttrs[0] == "index" {
							meta.indexMap[subAttrs[1]] = tag
						} else if subAttrs[0] == "unique" {
							meta.uniqueMap[subAttrs[1]] = append(meta.uniqueMap[subAttrs[1]], tag)
						} else {
							return nil, ee.New(nil, "tagAttr:%v", tagAttr)
						}
					}
				}
			}
		}

		meta.fixedAttr = append(meta.fixedAttr, fixedAttr)
		if strings.Contains(fixedAttr, "DEFAULT") && strings.Contains(fixedAttr, "CURRENT_TIMESTAMP") {
			isAuto = true
		}

		if !isAuto {
			meta.discardAutoCols = append(meta.discardAutoCols, tag)
			meta.discardAutoFieldIndex = append(meta.discardAutoFieldIndex, i)
		}

		tagSize := field.Tag.Get("size")
		tagSize = utilx.StringTrim(tagSize)
		sqlType, err := GoTypeToSql(field.Type, tagSize)
		if err != nil {
			return nil, ee.New(nil, "")
		}
		meta.sqlTypes = append(meta.sqlTypes, sqlType)
	}

	metaCache.Store(t, meta)
	return meta, nil
}

func GetCurrentDBName(rivetxsql *RivetxSql) (string, error) {
	var dbName string
	// MySQL 使用 DATABASE() 函数
	err := rivetxsql.Pool.QueryRow("SELECT DATABASE()").Scan(&dbName)
	if err != nil {
		return "", err
	}
	return dbName, nil
}

var dbNameCache sync.Map // map[*sql.DB]string
func GetDbName(rivetxsql *RivetxSql) (string, error) {
	if val, ok := dbNameCache.Load(rivetxsql); ok {
		return val.(string), nil
	} else {
		dbName, err := GetCurrentDBName(rivetxsql)
		if err != nil {
			return "", ee.New(nil, "")
		}
		dbNameCache.Store(rivetxsql, dbName)
		return dbName, nil
	}
}

type AutoIncrementColumnName struct {
	Name string `db:"COLUMN_NAME as name"`
}

type AutoIncrementColumnsValue struct {
	M     map[string]bool
	Mutex sync.Mutex
}

var autoIncrementColumns sync.Map // map[dbName, tableName string]  map[string]bool
func GetAutoIncrementColumns(rivetxsql *RivetxSql, tableName string) (map[string]bool, error) {
	dbName, err := GetDbName(rivetxsql)
	if err != nil {
		return nil, ee.New(nil, "")
	}
	key := fmt.Sprintf("%s_%s", dbName, tableName)
	valueI, ok := autoIncrementColumns.Load(key)
	if !ok {
		valueI, ok = autoIncrementColumns.LoadOrStore(key, &AutoIncrementColumnsValue{})
	}

	value := valueI.(*AutoIncrementColumnsValue)
	value.Mutex.Lock()
	defer value.Mutex.Unlock()
	if value.M != nil {
		return value.M, nil
	}

	datas, err := NewSelect[AutoIncrementColumnName]("information_schema.COLUMNS").
		WhereEq("TABLE_SCHEMA", dbName).
		WhereEq("TABLE_NAME", tableName).
		WhereEq("EXTRA", "auto_increment").
		Exec(rivetxsql)
	if err != nil {
		return nil, ee.New(nil, "")
	}
	m := make(map[string]bool)
	for _, data := range datas {
		m[data.Name] = true
	}
	value.M = m
	return m, nil
}

func StructFields[T any]() ([]string, error) {
	meta, err := StructMeta[T]()
	if err != nil {
		return nil, err
	}

	return meta.cols, nil
}

func StructMeta[T any]() (*structMeta, error) {
	typ := reflect.TypeFor[T]()

	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, ee.New(nil, "err:input must be a struct or pointer to struct, got %v", typ.Kind())
	}

	meta, err := getStructMeta(typ)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func StructValues(meta *structMeta, v any) ([]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}
	values := make([]interface{}, len(meta.fieldIndex))
	for i, idx := range meta.fieldIndex {
		values[i] = val.Field(idx).Interface()
	}

	return values, nil
}

func StructValuesByDiscardAuto(meta *structMeta, v any) ([]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}
	values := make([]interface{}, len(meta.discardAutoFieldIndex))
	for i, idx := range meta.discardAutoFieldIndex {
		values[i] = val.Field(idx).Interface()
	}

	return values, nil
}

func StructFieldsAndValues(v any) ([]string, []interface{}, error) {
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
		return nil, nil, ee.New(nil, "must be struct")
	}

	meta, err := getStructMeta(typ)
	if err != nil {
		return nil, nil, err
	}

	values := make([]interface{}, len(meta.fieldIndex))
	for i, idx := range meta.fieldIndex {
		values[i] = val.Field(idx).Interface()
	}

	return meta.cols, values, nil
}

func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// 前一个字符不是下划线，且当前字符是大写，则添加下划线
			if i > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

type EstimateJoin struct {
	Parts []string
	Sep   string
}

// estimateJoinLen 计算字符串数组 join 后的长度（不创建新字符串）
func estimateJoinLen(data EstimateJoin) int {
	parts := data.Parts
	sep := data.Sep
	if len(parts) == 0 {
		return 0
	}
	sepLen := len(sep)
	total := (len(parts) - 1) * sepLen
	for _, p := range parts {
		total += len(p)
	}
	return total
}

func estimateStrLen(strs []string, strs2 []string, joins []EstimateJoin) int {
	size := 128
	for _, str := range strs {
		size += len(str) + 5
	}

	for _, str := range strs2 {
		size += len(str) + 5
	}

	for _, join := range joins {
		size += estimateJoinLen(join) + 5
	}

	return size
}

// BuildQuery 构建高性能 DELETE SELECT 语句
func BuildQuery(sqls []string, table string, join string, fixedConds []string, cond string, inCols []string, tuples []string, order string, limit string) string {
	// 估算长度，避免 strings.Join
	estLen := estimateStrLen(sqls, []string{table, join, cond, limit}, []EstimateJoin{{fixedConds, " AND "}, {inCols, ", "}, {tuples, ","}})
	estLen += 64

	var b strings.Builder
	b.Grow(estLen)

	for _, sql := range sqls {
		b.WriteString(sql)
		b.WriteString(" ")
	}
	b.WriteString(table)
	b.WriteString(" ")
	b.WriteString(join)
	b.WriteString(" WHERE")

	isFirstAdd := true
	writeAnd := func() {
		if isFirstAdd {
			isFirstAdd = false
			b.WriteString(" ")
		} else {
			b.WriteString(" AND ")
		}
	}

	// 写固定条件
	if len(fixedConds) > 0 {
		writeAnd()
		for i, c := range fixedConds {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString(c)
		}
	}

	// 写单一条件
	if len(cond) > 0 {
		writeAnd()
		b.WriteString(cond)
	}

	// 写 IN 条件
	if len(tuples) > 0 {
		writeAnd()
		b.WriteByte('(')
		for i, c := range inCols {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(c)
		}
		b.WriteString(") IN (")

		for i, t := range tuples {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(t)
		}
		b.WriteByte(')')
	}

	if len(order) > 0 {
		b.WriteString(" ")
		b.WriteString(order)
	}

	if len(limit) > 0 {
		b.WriteString(" ")
		b.WriteString(limit)
	}

	return b.String()
}
