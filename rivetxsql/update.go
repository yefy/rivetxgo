package rivetxsql

import (
	"context"
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"strings"
	"time"
)

// UpdateRaw generic batch update
func UpdateRaw(rivetxsql *RivetxSql, table string, cols []string, vals [][]interface{}, joinOn []string, setExpr []string, maxPerBatch int, timeout time.Duration) (*UpdateResult, error) {
	if len(vals) == 0 || len(cols) == 0 || len(joinOn) == 0 || len(setExpr) == 0 {
		return nil, ee.New(nil, "len(vals) == 0 || len(cols) == 0 || len(joinOn) == 0 || len(setExpr) == 0")
	}

	// verify IN values have consistent column count
	for i, vals := range vals {
		if len(vals) != len(cols) {
			return nil, ee.New(nil, "InVals[%d] length %d does not match InCols length %d", i, len(vals), len(cols))
		}
	}

	if maxPerBatch <= 0 {
		maxPerBatch = BatchSize
	}

	if timeout == 0 {
		timeout = Timeout
	}

	startTime := time.Now()
	TotalAffected := int64(0)
	LastInsertId := int64(0)
	for start := 0; start < len(vals); start += maxPerBatch {
		err := func() error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			end := start + maxPerBatch
			if end > len(vals) {
				end = len(vals)
			}
			chunk := vals[start:end]

			// build VALUES ROW(...)
			rows := make([]string, 0, len(chunk))
			args := make([]interface{}, 0, len(chunk)*len(cols))
			for _, v := range chunk {
				rows = append(rows, "ROW("+strings.TrimRight(strings.Repeat("?,", len(v)), ",")+")")
				args = append(args, v...)
			}

			// ON conditions
			onConditions := make([]string, 0, len(joinOn))
			for _, c := range joinOn {
				onConditions = append(onConditions, fmt.Sprintf("u.%s = v.%s", c, c))
			}

			// SET expressions are provided externally and not modified
			query := fmt.Sprintf(` 
UPDATE %s AS u 
JOIN (VALUES %s) AS v(%s) 
ON %s 
SET %s `,
				table,
				strings.Join(rows, ", "),
				strings.Join(cols, ", "),
				strings.Join(onConditions, " AND "),
				strings.Join(setExpr, ", "),
			)

			query = strings.ReplaceAll(query, "\n", "")

			execTime := time.Now()
			ret, err := rivetxsql.Pool.ExecContext(ctx, query, args...)
			if err != nil {
				return ee.New(err, "start:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
					start, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, TotalAffected, 0, LastInsertId, args)
			}
			RowsAffected, _ := ret.RowsAffected()
			lastInsertId, _ := ret.LastInsertId()
			TotalAffected += RowsAffected
			if RowsAffected > 0 {
				LastInsertId = lastInsertId + RowsAffected - 1
			}
			if LogRivetxsql().GetLevel() == log4.DEBUG {
				LogRivetxsql().Debug("start:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
					start, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, TotalAffected, RowsAffected, LastInsertId, args)
			}
			return nil
		}()
		if err != nil {
			return nil, ee.New(err, "")
		}
	}

	return &UpdateResult{TotalAffected, LastInsertId}, nil
}

// Update supports batch updates for struct slices
func Update[T any](rivetxsql *RivetxSql, table string, vals []T, joinOn []string, setExpr []string, maxPerBatch int, timeout time.Duration) (*UpdateResult, error) {
	if len(vals) == 0 || len(joinOn) == 0 || len(setExpr) == 0 {
		return nil, ee.New(nil, "len(vals) == 0 || len(joinOn) == 0 || len(setExpr) == 0")
	}
	if maxPerBatch <= 0 {
		maxPerBatch = BatchSize
	}

	// extract column names and values
	cols, err := StructFields[T]()
	if err != nil {
		return nil, ee.New(err, "StructFields")
	}
	vals2d := make([][]interface{}, 0, len(vals))
	for _, d := range vals {
		_, v, err := StructFieldsAndValues(d)
		if err != nil {
			return nil, ee.New(err, "StructFieldsAndValues")
		}
		vals2d = append(vals2d, v)
	}

	// call generic UpdateRaw
	return UpdateRaw(rivetxsql, table, cols, vals2d, joinOn, setExpr, maxPerBatch, timeout)
}

type UpdateResult struct {
	TotalAffected int64
	LastInsertID  int64 // last batch's
}

type UpdateBuilder[T any] struct {
	table       string
	data        []T
	maxPerBatch int
	joinOn      []string
	setExpr     []string
	timeout     time.Duration
}

func (obj *UpdateBuilder[T]) BatchSize(maxPerBatch int) *UpdateBuilder[T] {
	obj.maxPerBatch = maxPerBatch
	return obj
}

func (obj *UpdateBuilder[T]) JoinOn(joinOn []string) *UpdateBuilder[T] {
	obj.joinOn = joinOn
	return obj
}

func (obj *UpdateBuilder[T]) SetExpr(setExpr []string) *UpdateBuilder[T] {
	obj.setExpr = setExpr
	return obj
}
func (obj *UpdateBuilder[T]) Timeout(timeout time.Duration) *UpdateBuilder[T] {
	obj.timeout = timeout
	return obj
}

func (obj *UpdateBuilder[T]) Exec(rivetxsql *RivetxSql) (*UpdateResult, error) {
	return Update(rivetxsql, obj.table, obj.data, obj.joinOn, obj.setExpr, obj.maxPerBatch, obj.timeout)
}

func NewUpdate[T any](table string, data []T) *UpdateBuilder[T] {
	return &UpdateBuilder[T]{
		table:       table,
		data:        data,
		maxPerBatch: BatchSize,
		timeout:     Timeout,
	}
}
