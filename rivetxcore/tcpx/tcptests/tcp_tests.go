package tcptests

import (
	"github.com/yefy/log4go/ee"
	"rivetxgo/rivetxcore/tcpx/tcptests1"
	"rivetxgo/rivetxcore/tcpx/tcptests2"
	"rivetxgo/rivetxcore/tcpx/tcptests3"
)

func TcpTests() error {
	err := tcptests1.TcpTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = tcptests2.TcpTests()
	if err != nil {
		return ee.New(err, "")
	}
	err = tcptests3.TcpTests()
	if err != nil {
		return ee.New(err, "")
	}
	return nil
}
