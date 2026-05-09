package log

import "github.com/yefy/log4go/log4"

func LogHttp() *log4.Log4Target {
	return log4.Target("http")
}

func LogTcp() *log4.Log4Target {
	return log4.Target("tcp")
}
