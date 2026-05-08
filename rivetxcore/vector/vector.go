package vector

import (
	"fmt"
	"slices"
	"strings"
)

func SliceGet[T any](slice []T, index int) (T, bool) {
	if index < 0 || index >= len(slice) {
		var zero T
		return zero, false
	}
	return slice[index], true
}

func SliceSet[T any](slice []T, index int, value T) bool {
	if index < 0 || index >= len(slice) {
		return false
	}
	slice[index] = value
	return true
}

// 只有当 T 是 comparable 时，这些函数才能被调用
func Contains[T comparable](v *Vector[T], val T) bool {
	return slices.Contains(v.datas, val)
}

func Equal[T comparable](v1, v2 *Vector[T]) bool {
	return slices.Equal(v1.datas, v2.datas)
}

type Vector[T any] struct {
	datas []T
}

func New[T any](size ...int) *Vector[T] {
	length := 0
	capacity := 0
	if len(size) > 0 {
		length = size[0]
		capacity = size[0]
	}
	if len(size) > 1 {
		capacity = size[1]
	}
	if capacity < length {
		capacity = length
	}
	return &Vector[T]{
		datas: make([]T, length, capacity),
	}
}

func FromSlice[T any](datas []T) *Vector[T] {
	return &Vector[T]{
		datas: datas,
	}
}

func (self *Vector[T]) Get(index int) (T, bool) {
	return SliceGet(self.datas, index)
}

func (self *Vector[T]) Set(index int, value T) bool {
	return SliceSet(self.datas, index, value)
}

func (self *Vector[T]) Append(elems ...T) {
	self.datas = append(self.datas, elems...)
}

func (self *Vector[T]) Len() int {
	return len(self.datas)
}

func (self *Vector[T]) Cap() int {
	return cap(self.datas)
}

func (self *Vector[T]) Datas() []T {
	return self.datas
}

func (self *Vector[T]) MoveDatas(datas []T) {
	self.datas = datas
}

func (self *Vector[T]) CopyDatas(datas []T) {
	self.datas = slices.Clone(datas)
}

func (self *Vector[T]) Insert(index int, elems ...T) bool {
	if index < 0 || index > len(self.datas) {
		return false
	}

	if len(elems) == 0 {
		return true
	}

	self.datas = slices.Insert(self.datas, index, elems...)

	return true
}

func (self *Vector[T]) Remove(index int) bool {
	if index < 0 || index >= len(self.datas) {
		return false
	}
	self.datas = slices.Delete(self.datas, index, index+1)
	return true
}

func (self *Vector[T]) RemoveRange(startIndex int, endIndex int) bool {
	if startIndex > endIndex {
		return false
	}
	if startIndex < 0 || startIndex >= len(self.datas) {
		return false
	}

	if endIndex > len(self.datas) {
		return false
	}

	if startIndex == endIndex {
		return true
	}

	self.datas = slices.Delete(self.datas, startIndex, endIndex)
	return true
}

// 此方法会改变元素顺序
func (self *Vector[T]) RemoveFast(index int) bool {
	if index < 0 || index >= len(self.datas) {
		return false
	}
	self.datas[index] = self.datas[len(self.datas)-1]
	self.datas = self.datas[:len(self.datas)-1]
	return true
}

func (self *Vector[T]) Clear() {
	self.datas = self.datas[:0]
}

func (self *Vector[T]) ContainsFunc(f func(T) bool) bool {
	return slices.ContainsFunc(self.datas, f)
}

func (self *Vector[T]) IsValidIndex(index int) bool {
	return index >= 0 && index < len(self.datas)
}

func (self *Vector[T]) IsEmpty() bool {
	return len(self.datas) == 0
}

func (self *Vector[T]) Clone() *Vector[T] {
	return &Vector[T]{datas: slices.Clone(self.datas)}
}

func (self *Vector[T]) SortFunc(cmp func(a, b T) int) {
	slices.SortFunc(self.datas, cmp)
}

func (self *Vector[T]) SortStableFunc(cmp func(a, b T) int) {
	slices.SortStableFunc(self.datas, cmp)
}

func (self *Vector[T]) Reverse() {
	slices.Reverse(self.datas)
}

func (self *Vector[T]) EqualFunc(other *Vector[T], eq func(a, b T) bool) bool {
	if other == nil {
		return false
	}
	return slices.EqualFunc(self.datas, other.Datas(), eq)
}

func (self *Vector[T]) Range(f func(int, T)) {
	for index, data := range self.datas {
		f(index, data)
	}
}

func (self *Vector[T]) String() string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range self.datas {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprint(&sb, v)
	}
	sb.WriteByte(']')
	return sb.String()
}
