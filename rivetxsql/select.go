package rivetxsql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"math"
	"reflect"
	"strings"
	"time"
)

func scanRow[T any](rows *sql.Rows, columns []string) (T, error) {
	var t T
	typ := reflect.TypeFor[T]()

	// 1. determine the underlying struct type
	isPtr := false
	structTyp := typ
	if typ.Kind() == reflect.Pointer {
		isPtr = true
		structTyp = typ.Elem()
	}

	// verify the type is a struct
	if structTyp.Kind() != reflect.Struct {
		return t, fmt.Errorf("expected struct, got %v", structTyp.Kind())
	}

	// 2. prepare an instance: regardless of T, we ultimately need to manipulate struct fields
	var val reflect.Value
	if isPtr {
		// if T is *User, create a new User instance
		// reflect.New returns a pointer to User (*User)
		newObjPtr := reflect.New(structTyp)
		val = newObjPtr.Elem()        // get the User instance to manipulate fields
		t = newObjPtr.Interface().(T) // assign *User to return variable t
	} else {
		// if T is User, use &t directly to operate
		// note: pass the Value of &t so Elem() produces an addressable val
		val = reflect.ValueOf(&t).Elem()
	}

	// 3. get metadata (includes column-to-field index mapping)
	meta, err := getStructMeta(structTyp)
	if err != nil {
		return t, err
	}

	if len(columns) != len(meta.fieldIndex) {
		return t, ee.New(nil, "len(columns):%v != len(meta.fieldIndex):%v", len(columns), len(meta.fieldIndex))
	}

	// prepare scan targets for each column
	dest := make([]interface{}, len(columns))
	for i, _ := range columns {
		idx := meta.fieldIndex[i]
		dest[i] = val.Field(idx).Addr().Interface()
	}

	err = rows.Scan(dest...)
	return t, err
}

// SelectRaw supports paging, fixed columns, and IN conditions
func SelectRaw[T any](rivetxsql *RivetxSql, table string, join string, queryCond QueryCond,
	cond string, condArgs []interface{}, order string, limit int, offset int, batchSize int,
	timeout time.Duration) ([]T, error) {

	if len(order) > 0 || limit > 0 || offset > 0 {
		if queryCond.InBatchSize > 0 {
			return nil, ee.New(nil, "(len(order) > 0  || limit > 0|| offset > 0) not supper inBatchSize")
		}
	}

	if len(order) <= 0 && limit <= 0 && offset <= 0 {
		if queryCond.InBatchSize <= 0 {
			queryCond.InBatchSize = BatchSize
		}
	}

	if queryCond.InBatchSize <= 0 {
		queryCond.InBatchSize = math.MaxInt32
	}

	if timeout == 0 {
		timeout = Timeout
	}

	if limit <= 0 {
		limit = math.MaxInt32
	}

	if batchSize <= 0 {
		batchSize = BatchSize
	}

	if len(queryCond.FixedCols) != len(queryCond.FixedVals) {
		return nil, ee.New(nil, "fixedCols and fixedVals length mismatch")
	}

	if len(queryCond.InCols) > 0 {
		if len(queryCond.InVals) == 0 {
			return nil, ee.New(nil, "len(queryCond.InCols) > 0 && len(queryCond.InVals) == 0")
		} else {
			if len(queryCond.InCols) != len(queryCond.InVals[0]) {
				return nil, ee.New(nil, "len(queryCond.InCols) != len(queryCond.InVals[0])")
			}
		}
	}

	// verify IN values have consistent column count
	for i, vals := range queryCond.InVals {
		if len(vals) != len(queryCond.InCols) {
			return nil, ee.New(nil, "InVals[%d] length %d does not match InCols length %d", i, len(vals), len(queryCond.InCols))
		}
	}

	if len(queryCond.FixedCols) <= 0 && len(queryCond.InCols) <= 0 && len(cond) <= 0 {
		return nil, ee.New(nil, "both FixedCols and InCols and cond are empty")
	}

	chunksSize := 1
	if len(queryCond.InVals) > 0 {
		chunksSize = (len(queryCond.InVals) + queryCond.InBatchSize - 1) / queryCond.InBatchSize
	}
	chunks := make([][][]interface{}, 0, chunksSize)
	if len(queryCond.InVals) > 0 {
		for start := 0; start < len(queryCond.InVals); start += queryCond.InBatchSize {
			end := start + queryCond.InBatchSize
			if end > len(queryCond.InVals) {
				end = len(queryCond.InVals)
			}
			chunks = append(chunks, queryCond.InVals[start:end])
		}
	} else {
		chunks = append(chunks, [][]interface{}{})
	}

	fields, err := StructFields[T]()
	if err != nil {
		return nil, ee.New(err, "StructFields")
	}
	result := make([]T, 0, 1024)

	startTime := time.Now()
	totalCount := 0
	for chunkIndex, chunk := range chunks {
		dataOffset := offset
		dataLimit := limit
		for {
			isBreak, err := func() (bool, error) {
				if dataLimit <= 0 {
					return true, nil
				}

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				// build IN ((?, ?), ...)
				chunkSize := len(chunk) + len(condArgs)
				if chunkSize <= 0 {
					chunkSize = 1
				}
				tuples := make([]string, 0, chunkSize)
				args := make([]interface{}, 0, len(queryCond.FixedVals)+chunkSize*len(queryCond.InCols))

				if len(queryCond.FixedCols) > 0 {
					args = append(args, queryCond.FixedVals...)
				}

				if len(condArgs) > 0 {
					args = append(args, condArgs...)
				}

				for _, vals := range chunk {
					tuples = append(tuples, "("+strings.TrimRight(strings.Repeat("?,", len(vals)), ",")+")")
					args = append(args, vals...)
				}

				condsSize := 1
				if condsSize < len(queryCond.FixedCols) {
					condsSize = len(queryCond.FixedCols)
				}
				fixedConds := make([]string, 0, condsSize)
				for _, col := range queryCond.FixedCols {
					fixedConds = append(fixedConds, fmt.Sprintf("%s = ?", col))
				}
				minLimit := min(dataLimit, batchSize)
				limit := fmt.Sprintf(" LIMIT %d OFFSET %d", minLimit, dataOffset)
				query := BuildQuery([]string{"SELECT ", strings.Join(fields, ", "), " FROM "}, table, join, fixedConds, cond, queryCond.InCols, tuples, order, limit)

				execTime := time.Now()
				rows, err := rivetxsql.Pool.QueryContext(ctx, query, args...)
				if err != nil {
					return false, ee.New(err, "chunkIndex:%v, dataOffset:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
						chunkIndex, dataOffset, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, totalCount, 0, 0, args)
				}

				columns, _ := rows.Columns()
				batchCount := 0
				for rows.Next() {
					rowData, err := scanRow[T](rows, columns)
					if err != nil {
						rows.Close()
						return false, ee.New(err, "")
					}
					result = append(result, rowData)
					batchCount++
				}
				rows.Close()

				totalCount += batchCount
				if LogRivetxsql().GetLevel() == log4.DEBUG {
					LogRivetxsql().Debug("chunkIndex:%v, dataOffset:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
						chunkIndex, dataOffset, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, totalCount, batchCount, 0, args)
				}

				// if this query returns fewer than batchSize, the chunk is complete
				if batchCount < minLimit {
					return true, nil
				}
				dataOffset += minLimit
				dataLimit -= minLimit
				return false, nil
			}()

			if err != nil {
				return nil, ee.New(err, "")
			}
			if isBreak {
				break
			}
		}
	}

	return result, nil
}

