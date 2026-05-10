package main

import (
	"fmt"
	"math/rand"
	"os"
	"rivetxgo/rivetxcore/gox"
	"rivetxgo/rivetxcore/limitx"
	"rivetxgo/rivetxcore/recoverx"
	"rivetxgo/rivetxcore/tcpx"
	"rivetxgo/rivetxexample/examples"
	"rivetxgo/rivetxsql"
	"time"

	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

// go test ./...
func main() {
	defer func() {
		log4.Close(true)
	}()

	err := doMain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err:%v\n", err)
		log4.Error("err:%v", err)
	}
}

func doMain() error {
	defer func() {
		if r := recover(); r != nil {
			log4.Recover(r)
		}
	}()

	err := log4.InitFile("./conf/log4.yaml")
	if err != nil {
		return ee.New(err, "err:log.InitFile")
	}
	examples.LogReopenTests()

	recoverx.RedirectStderr("./logs/recover.log")

	rand.Seed(time.Now().UnixNano())

	err = limitx.SetUlimit(nil)
	if err != nil {
		return ee.New(err, "")
	}

	dir, err := os.Getwd()
	if err != nil {
		return ee.New(err, "pwd")
	}
	log4.Info("Current working directory:%v", dir)

	log4.Info("doMain start")
	err = rivetxsql.RivetxSqlTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = examples.LruTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = gox.GoTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = tcpx.TcpTests()
	if err != nil {
		return ee.New(err, "")
	}
	log4.Info("doMain end")
	return nil
}
