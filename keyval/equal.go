package keyval

import (
	"bytes"
	"math/big"
)

func (x Int) Eq(e interface{}) bool {
	return x == e
}

func (x Uint) Eq(e interface{}) bool {
	return x == e
}

func (x Bool) Eq(e interface{}) bool {
	return x == e
}

func (x Float) Eq(e interface{}) bool {
	return x == e
}

func (x String) Eq(e interface{}) bool {
	return x == e
}

func (x UUID) Eq(e interface{}) bool {
	return x == e
}

func (x BigInt) Eq(e interface{}) bool {
	v, ok := e.(BigInt)
	if !ok {
		return false
	}

	X, V := big.Int(x), big.Int(v)
	return X.Cmp(&V) == 0
}

func (x Bytes) Eq(e interface{}) bool {
	v, ok := e.(Bytes)
	if !ok {
		return false
	}
	return bytes.Equal(x, v)
}

func (x KeyValue) Eq(e interface{}) bool {
	v, ok := e.(KeyValue)
	if !ok {
		return false
	}
	return x.Key.Eq(v.Key) && x.Value.Eq(v.Value)
}

func (x Key) Eq(e interface{}) bool {
	v, ok := e.(Key)
	if !ok {
		return false
	}
	return x.Directory.Eq(v.Directory) && x.Tuple.Eq(v.Tuple)
}

func (x Variable) Eq(e interface{}) bool {
	v, ok := e.(Variable)
	if !ok {
		return false
	}
	if len(x) != len(v) {
		return false
	}
	for i := range x {
		if x[i] != v[i] {
			return false
		}
	}
	return true
}

func (x Directory) Eq(e interface{}) bool {
	v, ok := e.(Directory)
	if !ok {
		return false
	}
	if len(v) != len(x) {
		return false
	}
	for i := range x {
		if x[i] != v[i] {
			return false
		}
	}
	return true
}

func (x Tuple) Eq(e interface{}) bool {
	v, ok := e.(Tuple)
	if !ok {
		return false
	}
	if len(x) != len(v) {
		return false
	}
	for i := range x {
		if !x[i].Eq(v[i]) {
			return false
		}
	}
	return true
}

func (x Nil) Eq(e interface{}) bool {
	_, ok := e.(Nil)
	return ok
}

func (x Clear) Eq(e interface{}) bool {
	_, ok := e.(Clear)
	return ok
}

func (x MaybeMore) Eq(e interface{}) bool {
	_, ok := e.(MaybeMore)
	return ok
}
