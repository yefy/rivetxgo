package vector

import (
	"reflect"
	"testing"
)

func TestSliceGet(t *testing.T) {
	{
		nums := []uint32{1, 2, 3}
		num, ok := SliceGet(nums, 2)
		if !ok || num != 3 {
			t.Errorf("nums:%v, index:%v, num:%v, ok:%v", nums, 2, num, ok)
		}
		num, ok = SliceGet(nums, 3)
		if ok {
			t.Errorf("nums:%v, index:%v, num:%v, ok:%v", nums, 3, num, ok)
		}
		num, ok = SliceGet(nums, -1)
		if ok {
			t.Errorf("nums:%v, index:%v, num:%v, ok:%v", nums, -1, num, ok)
		}
	}
	{
		strs := []string{"1", "2", "3"}
		str, ok := SliceGet(strs, 2)
		if !ok || str != "3" {
			t.Errorf("strs:%v, index:%v, str:%v, ok:%v", strs, 2, str, ok)
		}
		str, ok = SliceGet(strs, 3)
		if ok {
			t.Errorf("strs:%v, index:%v, str:%v, ok:%v", strs, 3, str, ok)
		}
		str, ok = SliceGet(strs, -1)
		if ok {
			t.Errorf("strs:%v, index:%v, str:%v, ok:%v", strs, -1, str, ok)
		}
	}
	{
		// Empty slice
		empty := []int{}
		val, ok := SliceGet(empty, 0)
		if ok {
			t.Errorf("empty slice, index:0, val:%v, ok:%v", val, ok)
		}
	}
}

func TestSliceSet(t *testing.T) {
	nums := []int{1, 2, 3}
	ok := SliceSet(nums, 1, 5)
	if !ok || nums[1] != 5 {
		t.Errorf("nums:%v, index:1, value:5, ok:%v", nums, ok)
	}
	ok = SliceSet(nums, 3, 4)
	if ok {
		t.Errorf("nums:%v, index:3, value:4, ok:%v", nums, ok)
	}
	ok = SliceSet(nums, -1, 4)
	if ok {
		t.Errorf("nums:%v, index:-1, value:4, ok:%v", nums, ok)
	}
}

func TestContains(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	if !Contains(v, 2) {
		t.Errorf("vector:%v, contains 2: false", v.Datas())
	}
	if Contains(v, 4) {
		t.Errorf("vector:%v, contains 4: true", v.Datas())
	}
}

func TestEqual(t *testing.T) {
	v1 := New[int](3)
	v1.Set(0, 1)
	v1.Set(1, 2)
	v1.Set(2, 3)
	v2 := New[int](3)
	v2.Set(0, 1)
	v2.Set(1, 2)
	v2.Set(2, 3)
	if !Equal(v1, v2) {
		t.Errorf("v1:%v, v2:%v, equal: false", v1.Datas(), v2.Datas())
	}
	v2.Set(2, 4)
	if Equal(v1, v2) {
		t.Errorf("v1:%v, v2:%v, equal: true", v1.Datas(), v2.Datas())
	}
}

func TestNew(t *testing.T) {
	v := New[int]()
	if v.Len() != 0 || v.Cap() != 0 {
		t.Errorf("New() len:%d, cap:%d", v.Len(), v.Cap())
	}
	v = New[int](5)
	if v.Len() != 5 || v.Cap() != 5 {
		t.Errorf("New(5) len:%d, cap:%d", v.Len(), v.Cap())
	}
	v = New[int](5, 10)
	if v.Len() != 5 || v.Cap() != 10 {
		t.Errorf("New(5,10) len:%d, cap:%d", v.Len(), v.Cap())
	}
	v = New[int](10, 5) // capacity < length, should set capacity to length
	if v.Len() != 10 || v.Cap() != 10 {
		t.Errorf("New(10,5) len:%d, cap:%d", v.Len(), v.Cap())
	}
}

func TestFromSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}
	v := FromSlice(slice)
	if !reflect.DeepEqual(v.Datas(), slice) {
		t.Errorf("FromSlice(%v) datas:%v", slice, v.Datas())
	}
}

func TestVectorGet(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	val, ok := v.Get(1)
	if !ok || val != 2 {
		t.Errorf("Get(1) val:%v, ok:%v", val, ok)
	}
	val, ok = v.Get(3)
	if ok {
		t.Errorf("Get(3) val:%v, ok:%v", val, ok)
	}
	val, ok = v.Get(-1)
	if ok {
		t.Errorf("Get(-1) val:%v, ok:%v", val, ok)
	}
}

func TestVectorSet(t *testing.T) {
	v := New[int](3)
	ok := v.Set(1, 5)
	if !ok || v.Datas()[1] != 5 {
		t.Errorf("Set(1,5) ok:%v, datas:%v", ok, v.Datas())
	}
	ok = v.Set(3, 4)
	if ok {
		t.Errorf("Set(3,4) ok:%v", ok)
	}
}

func TestAppend(t *testing.T) {
	v := New[int](2)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Append(3, 4)
	expected := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("Append(3,4) datas:%v, expected:%v", v.Datas(), expected)
	}
}

func TestLenCap(t *testing.T) {
	v := New[int](2, 5)
	if v.Len() != 2 {
		t.Errorf("Len() %d != 2", v.Len())
	}
	if v.Cap() != 5 {
		t.Errorf("Cap() %d != 5", v.Cap())
	}
}

func TestInsert(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	ok := v.Insert(1, 4, 5)
	if !ok {
		t.Errorf("Insert(1,4,5) failed")
	}
	expected := []int{1, 4, 5, 2, 3}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("Insert(1,4,5) datas:%v, expected:%v", v.Datas(), expected)
	}
	ok = v.Insert(10, 6)
	if ok {
		t.Errorf("Insert(10,6) should fail")
	}
	ok = v.Insert(-1, 6)
	if ok {
		t.Errorf("Insert(-1,6) should fail")
	}
}

func TestRemove(t *testing.T) {
	v := New[int](4)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	v.Set(3, 4)
	ok := v.Remove(1)
	if !ok {
		t.Errorf("Remove(1) failed")
	}
	expected := []int{1, 3, 4}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("Remove(1) datas:%v, expected:%v", v.Datas(), expected)
	}
	ok = v.Remove(10)
	if ok {
		t.Errorf("Remove(10) should fail")
	}
}

func TestRemoveRange(t *testing.T) {
	v := New[int](5)
	for i := 0; i < 5; i++ {
		v.Set(i, i+1)
	}
	ok := v.RemoveRange(1, 3)
	if !ok {
		t.Errorf("RemoveRange(1,3) failed")
	}
	expected := []int{1, 4, 5}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("RemoveRange(1,3) datas:%v, expected:%v", v.Datas(), expected)
	}
	ok = v.RemoveRange(2, 1) // start > end
	if ok {
		t.Errorf("RemoveRange(2,1) should fail")
	}
	ok = v.RemoveRange(-1, 2)
	if ok {
		t.Errorf("RemoveRange(-1,2) should fail")
	}
	ok = v.RemoveRange(1, 10)
	if ok {
		t.Errorf("RemoveRange(1,10) should fail")
	}
}

func TestRemoveFast(t *testing.T) {
	v := New[int](4)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	v.Set(3, 4)
	ok := v.RemoveFast(1)
	if !ok {
		t.Errorf("RemoveFast(1) failed")
	}
	// Order may change, check length and contents
	if v.Len() != 3 {
		t.Errorf("RemoveFast(1) len:%d != 3", v.Len())
	}
	contains := false
	for _, val := range v.Datas() {
		if val == 2 {
			contains = true
			break
		}
	}
	if contains {
		t.Errorf("RemoveFast(1) still contains 2: %v", v.Datas())
	}
	ok = v.RemoveFast(10)
	if ok {
		t.Errorf("RemoveFast(10) should fail")
	}
}

