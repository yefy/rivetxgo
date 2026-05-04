package rivetxsql

import (
	"context"
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"strings"
	"time"
)

// -----------------------------
// InsertRaw 支持手动列和值
// -----------------------------
func InsertRaw(rivetxsql *RivetxSql, table string, cols []string, vals [][]interface{}, maxPerBatch int,
	onDuplicateUpdate string, ignoreDuplicate bool, timeout time.Duration) (*InsertResult, error) {
	if len(vals) == 0 || len(cols) == 0 {
		return nil, ee.New(nil, "len(vals) == 0 || len(cols) == 0")
	}

	// 验证IN值的列数一致性
	for i, vals := range vals {
		if len(vals) != len(cols) {
			return nil, ee.New(nil, "InVals[%d] length %d does not match InCols length %d", i, len(vals), len(cols))
		}
	}

	if timeout == 0 {
		timeout = Timeout
	}

	if maxPerBatch <= 0 {
		maxPerBatch = BatchSize
	}

	insertKeyword := "INSERT"
	if ignoreDuplicate {
		insertKeyword = "INSERT IGNORE"
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

			// 构造 (?, ?, ...)
			tuples := make([]string, 0, len(chunk))
			args := make([]interface{}, 0, len(chunk)*len(cols))
			for _, v := range chunk {
				tuples = append(tuples, "("+strings.TrimRight(strings.Repeat("?,", len(v)), ",")+")")
				args = append(args, v...)
			}

			query := fmt.Sprintf("%s INTO %s (%s) VALUES %s",
				insertKeyword,
				table,
				strings.Join(cols, ", "),
				strings.Join(tuples, ","))

			if onDuplicateUpdate != "" {
				query += " ON DUPLICATE KEY UPDATE " + onDuplicateUpdate
			}

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
	return &InsertResult{TotalAffected, LastInsertId}, nil
}

// -----------------------------
// Insert 支持结构体
// -----------------------------
func Insert[T any](rivetxsql *RivetxSql, table string, data []T, maxPerBatch int, onDuplicateUpdate string, ignoreDuplicate bool, timeout time.Duration) (*InsertResult, error) {
	if len(data) == 0 {
		return nil, ee.New(nil, "len(data) == 0")
	}

	meta, err := StructMeta[T]()
	if err != nil {
		return nil, ee.New(err, "StructFields")
	}
	cols := meta.discardAutoCols

	vals := make([][]interface{}, 0, len(data))
	for _, d := range data {
		v, err := StructValuesByDiscardAuto(meta, d)
		if err != nil {
			return nil, ee.New(err, "StructFieldsAndValues")
		}
		vals = append(vals, v)
	}

	return InsertRaw(rivetxsql, table, cols, vals, maxPerBatch, onDuplicateUpdate, ignoreDuplicate, timeout)
}

type InsertResult struct {
	TotalAffected int64
	LastInsertID  int64 // 最后一个 batch 的
}

type InsertBuilder[T any] struct {
	table             string
	data              []T
	maxPerBatch       int
	onDuplicateUpdate string
	ignoreDuplicate   bool
	timeout           time.Duration
}

func (obj *InsertBuilder[T]) BatchSize(maxPerBatch int) *InsertBuilder[T] {
	obj.maxPerBatch = maxPerBatch
	return obj
}

func (obj *InsertBuilder[T]) OnDuplicateUpdate(onDuplicateUpdate string) *InsertBuilder[T] {
	obj.onDuplicateUpdate = onDuplicateUpdate
	return obj
}

func (obj *InsertBuilder[T]) IgnoreDuplicate() *InsertBuilder[T] {
	obj.ignoreDuplicate = true
	return obj
}
func (obj *InsertBuilder[T]) Timeout(timeout time.Duration) *InsertBuilder[T] {
	obj.timeout = timeout
	return obj
}

func (obj *InsertBuilder[T]) Exec(rivetxsql *RivetxSql) (*InsertResult, error) {
	return Insert(rivetxsql, obj.table, obj.data, obj.maxPerBatch, obj.onDuplicateUpdate, obj.ignoreDuplicate, obj.timeout)
}

func NewInsert[T any](table string, data []T) *InsertBuilder[T] {
	return &InsertBuilder[T]{
		table:             table,
		data:              data,
		maxPerBatch:       BatchSize,
		onDuplicateUpdate: "",
		ignoreDuplicate:   false,
		timeout:           Timeout,
	}
}
