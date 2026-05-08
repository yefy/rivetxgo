package utilx

import (
	"strings"
	"unsafe"
)

func StringTrim(str string) string {
	return strings.Trim(str, " \r\n\t")
}

func SliceByteToString(b []byte) string {
	if b == nil {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}
func StringToSliceByte(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
