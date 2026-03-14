package raf

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

type cachedField struct {
	index int
	name  string
}

type structFields struct {
	fields []cachedField  // sorted by name, for marshal
	byName map[string]int // name to struct field index, for unmarshal
}

func computeStructFields(rt reflect.Type) *structFields {
	n := rt.NumField()
	fields := make([]cachedField, 0, n)
	byName := make(map[string]int, n)
	for i := range n {
		f := rt.Field(i)
		name, skip := fieldName(f)
		if skip {
			continue
		}
		fields = append(fields, cachedField{index: i, name: name})
		byName[name] = i
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})
	return &structFields{fields: fields, byName: byName}
}

// Marshaler encodes Go values to RAF format, caching reflect metadata for struct types.
// A zero-value Marshaler is ready to use.
type Marshaler struct {
	cache       sync.Map // reflect.Type to *structFields
	builderPool *sync.Pool
}

func (m *Marshaler) getStructFields(rt reflect.Type) *structFields {
	if v, ok := m.cache.Load(rt); ok {
		return v.(*structFields)
	}
	sf := computeStructFields(rt)
	v, _ := m.cache.LoadOrStore(rt, sf)
	return v.(*structFields)
}

// Marshal returns the RAF encoding of v.
// v must be a struct, a map with string keys, or a pointer to one of them.
func (m *Marshaler) Marshal(v any) ([]byte, error) {
	if m.builderPool == nil {
		m.builderPool = &sync.Pool{New: func() any { return NewBuilder() }}
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, errors.New("raf: Marshal(nil)")
	}

	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil, errors.New("raf: Marshal called with nil pointer or interface")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Map && rv.Kind() != reflect.Struct {
		return nil, errors.New("raf: Marshal called with unsupported root type, must be struct or map")
	}

	builder := m.builderPool.Get().(*Builder)
	defer m.builderPool.Put(builder)
	builder.Reset()

	if err := m.marshalMapOrStruct(builder, rv); err != nil {
		return nil, err
	}
	return builder.Build(nil)
}

func (m *Marshaler) marshalToBuilder(builder *Builder, rv reflect.Value, key string) error {
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return builder.AddNull(key)
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.String:
		return builder.AddStringString(key, rv.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return builder.AddInt64(key, rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return builder.AddInt64(key, int64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		return builder.AddFloat64(key, rv.Float())
	case reflect.Bool:
		return builder.AddBool(key, rv.Bool())
	case reflect.Map, reflect.Struct:
		if rv.Kind() == reflect.Map && rv.IsNil() {
			return builder.AddNull(key)
		}
		return builder.AddMapFn(key, func(inner *Builder) error {
			return m.marshalMapOrStruct(inner, rv)
		})
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return builder.AddNull(key)
		}
		return m.marshalArray(builder, rv, key)
	default:
		return fmt.Errorf("raf: unsupported type %s", rv.Type().String())
	}
}

func (m *Marshaler) marshalMapOrStruct(builder *Builder, rv reflect.Value) error {
	if rv.Kind() == reflect.Struct {
		sf := m.getStructFields(rv.Type())
		for _, f := range sf.fields {
			if err := m.marshalToBuilder(builder, rv.Field(f.index), f.name); err != nil {
				return err
			}
		}
		return nil
	}

	type kv struct {
		key string
		val reflect.Value
	}

	if rv.Type().Key().Kind() != reflect.String {
		return errors.New("raf: map key must be string")
	}

	pairs := make([]kv, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		pairs = append(pairs, kv{key: k.String(), val: rv.MapIndex(k)})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	for _, p := range pairs {
		if err := m.marshalToBuilder(builder, p.val, p.key); err != nil {
			return err
		}
	}
	return nil
}

func (m *Marshaler) marshalArray(builder *Builder, rv reflect.Value, key string) error {
	if rv.Len() == 0 {
		return builder.AddStringArray(key, nil)
	}

	elemType := rv.Type().Elem()
	kind := elemType.Kind()

	for kind == reflect.Pointer {
		elemType = elemType.Elem()
		kind = elemType.Kind()
	}

	switch kind {
	case reflect.String:
		vals := make([]string, rv.Len())
		for i := range rv.Len() {
			vals[i] = indirect(rv.Index(i)).String()
		}
		return builder.AddStringStringArray(key, vals)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vals := make([]int64, rv.Len())
		for i := range rv.Len() {
			vals[i] = indirect(rv.Index(i)).Int()
		}
		return builder.AddInt64Array(key, vals)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vals := make([]int64, rv.Len())
		for i := range rv.Len() {
			vals[i] = int64(indirect(rv.Index(i)).Uint())
		}
		return builder.AddInt64Array(key, vals)
	case reflect.Float32, reflect.Float64:
		vals := make([]float64, rv.Len())
		for i := range rv.Len() {
			vals[i] = indirect(rv.Index(i)).Float()
		}
		return builder.AddFloat64Array(key, vals)
	case reflect.Bool:
		vals := make([]bool, rv.Len())
		for i := range rv.Len() {
			vals[i] = indirect(rv.Index(i)).Bool()
		}
		return builder.AddBoolArray(key, vals)
	case reflect.Map, reflect.Struct:
		return builder.addMapArrayFromFn(key, rv.Len(), func(i int, inner *Builder) error {
			return m.marshalMapOrStruct(inner, indirect(rv.Index(i)))
		})
	}
	return fmt.Errorf("raf: unsupported array element type %s", elemType.String())
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Zero(v.Type().Elem())
		}
		v = v.Elem()
	}
	return v
}
