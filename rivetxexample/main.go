package main

import (
	"fmt"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"os"
	"rivetxgo/rivetxexample/examples"
	"rivetxgo/rivetxsql"
)

func main() {
	err := doMain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err:%v\n", err)
		log4.Error("err:%v", err)
	}
}

func doMain() error {
	err := log4.InitFile("./conf/log4.yaml")
	if err != nil {
		return ee.New(err, "err:log4.InitFile")
	}
	defer func() {
		log4.Close(true)
	}()
	log4.Info("doMain start")
	err = rivetxsql.RivetxSqlTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = examples.LruTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = examples.SpawnTests()
	if err != nil {
		return ee.New(err, "")
	}
	log4.Info("doMain end")
	return nil
}
