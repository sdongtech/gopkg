// Copyright 2021 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metainfo

type ctxKeyType struct{}

var ctxKey ctxKeyType

type kv struct {
	key string
	val string
}

func newNodeFromMaps(persistent, transient, stale kvstore) *node {
	ps, ts, sz := persistent.size(), transient.size(), stale.size()
	// make slices together to reduce malloc cost
	kvs := make([]kv, ps+ts+sz)
	nd := new(node)
	nd.persistent = kvs[:ps]
	nd.transient = kvs[ps : ps+ts]
	nd.stale = kvs[ps+ts:]

	i := 0
	for k, v := range persistent {
		nd.persistent[i].key, nd.persistent[i].val = k, v
		i++
	}
	i = 0
	for k, v := range transient {
		nd.transient[i].key, nd.transient[i].val = k, v
		i++
	}
	i = 0
	for k, v := range stale {
		nd.stale[i].key, nd.stale[i].val = k, v
		i++
	}
	return nd
}

type node struct {
	persistent []kv
	transient  []kv
	stale      []kv
}

func (n *node) size() int {
	return len(n.persistent) + len(n.transient) + len(n.stale)
}

func (n *node) transferForward() (r *node) {
	r = &node{
		persistent: n.persistent,
		stale:      n.transient,
	}
	return
}

func (n *node) addTransient(k, v string) *node {
	if res, ok := remove(n.stale, k); ok {
		return &node{
			persistent: n.persistent,
			transient: appendEx(n.transient, kv{
				key: k,
				val: v,
			}),
			stale: res,
		}
	}

	if idx, ok := search(n.transient, k); ok {
		if n.transient[idx].val == v {
			return n
		}
		r := *n
		r.transient = make([]kv, len(n.transient))
		copy(r.transient, n.transient)
		r.transient[idx].val = v
		return &r
	}

	r := *n
	r.transient = appendEx(r.transient, kv{
		key: k,
		val: v,
	})
	return &r
}

func (n *node) addPersistent(k, v string) *node {
	if idx, ok := search(n.persistent, k); ok {
		if n.persistent[idx].val == v {
			return n
		}
		r := *n
		r.persistent = make([]kv, len(n.persistent))
		copy(r.persistent, n.persistent)
		r.persistent[idx].val = v
		return &r
	}
	r := *n
	r.persistent = appendEx(r.persistent, kv{
		key: k,
		val: v,
	})
	return &r
}

func (n *node) delTransient(k string) (r *node) {
	if res, ok := remove(n.stale, k); ok {
		return &node{
			persistent: n.persistent,
			transient:  n.transient,
			stale:      res,
		}
	}
	if res, ok := remove(n.transient, k); ok {
		return &node{
			persistent: n.persistent,
			transient:  res,
			stale:      n.stale,
		}
	}
	return n
}

func (n *node) delPersistent(k string) (r *node) {
	if res, ok := remove(n.persistent, k); ok {
		return &node{
			persistent: res,
			transient:  n.transient,
			stale:      n.stale,
		}
	}
	return n
}

func search(kvs []kv, key string) (idx int, ok bool) {
	for i := range kvs {
		if kvs[i].key == key {
			return i, true
		}
	}
	return
}

func remove(kvs []kv, key string) (res []kv, removed bool) {
	if idx, ok := search(kvs, key); ok {
		if cnt := len(kvs); cnt == 1 {
			removed = true
			return
		}
		res = make([]kv, len(kvs)-1)
		copy(res, kvs[:idx])
		copy(res[idx:], kvs[idx+1:])
		return res, true
	}
	return kvs, false
}

func getNode(ctx context.Context) *node {
	if ctx != nil {
		if val, ok := ctx.Value(ctxKey).(*node); ok {
			return val
		}
	}
	return nil
}

func withNode(ctx context.Context, n *node) context.Context {
	if ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey, n)
}

func appendEx(arr []kv, x kv) (res []kv) {
	res = make([]kv, len(arr)+1)
	copy(res, arr)
	res[len(arr)] = x
	return res
}





