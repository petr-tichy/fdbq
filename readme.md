# FDBQ

FDBQ provides a query language and an alternative client API
for FoundationDB. Some of the things this project aims to
acheive are:
- [x] Provide a textual description of key-value schemas.
- [x] Provide an intuitive query language for FDB.
- [x] Provide a Go API which is structurally equivalent to
  the query language.
- [ ] Improve the ergonomics of the FoundationDB API.
  - [ ] Simplify the directory, tuple, & core APIs via
    a single unified API.
  - [ ] Gracefully handle multi-transaction range-reads.
  - [ ] Gracefully handle transient errors.
- [ ] Standardize the encoding of primitives (int, float,
  bool) as FDB values.

Here is the [syntax definition](syntax.ebnf) for the query
language.

## Docker

FDBQ is available as a Docker image for running queries. The
first argument passed to the container is the contents of
the cluster file. The remaning arguments are passed to the
FDBQ binary.

```bash
# 'my_cluster:baoeA32@172.20.3.33:4500' is used as the contents
# for the cluster file. '-log' and '/my/dir{<>}=42' are passed
# as args to the FDBQ binary.
docker run docker.io/janderland/fdbq 'my_cluster:baoeA32@172.20.3.33:4500' -log '/my/dir{<>}=42'
```

The cluster file contents (first argument) is evaluated by
Bash within the container before being written to disk,
which allows for converting hostnames into IPs.

```bash
# The cluster file contents includes a bit of Bash which
# converts the hostname 'fdb' to an IP address before
# writing the cluster file on to the container's disk.
CFILE='docker:docker@$(getent hosts fdb | cut -d" " -f1):4500'
docker run docker.io/janderland/fdbq $CFILE -log '/my/dir{<>}=42'
```

## Examples

The following examples showcase FDBQ queries and the
equivalent FDB API calls implemented in Go.

### Set

```fdbq
/my/dir{"hello", "world"}=42
```

```Go
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.CreateOrOpen(tr, []string{"my", "dir"}, nil)
  if err != nil {
    return nil, err
  }

  val := make([]byte, 8)
  binary.LittleEndian.PutUint64(val, 42)
  tr.Set(dir.Pack(tuple.Tuple{"hello", "world"}), val)
  return nil, nil
})
```

### Clear

```fdbq
/my/dir{"hello", "world"}=clear
```

```Go
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"my", "dir"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  tr.Clear(dir.Pack(tuple.Tuple{"hello", "world"}))
  return nil, nil
})
```

### Get Single Key

```fdbq
/my/dir{99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136}
```

```Go
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"my", "dir"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  return tr.Get(dir.Pack(tuple.Tuple{99.8,
    tuple.UUID{0x7d, 0xfb, 0x10, 0xd1, 0x24, 0x93, 0x4f, 0xb5, 0x92, 0x8e, 0x88, 0x9f, 0xdc, 0x6a, 0x71, 0x36}))
})
```

### Read & Filter Range of Keys

```fdbq
/people{3392, <string|int>, <>}={<uint>, ...}
```

```Go
db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"people"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  rng, err := fdb.PrefixRange(dir.Pack(tuple.Tuple{3392}))
  if err != nil {
    return nil, err
  }

  var results []fdb.KeyValue
  iter := tr.GetRange(rng, fdb.RangeOptions{}).Iterator()
  for iter.Advance() {
    kv := iter.MustGet()

    tup, err := dir.Unpack(kv.Key)
    if err != nil {
      return nil, err
    }

    if len(tup) != 3 {
      continue
    }

    switch tup[0].(type) {
    default:
      continue
    case string | int64:
    }

    val, err := tuple.Unpack(kv.Value)
    if err != nil {
      continue
    }
    if len(val) == 0 {
      continue
    }
    if _, isInt := val[0].(uint64); !isInt {
      continue
    }

    results = append(results, kv)
  }
  return results, nil
})
```

### Read & Filter Directory Paths

```fdbq
/root/<>/items/<>
```

```Go
db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
  root, err := directory.Open(tr, []string{"root"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  oneDeep, err := root.List(tr, nil)
  if err != nil {
    return nil, err
  }

  var results [][]string
  for _, dir1 := range oneDeep {
    twoDeep, err := root.List(tr, []string{dir1, "items"})
    if err != nil {
      return nil, err
    }

    for _, dir2 := range twoDeep {
      results = append(results, []string{"root", dir1, dir2})
    }
  }
  return results, nil
})
```
