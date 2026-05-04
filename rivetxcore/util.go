package rivetxcore

import "strings"

func StringTrim(str string) string {
	return strings.Trim(str, " \r\n\t")
}
