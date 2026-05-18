package config

var openStackInfoToErrorLog bool

func OpenStackInfoToErrorLog(b bool)  {
	openStackInfoToErrorLog = b
}

func IsOpenStackInfoToErrorLog() bool {
	return openStackInfoToErrorLog
}
