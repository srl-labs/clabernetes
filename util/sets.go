package util

// StringSet defines a simple string set interface.
type StringSet interface {
	Add(s string)
	Extend(sSlice []string)
	Remove(s string)
	Contains(s string) bool
	Len() int
	Items() []string
}

// NewStringSet returns an object satisfying the StringSet interface.
func NewStringSet() StringSet {
	ss := &stringSet{
		objs: make(map[string]interface{}),
	}

	return ss
}

// NewStringSetWithValues returns an object satisfying the StringSet interface with the given
// values pre-loaded to the set.
func NewStringSetWithValues(vals ...string) StringSet {
	ss := &stringSet{
		objs: make(map[string]interface{}),
	}

	for _, s := range vals {
		ss.Add(s)
	}

	return ss
}

type stringSet struct {
	objs map[string]interface{}
}

func (ss *stringSet) Add(s string) {
	_, ok := ss.objs[s]
	if ok {
		return
	}

	ss.objs[s] = nil
}

func (ss *stringSet) Extend(sSlice []string) {
	for _, s := range sSlice {
		ss.Add(s)
	}
}

func (ss *stringSet) Remove(s string) {
	if !ss.Contains(s) {
		return
	}

	delete(ss.objs, s)
}

func (ss *stringSet) Contains(s string) bool {
	_, ok := ss.objs[s]

	return ok
}

func (ss *stringSet) Len() int {
	return len(ss.objs)
}

func (ss *stringSet) Items() []string {
	items := make([]string, len(ss.objs))

	idx := 0

	for k := range ss.objs {
		items[idx] = k

		idx++
	}

	return items
}