func TestClear(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	v.Clear()
	if v.Len() != 0 {
		t.Errorf("Clear() len:%d != 0", v.Len())
	}
}

func TestContainsFunc(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	contains := v.ContainsFunc(func(x int) bool { return x == 2 })
	if !contains {
		t.Errorf("ContainsFunc(x==2) false")
	}
	contains = v.ContainsFunc(func(x int) bool { return x == 4 })
	if contains {
		t.Errorf("ContainsFunc(x==4) true")
	}
}

func TestIsValidIndex(t *testing.T) {
	v := New[int](3)
	if !v.IsValidIndex(0) || !v.IsValidIndex(2) {
		t.Errorf("IsValidIndex(0) or (2) false")
	}
	if v.IsValidIndex(3) || v.IsValidIndex(-1) {
		t.Errorf("IsValidIndex(3) or (-1) true")
	}
}

func TestIsEmpty(t *testing.T) {
	v := New[int]()
	if !v.IsEmpty() {
		t.Errorf("New() IsEmpty() false")
	}
	v.Append(1)
	if v.IsEmpty() {
		t.Errorf("After Append IsEmpty() true")
	}
}

func TestClone(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	clone := v.Clone()
	if !reflect.DeepEqual(v.Datas(), clone.Datas()) {
		t.Errorf("Clone datas not equal: %v != %v", v.Datas(), clone.Datas())
	}
	// Modify original, clone should not change
	v.Set(0, 0)
	if clone.Datas()[0] != 1 {
		t.Errorf("Clone modified: %v", clone.Datas())
	}
}

func TestSortFunc(t *testing.T) {
	v := New[int](3)
	v.Set(0, 3)
	v.Set(1, 1)
	v.Set(2, 2)
	v.SortFunc(func(a, b int) int { return a - b })
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("SortFunc datas:%v, expected:%v", v.Datas(), expected)
	}
}

func TestSortStableFunc(t *testing.T) {
	v := New[int](3)
	v.Set(0, 3)
	v.Set(1, 1)
	v.Set(2, 2)
	v.SortStableFunc(func(a, b int) int { return a - b })
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("SortStableFunc datas:%v, expected:%v", v.Datas(), expected)
	}
}

func TestReverse(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	v.Reverse()
	expected := []int{3, 2, 1}
	if !reflect.DeepEqual(v.Datas(), expected) {
		t.Errorf("Reverse datas:%v, expected:%v", v.Datas(), expected)
	}
}

func TestEqualFunc(t *testing.T) {
	v1 := New[int](3)
	v1.Set(0, 1)
	v1.Set(1, 2)
	v1.Set(2, 3)
	v2 := New[int](3)
	v2.Set(0, 1)
	v2.Set(1, 2)
	v2.Set(2, 3)
	equal := v1.EqualFunc(v2, func(a, b int) bool { return a == b })
	if !equal {
		t.Errorf("EqualFunc true expected")
	}
	v2.Set(2, 4)
	equal = v1.EqualFunc(v2, func(a, b int) bool { return a == b })
	if equal {
		t.Errorf("EqualFunc false expected")
	}
	equal = v1.EqualFunc(nil, func(a, b int) bool { return a == b })
	if equal {
		t.Errorf("EqualFunc with nil false expected")
	}
}

func TestRange(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	sum := 0
	v.Range(func(index int, val int) {
		sum += val
	})
	if sum != 6 {
		t.Errorf("Range sum:%d != 6", sum)
	}
}

func TestString(t *testing.T) {
	v := New[int](3)
	v.Set(0, 1)
	v.Set(1, 2)
	v.Set(2, 3)
	str := v.String()
	expected := "[1, 2, 3]"
	if str != expected {
		t.Errorf("String() %s != %s", str, expected)
	}
	v = New[int]()
	str = v.String()
	expected = "[]"
	if str != expected {
		t.Errorf("String() %s != %s", str, expected)
	}
}
