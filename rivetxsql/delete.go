package rivetxsql

import (
	"context"
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"strings"
	"time"
)

// DeleteRaw 支持每组独立删除，每组内 IN 条目分批，固定列可选
func DeleteRaw(rivetxsql *RivetxSql, table string, g QueryCond, cond string, condArgs []interface{}, limit int, timeout time.Duration) (*DeleteResult, error) {
	if timeout == 0 {
		timeout = Timeout
	}

	startTime := time.Now()
	TotalAffected := int64(0)
	LastInsertId := int64(0)

	if len(g.FixedCols) != len(g.FixedVals) {
		return nil, ee.New(nil, "fixedCols and fixedVals length mismatch")
	}

	// 验证IN值的列数一致性
	for i, vals := range g.InVals {
		if len(vals) != len(g.InCols) {
			return nil, ee.New(nil, "InVals[%d] length %d does not match InCols length %d", i, len(vals), len(g.InCols))
		}
	}

	if len(g.FixedCols) <= 0 && len(g.InCols) <= 0 && len(cond) <= 0 {
		return nil, ee.New(nil, "both FixedCols and InCols and cond are empty")
	}

	if g.InBatchSize <= 0 {
		g.InBatchSize = BatchSize
	}

	if len(g.InVals) > g.InBatchSize && limit > 0 {
		return nil, ee.New(nil, "len(g.InVals) > g.InBatchSize && limit > 0")
	}

	chunksSize := 1
	if len(g.InVals) > 0 {
		chunksSize = (len(g.InVals) + g.InBatchSize - 1) / g.InBatchSize
	}
	chunks := make([][][]interface{}, 0, chunksSize)
	if len(g.InVals) > 0 {
		for start := 0; start < len(g.InVals); start += g.InBatchSize {
			end := start + g.InBatchSize
			if end > len(g.InVals) {
				end = len(g.InVals)
			}
			chunks = append(chunks, g.InVals[start:end])
		}
	} else {
		chunks = append(chunks, [][]interface{}{})
	}

	for index, chunk := range chunks {
		err := func() error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			// 构造 IN ((?, ?), ...)
			chunkSize := len(chunk) + len(condArgs)
			if chunkSize <= 0 {
				chunkSize = 1
			}
			tuples := make([]string, 0, chunkSize)
			args := make([]interface{}, 0, len(g.FixedVals)+chunkSize*len(g.InCols))

			if len(g.FixedCols) > 0 {
				args = append(args, g.FixedVals...)
			}

			if len(condArgs) > 0 {
				args = append(args, condArgs...)
			}

			for _, vals := range chunk {
				tuples = append(tuples, "("+strings.TrimRight(strings.Repeat("?,", len(vals)), ",")+")")
				args = append(args, vals...)
			}

			conds := make([]string, 0, len(g.FixedCols))
			if len(g.FixedCols) > 0 {
				for _, col := range g.FixedCols {
					conds = append(conds, fmt.Sprintf("%s = ?", col))
				}
			}
			limitStr := ""
			if limit > 0 {
				limitStr = fmt.Sprintf(" LIMIT %d ", limit)
			}
			query := BuildQuery([]string{"DELETE FROM "}, table, "", conds, cond, g.InCols, tuples, "", limitStr)

			execTime := time.Now()
			ret, err := rivetxsql.Pool.ExecContext(ctx, query, args...)
			if err != nil {
				return ee.New(err, "index:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
					index, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, TotalAffected, 0, LastInsertId, args)
			}
			RowsAffected, _ := ret.RowsAffected()
			lastInsertId, _ := ret.LastInsertId()
			TotalAffected += RowsAffected
			if RowsAffected > 0 {
				LastInsertId = lastInsertId + RowsAffected - 1
			}
			if LogRivetxsql().GetLevel() == log4.DEBUG {
				LogRivetxsql().Debug("index:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v, args:%+v",
					index, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, TotalAffected, RowsAffected, LastInsertId, args)
			}
			return nil
		}()
		if err != nil {
			return nil, ee.New(err, "")
		}
	}
	return &DeleteResult{TotalAffected, LastInsertId}, nil
}

