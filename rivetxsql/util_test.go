package rivetxsql

import (
	"reflect"
	"testing"
	"time"
)

type utilTestStruct struct {
	Id     uint64 `db:"id" attr:"auto,primary"`
	Name   string `db:"name" size:"64"`
	Age    int
	hidden string
}

func TestGoTypeToSql(t *testing.T) {
	cases := []struct {
		typ     reflect.Type
		tag     string
		want    string
		wantErr bool
	}{
		{reflect.TypeOf(uint64(0)), "", "BIGINT UNSIGNED", false},
		{reflect.TypeOf(uint8(0)), "", "INT UNSIGNED", false},
		{reflect.TypeOf(int64(0)), "", "BIGINT", false},
		{reflect.TypeOf(int32(0)), "", "INT", false},
		{reflect.TypeOf(""), "64", "VARCHAR(64)", false},
		{reflect.TypeOf(""), "TEXT", "TEXT", false},
		{reflect.TypeOf(true), "", "TINYINT(1)", false},
		{reflect.TypeOf(time.Time{}), "", "DATETIME", false},
		{reflect.TypeOf(time.Time{}), "3", "DATETIME(3)", false},
		{reflect.TypeOf(struct{ X int }{}), "", "", true},
	}

	for _, c := range cases {
		got, err := GoTypeToSql(c.typ, c.tag)
		if c.wantErr {
			if err == nil {
				t.Fatalf("expected error for type %v tag %q", c.typ, c.tag)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for type %v tag %q: %v", c.typ, c.tag, err)
		}
		if got != c.want {
			t.Fatalf("GoTypeToSql(%v,%q) = %q, want %q", c.typ, c.tag, got, c.want)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"HelloWorld", "hello_world"},
		{"HTTPServer", "h_t_t_p_server"},
		{"snake_case", "snake_case"},
		{"", ""},
	}

	for _, c := range cases {
		got := ToSnakeCase(c.input)
		if got != c.want {
			t.Fatalf("ToSnakeCase(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestStructMetaAndFields(t *testing.T) {
	meta, err := StructMeta[utilTestStruct]()
	if err != nil {
		t.Fatalf("StructMeta failed: %v", err)
	}

	wantCols := []string{"id", "name", "age"}
	if !reflect.DeepEqual(meta.cols, wantCols) {
		t.Fatalf("meta.cols = %v, want %v", meta.cols, wantCols)
	}

	fields, err := StructFields[utilTestStruct]()
	if err != nil {
		t.Fatalf("StructFields failed: %v", err)
	}
	if !reflect.DeepEqual(fields, wantCols) {
		t.Fatalf("StructFields = %v, want %v", fields, wantCols)
	}
}

func TestStructValuesAndDiscardAuto(t *testing.T) {
	meta, err := StructMeta[utilTestStruct]()
	if err != nil {
		t.Fatalf("StructMeta failed: %v", err)
	}

	valueObj := utilTestStruct{Id: 1, Name: "alice", Age: 18}
	values, err := StructValues(meta, valueObj)
	if err != nil {
		t.Fatalf("StructValues failed: %v", err)
	}
	wantValues := []interface{}{uint64(1), "alice", 18}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("StructValues = %v, want %v", values, wantValues)
	}

	values2, err := StructValuesByDiscardAuto(meta, valueObj)
	if err != nil {
		t.Fatalf("StructValuesByDiscardAuto failed: %v", err)
	}
	wantValues2 := []interface{}{"alice", 18}
	if !reflect.DeepEqual(values2, wantValues2) {
		t.Fatalf("StructValuesByDiscardAuto = %v, want %v", values2, wantValues2)
	}
}

func TestStructFieldsAndValues(t *testing.T) {
	cols, vals, err := StructFieldsAndValues(utilTestStruct{Id: 1, Name: "bob", Age: 21})
	if err != nil {
		t.Fatalf("StructFieldsAndValues failed: %v", err)
	}
	wantCols := []string{"id", "name", "age"}
	wantVals := []interface{}{uint64(1), "bob", 21}
	if !reflect.DeepEqual(cols, wantCols) {
		t.Fatalf("StructFieldsAndValues cols = %v, want %v", cols, wantCols)
	}
	if !reflect.DeepEqual(vals, wantVals) {
		t.Fatalf("StructFieldsAndValues vals = %v, want %v", vals, wantVals)
	}

	cols, vals, err = StructFieldsAndValues(&utilTestStruct{Id: 2, Name: "cindy", Age: 22})
	if err != nil {
		t.Fatalf("StructFieldsAndValues pointer failed: %v", err)
	}
	wantVals = []interface{}{uint64(2), "cindy", 22}
	if !reflect.DeepEqual(vals, wantVals) {
		t.Fatalf("StructFieldsAndValues pointer vals = %v, want %v", vals, wantVals)
	}

	cols, vals, err = StructFieldsAndValues((*utilTestStruct)(nil))
	if err != nil {
		t.Fatalf("StructFieldsAndValues nil pointer failed: %v", err)
	}
	if cols != nil || vals != nil {
		t.Fatalf("expected nil cols and vals for nil pointer, got %v and %v", cols, vals)
	}
}

func TestEstimateLengths(t *testing.T) {
	join := EstimateJoin{Parts: []string{"a", "bc"}, Sep: ", "}
	if got := estimateJoinLen(join); got != 5 {
		t.Fatalf("estimateJoinLen = %d, want 5", got)
	}

	if got := estimateStrLen([]string{"aa", "bbb"}, []string{"c"}, []EstimateJoin{join}); got != 159 {
		t.Fatalf("estimateStrLen = %d, want 159", got)
	}
}

func TestBuildQuery(t *testing.T) {
	sql := BuildQuery([]string{"SELECT ", "col FROM "}, "table", "JOIN t2", []string{"a=1", "b=2"}, "c=3", []string{"id"}, []string{"(?,?)"}, "ORDER BY id", "LIMIT 10")
	want := "SELECT  col FROM  table JOIN t2 WHERE a=1 AND b=2 AND c=3 AND (id) IN ((?,?)) ORDER BY id LIMIT 10"
	if sql != want {
		t.Fatalf("BuildQuery = %q, want %q", sql, want)
	}
}