// Select supports struct conditions and paging
func Select[T any, F any, I any](rivetxsql *RivetxSql, table string, join string, queryStruct QueryStruct[F, I],
	cond string, condArgs []interface{}, order string, limit int, offset int, batchSize int,
	timeout time.Duration) ([]T, error) {
	// fixed columns and values
	fixedCols, fixedVals, err := StructFieldsAndValues(queryStruct.Fixed)
	if err != nil {
		return nil, ee.New(err, "StructFieldsAndValues")
	}

	inCols := []string{}
	inVals := make([][]interface{}, 0, len(queryStruct.InVals))
	// IN columns and values
	if len(queryStruct.InVals) > 0 {
		inCols, err = StructFields[I]()
		if err != nil {
			return nil, ee.New(err, "StructFields")
		}
		for _, v := range queryStruct.InVals {
			_, vals, err := StructFieldsAndValues(v)
			if err != nil {
				return nil, ee.New(err, "StructFieldsAndValues")
			}
			inVals = append(inVals, vals)
		}
	}

	return SelectRaw[T](rivetxsql, table, join, QueryCond{
		FixedCols: fixedCols,
		FixedVals: fixedVals,
		InCols:    inCols,
		InVals:    inVals,
	}, cond, condArgs, order, limit, offset, batchSize, timeout)
}

type SelectBuilder[T any] struct {
	table            string
	join             string
	queryCond        QueryCond
	cond             string
	condArgs         []interface{}
	order            string
	limit            int
	offset           int
	timeout          time.Duration
	batchSize        int
	orderField       string
	isDescOrderField bool
}

func (obj *SelectBuilder[T]) Join(join string) *SelectBuilder[T] {
	obj.join = join
	return obj
}

func (obj *SelectBuilder[T]) WhereEq(Col string, Val interface{}) *SelectBuilder[T] {
	obj.queryCond.FixedCols = append(obj.queryCond.FixedCols, Col)
	obj.queryCond.FixedVals = append(obj.queryCond.FixedVals, Val)
	return obj
}

