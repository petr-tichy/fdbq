package keyval

import (
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"

	"github.com/stretchr/testify/assert"
)

func TestToStringArray(t *testing.T) {
	dir := Directory{"my", "dir", "path"}
	arr, err := ToStringArray(dir)
	assert.NoError(t, err)
	assert.Equal(t, []string{"my", "dir", "path"}, arr)

	dir[1] = Variable{}
	_, err = ToStringArray(dir)
	assert.Error(t, err)
}

func TestFromStringArray(t *testing.T) {
	dir := FromStringArray([]string{"my", "dir", "path"})
	assert.Equal(t, Directory{"my", "dir", "path"}, dir)
}

func TestToFDBTuple(t *testing.T) {
	tup := ToFDBTuple(Tuple{nil, int64(22), false})
	assert.Equal(t, tuple.Tuple{nil, int64(22), false}, tup)
}

func TestFromFDBTuple(t *testing.T) {
	tup := FromFDBTuple(tuple.Tuple{nil, int64(22), false})
	assert.Equal(t, Tuple{nil, int64(22), false}, tup)
}

func TestDirIsVariable(t *testing.T) {
	dir := Directory{"my", "dir", "path"}
	assert.False(t, DirIsVariable(dir))
	dir[1] = Variable{}
	assert.True(t, DirIsVariable(dir))
}

func TestTupIsVariable(t *testing.T) {
	tup := Tuple{true, 22.8, "yup"}
	assert.False(t, TupIsVariable(tup))
	tup[1] = Variable{}
	assert.True(t, TupIsVariable(tup))
	tup[1] = Tuple{"hello", Variable{}}
	assert.True(t, TupIsVariable(tup))
}

func TestValIsVariable(t *testing.T) {
	val := Tuple{true, 22.8, "yup"}
	assert.False(t, ValIsVariable(val))
	val[1] = Variable{}
	assert.True(t, ValIsVariable(val))
	assert.True(t, ValIsVariable(Variable{}))
}

func TestIsVariable(t *testing.T) {
	newKV := func() *KeyValue {
		return &KeyValue{
			Key: Key{
				Directory: Directory{"my", "dir", "path"},
				Tuple:     Tuple{true, 22.8, "yup"},
			},
			Value: Tuple{true, 22.8, "yup"},
		}
	}
	assert.False(t, IsVariable(newKV()))

	kv := newKV()
	kv.Key.Directory[1] = Variable{}
	assert.True(t, IsVariable(kv))

	kv = newKV()
	kv.Key.Tuple[1] = Variable{}
	assert.True(t, IsVariable(kv))

	kv = newKV()
	kv.Value = Variable{}
	assert.True(t, IsVariable(kv))

	kv = newKV()
	kv.Value.(Tuple)[1] = Variable{}
	assert.True(t, IsVariable(kv))
}
