package reader

import (
	"context"
	"math/big"
	"sync"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	dir "github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	tup "github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/janderland/fdbq/keyval"
	"github.com/pkg/errors"
)

type Reader struct {
	tr     fdb.Transaction
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	errCh  chan error
}

type DirKeyValue struct {
	dir dir.DirectorySubspace
	kv  fdb.KeyValue
}

func New(tr fdb.Transaction) Reader {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	return Reader{
		tr:     tr,
		wg:     &wg,
		ctx:    ctx,
		cancel: cancel,
		errCh:  make(chan error),
	}
}

func (c *Reader) Wait() error {
	go func() {
		c.wg.Wait()
		close(c.errCh)
	}()

	for err := range c.errCh {
		return err
	}
	return nil
}

func (c *Reader) signalError(err error) {
	select {
	case c.errCh <- err:
		c.cancel()
	case <-c.ctx.Done():
	}
}

func (c *Reader) openDirectories(directory keyval.Directory) chan dir.DirectorySubspace {
	dirCh := make(chan dir.DirectorySubspace)
	c.wg.Add(1)

	go func() {
		defer close(dirCh)
		defer c.wg.Done()
		c.doOpenDirectories(directory, dirCh)
	}()

	return dirCh
}

func (c *Reader) doOpenDirectories(directory keyval.Directory, dirCh chan dir.DirectorySubspace) {
	prefix, variable, suffix := splitAtFirstVariable(directory)
	prefixStr := toStringArray(prefix)

	if variable != nil {
		subDirs, err := dir.List(c.tr, prefixStr)
		if err != nil {
			c.signalError(errors.Wrap(err, "failed to list directories"))
			return
		}

		for _, sDir := range subDirs {
			var directory keyval.Directory
			directory = append(directory, prefix...)
			directory = append(directory, sDir)
			directory = append(directory, suffix...)
			c.doOpenDirectories(directory, dirCh)
		}
	} else {
		directory, err := dir.Open(c.tr, prefixStr, nil)
		if err != nil {
			c.signalError(errors.Wrap(err, "failed to open directory"))
			return
		}

		select {
		case <-c.ctx.Done():
			return
		case dirCh <- directory:
		}
	}
}

func (c *Reader) readRange(tuple keyval.Tuple, dirCh chan dir.DirectorySubspace) chan DirKeyValue {
	kvCh := make(chan DirKeyValue)
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		c.wg.Add(1)
		wg.Add(1)

		go func() {
			defer c.wg.Done()
			defer wg.Done()
			c.doReadRange(toFDBTuple(tuple), dirCh, kvCh)
		}()
	}

	go func() {
		defer close(kvCh)
		wg.Wait()
	}()

	return kvCh
}

func (c *Reader) doReadRange(tuple tup.Tuple, dirCh chan dir.DirectorySubspace, kvCh chan DirKeyValue) {
	read := func() (dir.DirectorySubspace, bool) {
		select {
		case <-c.ctx.Done():
			return nil, false
		case directory, open := <-dirCh:
			return directory, open
		}
	}

	for directory, running := read(); running; directory, running = read() {
		rng, err := fdb.PrefixRange(directory.Pack(tuple))
		if err != nil {
			c.signalError(errors.Wrap(err, "failed to create prefix range"))
			return
		}

		iter := c.tr.GetRange(rng, fdb.RangeOptions{}).Iterator()
		for iter.Advance() {
			kv, err := iter.Get()
			if err != nil {
				c.signalError(errors.Wrap(err, "failed to get key-value"))
				return
			}

			select {
			case <-c.ctx.Done():
				return
			case kvCh <- DirKeyValue{dir: directory, kv: kv}:
			}
		}
	}
}

func (c *Reader) filterRange(tuple keyval.Tuple, in chan DirKeyValue) chan DirKeyValue {
	out := make(chan DirKeyValue)
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		c.wg.Add(1)
		wg.Add(1)

		go func() {
			defer c.wg.Done()
			defer wg.Done()
			c.doFilterRange(toFDBTuple(tuple), in, out)
		}()
	}

	go func() {
		defer close(out)
		wg.Wait()
	}()

	return out
}

func (c *Reader) doFilterRange(tuple tup.Tuple, in chan DirKeyValue, out chan DirKeyValue) {
	read := func() (DirKeyValue, bool) {
		select {
		case <-c.ctx.Done():
			return DirKeyValue{}, false
		case directory, open := <-in:
			return directory, open
		}
	}

	for dkv, running := read(); running; dkv, running = read() {
		otherTuple, err := dkv.dir.Unpack(dkv.kv.Key)
		if err != nil {
			c.signalError(errors.Wrap(err, "failed to unpack key"))
		}

		if compareTuples(tuple, otherTuple) == -1 {
			select {
			case <-c.ctx.Done():
				return
			case out <- dkv:
			}
		}
	}
}

func compareTuples(pattern tup.Tuple, candidate tup.Tuple) int {
	if len(pattern) < len(candidate) {
		return len(pattern)
	}
	if len(pattern) > len(candidate) {
		return len(candidate)
	}

	var i int
	var e tup.TupleElement
	err := keyval.ParseTuple(candidate, func(p *keyval.TupleParser) {
		for i, e = range pattern {
			switch e.(type) {
			// TODO: Ensure all possible FDB tuple elements are covered.
			case int64:
				if p.Int() != e.(int64) {
					return
				}
			case uint64:
				if p.Uint() != e.(uint64) {
					return
				}
			case string:
				if p.String() != e.(string) {
					return
				}
			case *big.Int:
				if p.BigInt().Cmp(e.(*big.Int)) != 0 {
					return
				}
			case float32:
				if p.Float() != float64(e.(float32)) {
					return
				}
			case float64:
				if p.Float() != e.(float64) {
					return
				}
			case bool:
				if p.Bool() != e.(bool) {
					return
				}
			}
		}
	})
	if err != nil {
		return i
	}
	return -1
}

func splitAtFirstVariable(list []interface{}) ([]interface{}, *keyval.Variable, []interface{}) {
	for i, segment := range list {
		switch segment.(type) {
		case keyval.Variable:
			v := segment.(keyval.Variable)
			return list[:i], &v, list[i+1:]
		}
	}
	return list, nil, nil
}

func toStringArray(in []interface{}) []string {
	out := make([]string, len(in))
	for i := range in {
		out[i] = in[i].(string)
	}
	return out
}

func toFDBTuple(in []interface{}) tup.Tuple {
	out := make(tup.Tuple, len(in))
	for i := range in {
		out[i] = tup.TupleElement(in[i])
	}
	return out
}