func (obj *SelectBuilder[T]) WhereIn(Cols []string, Vals [][]interface{}) *SelectBuilder[T] {
	obj.queryCond.InCols = Cols
	obj.queryCond.InVals = Vals
	return obj
}

func (obj *SelectBuilder[T]) WhereInBatchSize(batchSize int) *SelectBuilder[T] {
	obj.queryCond.InBatchSize = batchSize
	return obj
}

func (obj *SelectBuilder[T]) BatchSize(batchSize int) *SelectBuilder[T] {
	obj.batchSize = batchSize
	return obj
}

func Where(objCond string, objArgs []interface{}, cond string, args []interface{}) (string, []interface{}) {
	objCond += " "
	objCond += cond
	objArgs = append(objArgs, args...)
	return objCond, objArgs
}

func (obj *SelectBuilder[T]) Where(cond string, args []interface{}) *SelectBuilder[T] {
	obj.cond, obj.condArgs = Where(obj.cond, obj.condArgs, cond, args)
	return obj
}

func (obj *SelectBuilder[T]) Order(order string) *SelectBuilder[T] {
	obj.order = order
	return obj
}

func (obj *SelectBuilder[T]) Limit(limit int) *SelectBuilder[T] {
	obj.limit = limit
	return obj
}

func (obj *SelectBuilder[T]) Offset(offset int) *SelectBuilder[T] {
	obj.offset = offset
	return obj
}

func (obj *SelectBuilder[T]) Timeout(timeout time.Duration) *SelectBuilder[T] {
	obj.timeout = timeout
	return obj
}

func (obj *SelectBuilder[T]) OrderFieldSelect(field string, isDesc bool, limit int) *SelectBuilder[T] {
	obj.orderField = field
	obj.isDescOrderField = isDesc
	obj.Limit(limit)
	return obj
}

type OrderFieldSelectValue interface {
	OrderFieldSelectValue() interface{}
}

func (obj *SelectBuilder[T]) execOrderFieldSelect(rivetxsql *RivetxSql, sort string, operator string) ([]T, error) {
	totalLimit := obj.limit
	if obj.batchSize <= 0 {
		obj.batchSize = BatchSize
	}
	result := make([]T, 0, totalLimit)

	if obj.offset > totalLimit {
		return nil, ee.New(nil, "obj.offset:%v > totalLimit:%v", obj.offset, totalLimit)
	}

	mode := ""
	if len(obj.queryCond.FixedCols) > 0 || len(obj.cond) > 0 {
		mode = " AND "
	}
	var lastID interface{}
	for len(result) < totalLimit {
		remaining := totalLimit - len(result)
		limit := min(remaining, obj.batchSize)

		var (
			values []T
			err    error
		)

		if lastID != nil {
			cond, condArgs := Where(obj.cond, obj.condArgs, fmt.Sprintf("%s %s %s ?", mode, obj.orderField, operator), []interface{}{lastID})
			cond, condArgs = Where(cond, condArgs, "AND 1=1", []interface{}{})
			order := fmt.Sprintf("ORDER BY %s %s", obj.orderField, sort)
			values, err = SelectRaw[T](rivetxsql, obj.table, obj.join, obj.queryCond, cond, condArgs, order, limit, 0, obj.batchSize, obj.timeout)
		} else {
			order := fmt.Sprintf("ORDER BY %s %s", obj.orderField, sort)
			cond, condArgs := Where(obj.cond, obj.condArgs, fmt.Sprintf(" %s 1=1", mode), []interface{}{})
			values, err = SelectRaw[T](rivetxsql, obj.table, obj.join, obj.queryCond, cond, condArgs, order, limit, obj.offset, obj.batchSize, obj.timeout)
		}

		if err != nil {
			return nil, err
		}

		if len(values) == 0 {
			break
		}

		if obj, ok := any(values[len(values)-1]).(OrderFieldSelectValue); ok {
			lastID = obj.OrderFieldSelectValue()
		} else {
			return nil, ee.New(nil, "not supper OrderFieldSelectValue")
		}

		result = append(result, values...)
	}

	return result, nil
}

func (obj *SelectBuilder[T]) Exec(rivetxsql *RivetxSql) ([]T, error) {
	if len(obj.orderField) <= 0 {
		return SelectRaw[T](rivetxsql, obj.table, obj.join, obj.queryCond, obj.cond, obj.condArgs, obj.order, obj.limit, obj.offset, obj.batchSize, obj.timeout)
	} else {
		if obj.isDescOrderField {
			return obj.execOrderFieldSelect(rivetxsql, "DESC", "<")
		} else {
			return obj.execOrderFieldSelect(rivetxsql, "ASC", ">")
		}
	}
}

func NewSelect[T any](table string) *SelectBuilder[T] {
	return &SelectBuilder[T]{
		table:     table,
		join:      "",
		queryCond: QueryCond{},
		timeout:   Timeout,
	}
}
