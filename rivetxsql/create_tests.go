package rivetxsql

import (
	"github.com/yefy/log4go/ee"
)

func TestCreate() error {
	err := TestCreateTable()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}

func TestCreateTable() error {
	rivetxsql, err := testOpenRivetxSql()
	if err != nil {
		return ee.New(err, "")
	}
	defer rivetxsql.Pool.Close()

	if err := testDataCreateTable(rivetxsql); err != nil {
		return ee.New(err, "")
	}
	testDataDropTable(rivetxsql)
	err = Create[TestData](rivetxsql, "test_data", 0)
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}
