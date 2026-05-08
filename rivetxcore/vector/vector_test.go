package vector

import "testing"

func TestSliceGet(t *testing.T) {
	{
		nums := []uint32{1, 2, 3}
		num, ok := SliceGet(nums, 2)
		if !ok {
			t.Errorf("nums:%v, index:%v, num:%v", nums, 2, num)
		}
		num, ok = SliceGet(nums, 3)
		if ok {
			t.Errorf("nums:%v, index:%v, num:%v", nums, 3, num)
		}
	}
	{
		strs := []string{"1", "2", "3"}
		str, ok := SliceGet(strs, 2)
		if !ok {
			t.Errorf("strs:%v, index:%v, str:%v", strs, 2, str)
		}
		str, ok = SliceGet(strs, 3)
		if ok {
			t.Errorf("strs:%v, index:%v, str:%v", strs, 3, str)
		}
	}
}
