package rivetxsql

import (
	"context"
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"rivetxgo/rivetxcore"
	"time"
)

func Create[T any](rivetxsql *RivetxSql, tableName string, timeout time.Duration) error {
	meta, err := StructMeta[T]()
	if err != nil {
		return ee.New(err, "StructFields")
	}
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %v (`, tableName)
	for i, col := range meta.cols {
		auto := ""
		if meta.autoColMap[col] {
			auto = "AUTO_INCREMENT"
		}
		query += fmt.Sprintf(" %s %s NOT NULL %s, ", col, meta.sqlTypes[i], auto)
	}
	if len(meta.primary) > 0 {
		query += fmt.Sprintf(" PRIMARY KEY (%s), ", meta.primary)
	}

	for key, values := range meta.uniqueMap {
		query += fmt.Sprintf(" UNIQUE INDEX %s ( ", key)
		for i, value := range values {
			query += fmt.Sprintf(" %s ", value)
			if i != len(values)-1 {
				query += ","
			}
		}
		query += "),"
	}

	for key, value := range meta.indexMap {
		query += fmt.Sprintf(" INDEX %s ( %s ),", key, value)
	}
	query = rivetxcore.StringTrim(query)
	if query[len(query)-1] == ',' {
		query = query[0 : len(query)-1]
	}
	query += ");"

	if timeout == 0 {
		timeout = Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	startTime := time.Now()
	execTime := time.Now()
	_, err = rivetxsql.Pool.ExecContext(ctx, query)
	if err != nil {
		return ee.New(err, "tableName:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v",
			tableName, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, 1, 1, 0)
	}
	if LogRivetxsql().GetLevel() == log4.DEBUG {
		LogRivetxsql().Debug("tableName:%v, allTime:%v, execTime:%v, query:%v, TotalAffected:%v, RowsAffected:%v, LastInsertId:%v",
			tableName, time.Since(startTime).Milliseconds(), time.Since(execTime).Milliseconds(), query, 1, 1, 0)
	}
	return nil
}
