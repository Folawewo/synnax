package rand

import (
	"github.com/samber/lo"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func MapKey[K comparable, V any](m map[K]V) (key K) {
	l := len(m)
	if l == 0 {
		return key
	}
	i, ri := 0, rand.Intn(l)
	for k, _ := range m {
		if i == ri {
			key = k
			break
		}
		i++
	}
	return key
}

func MapValue[K comparable, V any](m map[K]V) V {
	return m[MapKey[K, V](m)]
}

func MapElem[K comparable, V any](m map[K]V) (K, V) {
	k := MapKey[K, V](m)
	return k, m[k]
}

func SubMap[K comparable, V any](m map[K]V, n int) map[K]V {
	om := make(map[K]V)
	i := 0
	for {
		k, v := MapElem[K, V](m)
		_, ok := om[k]
		if !ok {
			om[k] = v
			i++
		}
		if i >= n {
			break
		}
	}
	return om
}

func Elem[V any](options ...V) V {
	return Slice[V](options)
}

func Slice[V any](slice []V) V {
	return slice[rand.Intn(len(slice))]
}

func SubSlice[V comparable](slice []V, n int) []V {
	if n >= len(slice) {
		return slice
	}
	var (
		c        = 0
		subSlice = make([]V, n)
		indexes  = make([]int, n)
	)
	for {
		if c == n {
			break
		}
		i := rand.Intn(len(slice))
		if lo.Contains(indexes, i) {
			continue
		}
		indexes[c] = i
		subSlice[c] = slice[i]
		c++
	}
	return subSlice
}
