package slice

import (
	"iter"

	"github.com/james-orcales/golang_snacks/invariant"
)

type Fixed[T any] struct {
	data []T
}

func Make[T any](length, capacity int) Fixed[T] {
	return Fixed[T]{data: make([]T, length, capacity)}
}

func Len[T any](f *Fixed[T]) int {
	return len(f.data)
}

func Cap[T any](f *Fixed[T]) int {
	return cap(f.data)
}

func Get[T any](f *Fixed[T], i int) T {
	return f.data[i]
}

func Set[T any](f *Fixed[T], i int, v T) {
	f.data[i] = v
}

func Append[T any](f *Fixed[T], items ...T) {
	invariant.Always(len(f.data)+len(items) <= cap(f.data), "slice.Append doesn't exceed capacity")
	f.data = append(f.data, items...)
}

func Subslice[T any](f *Fixed[T], low, high int) *Fixed[T] {
	invariant.Always(high <= cap(f.data), "slice.Subslice high doesn't exceed capacity")
	return &Fixed[T]{data: f.data[low:high]}
}

func Resize[T any](f *Fixed[T], length int) {
	invariant.Always(length <= cap(f.data), "slice.Resize length doesn't exceed capacity")
	f.data = f.data[:length]
}

func Copy[T any](dst, src *Fixed[T]) int {
	return copy(dst.data, src.data)
}

func Clear[T any](f *Fixed[T]) {
	clear(f.data)
}

func All[T any](f *Fixed[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, v := range f.data {
			if !yield(i, v) {
				return
			}
		}
	}
}

func Values[T any](f *Fixed[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range f.data {
			if !yield(v) {
				return
			}
		}
	}
}
