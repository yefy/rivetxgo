package examples

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/yefy/log4go/ee"
)

func LruTests() error {
	l, err := lru.New[int, any](3)
	if err != nil {
		return ee.New(err, "")
	}
	printFunc := func() {
		keys := l.Keys()
		fmt.Printf("len:%v\n", l.Len())
		for _, key := range keys {
			value, ok := l.Get(key)
			if !ok {
				continue
			}
			fmt.Printf("key:%v, value:%v\n", key, value)
		}
	}
	l.Add(1, 1)
	l.Add(2, 2)
	l.Add(3, 3)
	l.Add(4, 4)
	printFunc()
	l.Add(3, 33)
	printFunc()

	{
		var p *map[string]bool
		func(p **map[string]bool) {

		}(&p)
		var m map[string]bool
		if p != nil {
			m = *p
		}
		var m2 map[string]bool
		fmt.Printf("m:%v, m2:%v, %v:%v \n", m, m2, m == nil, m2 == nil)
	}
	{
		var p *map[string]bool
		func(pp **map[string]bool) {
			p := make(map[string]bool)
			*pp = &p
			p["aaa"] = true
		}(&p)
		fmt.Printf("*p:%v \n", *p)
	}

	return nil
}