// Delete 支持结构体解析，封装调用 Delete
func Delete[F any, I any](rivetxsql *RivetxSql, table string, g QueryStruct[F, I], cond string, condArgs []interface{}, timeout time.Duration) (*DeleteResult, error) {
	// 固定列和值
	fixedCols, fixedVals, err := StructFieldsAndValues(g.Fixed)
	if err != nil {
		return nil, ee.New(err, "StructFieldsAndValues")
	}

	inCols := []string{}
	inVals := make([][]interface{}, 0, len(g.InVals))
	// IN 列和值
	if len(g.InVals) > 0 {
		inCols, err = StructFields[I]()
		if err != nil {
			return nil, ee.New(err, "StructFields")
		}
		for _, v := range g.InVals {
			_, vals, err := StructFieldsAndValues(v)
			if err != nil {
				return nil, ee.New(err, "StructFieldsAndValues")
			}
			inVals = append(inVals, vals)
		}
	}

	groupDelete := QueryCond{
		FixedCols: fixedCols,
		FixedVals: fixedVals,
		InCols:    inCols,
		InVals:    inVals,
	}

	// 调用原始 Delete 执行
	return DeleteRaw(rivetxsql, table, groupDelete, cond, condArgs, 0, timeout)
}

type DeleteResult struct {
	TotalAffected int64
	LastInsertID  int64 // 最后一个 batch 的
}

type DeleteBuilder struct {
	table        string
	queryCond    QueryCond
	cond         string
	condArgs     []interface{}
	limit        int
	timeout      time.Duration
	reserveField string
	reserveSize  int
	reserveSleep time.Duration
}

func (obj *DeleteBuilder) WhereEq(Col string, Val interface{}) *DeleteBuilder {
	obj.queryCond.FixedCols = append(obj.queryCond.FixedCols, Col)
	obj.queryCond.FixedVals = append(obj.queryCond.FixedVals, Val)
	return obj
}

func (obj *DeleteBuilder) WhereIn(Cols []string, Vals [][]interface{}) *DeleteBuilder {
	obj.queryCond.InCols = Cols
	obj.queryCond.InVals = Vals
	return obj
}

func (obj *DeleteBuilder) WhereInBatchSize(batchSize int) *DeleteBuilder {
	obj.queryCond.InBatchSize = batchSize
	return obj
}

//func (obj *DeleteBuilder) BatchSize(batchSize int) *DeleteBuilder {
//	obj.batchSize = batchSize
//	return obj
//}

func (obj *DeleteBuilder) Where(cond string, args []interface{}) *DeleteBuilder {
	obj.cond, obj.condArgs = Where(obj.cond, obj.condArgs, cond, args)
	return obj
}

func (obj *DeleteBuilder) Limit(limit int) *DeleteBuilder {
	obj.limit = limit
	return obj
}

func (obj *DeleteBuilder) Timeout(timeout time.Duration) *DeleteBuilder {
	obj.timeout = timeout
	return obj
}

func (obj *DeleteBuilder) ReserveSize(field string, reserveSize int, reserveSleep time.Duration) *DeleteBuilder {
	obj.reserveField = field
	obj.reserveSize = reserveSize
	obj.reserveSleep = reserveSleep
	obj.Limit(BatchSize)
	return obj
}

func (obj *DeleteBuilder) execReserveSize(rivetxsql *RivetxSql) (*DeleteResult, error) {
	var key interface{}
	rows, err := rivetxsql.Pool.Query(fmt.Sprintf("SELECT %v FROM %v ORDER BY %v DESC LIMIT 1 OFFSET %v", obj.reserveField, obj.table, obj.reserveField, obj.reserveSize))
	if err != nil {
		return nil, ee.New(err, "")
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&key); err != nil {
			return nil, ee.New(err, "")
		}
		break
	}

	if key == nil {
		return &DeleteResult{}, nil
	}

	DeleteResult := &DeleteResult{}
	for {
		limit := obj.limit
		if limit <= 0 {
			limit = BatchSize
		}
		res, err := NewDelete(obj.table).Where(fmt.Sprintf("%v <= ?", obj.reserveField), []interface{}{key}).Limit(limit).Exec(rivetxsql)
		if err != nil {
			return nil, ee.New(err, "")
		}
		if res.TotalAffected <= 0 {
			break
		}
		DeleteResult.TotalAffected += res.TotalAffected
		DeleteResult.LastInsertID = res.LastInsertID
		if obj.reserveSleep != 0 {
			time.Sleep(obj.reserveSleep)
		}
	}

	return DeleteResult, nil
}

func (obj *DeleteBuilder) Exec(rivetxsql *RivetxSql) (*DeleteResult, error) {
	if len(obj.reserveField) <= 0 {
		return DeleteRaw(rivetxsql, obj.table, obj.queryCond, obj.cond, obj.condArgs, obj.limit, obj.timeout)
	} else {
		return obj.execReserveSize(rivetxsql)
	}
}

func NewDelete(table string) *DeleteBuilder {
	return &DeleteBuilder{
		table:   table,
		limit:   0,
		timeout: Timeout,
	}
}
