package rivetxsql

import (
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

func RivetxSqlTests() error {
	log4.Info("rivetxsql start")
	ToSnakeCase := ToSnakeCase("ToSnakeCase")
	log4.Info("ToSnakeCase:%+v", ToSnakeCase)

	{
		fixedCols, fixedVals, _ := StructFieldsAndValues(struct {
		}{})
		log4.Info("%v%v%v%v%v|%+v,%+v", "struct {}{} ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(struct {
		}{})
		log4.Info("%v%v%v%v%v|%+v,%+v", "struct {}{} ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(TestData{0, 0, "abc", 1, 1000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(TestData{0, 1, "abc", 2, 2000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(&TestData{0, 0, "abc", 1, 1000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "&TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(&TestData{0, 1, "abc", 2, 2000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "&TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(TestData{0, 0, "abc", 1, 1000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)

		fixedCols, fixedVals, _ = StructFieldsAndValues(TestData{0, 1, "abc", 2, 2000})
		log4.Info("%v%v%v%v%v|%+v,%+v", "TestData ", "fixedCols:", len(fixedCols), "fixedVals:", len(fixedVals), fixedCols, fixedVals)
	}

	err := TestCreate()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestSelete()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestInsert()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestDetele()
	if err != nil {
		return ee.New(err, "")
	}

	err = TestUpdate()
	if err != nil {
		return ee.New(err, "")
	}

	log4.Info("rivetxsql end")
	return nil
}
