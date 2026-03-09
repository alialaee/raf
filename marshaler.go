package raf

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type cachedField struct {
	index int
	name  []byte
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
		skip, name := fieldName(f)
		if skip {
			continue
		}
		fields = append(fields, cachedField{index: i, name: []byte(name)})
		byName[name] = i
	}
	sort.Slice(fields, func(i, j int) bool {
		return bytes.Compare(fields[i].name, fields[j].name) < 0
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

func (m *Marshaler) marshalToBuilder(builder *Builder, rv reflect.Value, key []byte) error {
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
		innerBuilder := NewBuilder()
		if err := m.marshalMapOrStruct(innerBuilder, rv); err != nil {
			return err
		}
		innerBytes, err := innerBuilder.Build(nil)
		if err != nil {
			return err
		}
		return builder.AddMap(key, innerBytes)
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return builder.AddNull(key)
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return builder.AddString(key, rv.Bytes())
		}
		return marshalArray(builder, rv, key)
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
		key []byte
		val reflect.Value
	}

	if rv.Type().Key().Kind() != reflect.String {
		return errors.New("raf: map key must be string")
	}

	pairs := make([]kv, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		pairs = append(pairs, kv{key: []byte(k.String()), val: rv.MapIndex(k)})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return bytes.Compare(pairs[i].key, pairs[j].key) < 0
	})

	for _, p := range pairs {
		if err := m.marshalToBuilder(builder, p.val, p.key); err != nil {
			return err
		}
	}
	return nil
}

func fieldName(f reflect.StructField) (skip bool, name string) {
	tag := f.Tag.Get("raf")
	if !f.IsExported() {
		return true, ""
	}
	if tag == "-" {
		return true, ""
	}
	name = tag
	if name == "" {
		name = strings.ToLower(f.Name)
	}
	return false, name
}

func marshalArray(builder *Builder, rv reflect.Value, key []byte) error {
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

// Unmarshaler decodes RAF data into Go values, caching reflect metadata for struct types.
// A zero-value Unmarshaler is ready to use.
type Unmarshaler struct {
	cache sync.Map // reflect.Type to *structFields
}

func (u *Unmarshaler) getStructFields(rt reflect.Type) *structFields {
	if v, ok := u.cache.Load(rt); ok {
		return v.(*structFields)
	}
	sf := computeStructFields(rt)
	v, _ := u.cache.LoadOrStore(rt, sf)
	return v.(*structFields)
}

// Unmarshal parses the RAF-encoded data and stores the result in the value pointed to by v.
func (u *Unmarshaler) Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("raf: Unmarshal(non-pointer or nil)")
	}
	rv = rv.Elem()

	block := Block(data)
	if !block.Valid() {
		return errors.New("raf: invalid block structure")
	}

	return u.unmarshalMapOrStruct(block, rv)
}

func (u *Unmarshaler) unmarshalMapOrStruct(block Block, rv reflect.Value) error {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Interface {
		m := make(map[string]any)
		for i := 0; i < block.NumPairs(); i++ {
			k := block.KeyAt(i)
			v := block.ValueAt(i)
			val, err := valueToInterface(v)
			if err != nil {
				return err
			}
			m[string(k)] = val
		}
		rv.Set(reflect.ValueOf(m))
		return nil
	}

	if rv.Kind() == reflect.Map {
		if rv.Type().Key().Kind() != reflect.String {
			return errors.New("raf: map key must be string")
		}
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		elemType := rv.Type().Elem()
		for i := 0; i < block.NumPairs(); i++ {
			elemPtr := reflect.New(elemType)
			if err := u.unmarshalValue(block.ValueAt(i), elemPtr.Elem()); err != nil {
				return err
			}
			rv.SetMapIndex(reflect.ValueOf(string(block.KeyAt(i))), elemPtr.Elem())
		}
		return nil
	}

	if rv.Kind() == reflect.Struct {
		sf := u.getStructFields(rv.Type())

		for i := 0; i < block.NumPairs(); i++ {
			if fieldIdx, ok := sf.byName[string(block.KeyAt(i))]; ok {
				if err := u.unmarshalValue(block.ValueAt(i), rv.Field(fieldIdx)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return fmt.Errorf("raf: unsupported type %s to unmarshal into from map", rv.Type().String())
}

func (u *Unmarshaler) unmarshalValue(val Value, rv reflect.Value) error {
	for rv.Kind() == reflect.Pointer {
		if val.Type == TypeNull {
			rv.SetZero()
			return nil
		}
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if val.Type == TypeNull {
		rv.SetZero()
		return nil
	}

	if rv.Kind() == reflect.Interface {
		v, err := valueToInterface(val)
		if err != nil {
			return err
		}
		if v != nil {
			rv.Set(reflect.ValueOf(v))
		} else {
			rv.SetZero()
		}
		return nil
	}

	switch val.Type {
	case TypeString:
		if rv.Kind() == reflect.String {
			rv.SetString(val.String())
		} else if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			rv.SetBytes(append([]byte(nil), val.Data...))
		} else {
			return fmt.Errorf("raf: cannot unmarshal string into %s", rv.Type())
		}
	case TypeInt64:
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			rv.SetInt(val.Int64())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			rv.SetUint(uint64(val.Int64()))
		case reflect.Float32, reflect.Float64:
			rv.SetFloat(float64(val.Int64()))
		default:
			return fmt.Errorf("raf: cannot unmarshal int64 into %s", rv.Type())
		}
	case TypeFloat64:
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			rv.SetFloat(val.Float64())
		default:
			return fmt.Errorf("raf: cannot unmarshal float64 into %s", rv.Type())
		}
	case TypeBool:
		if rv.Kind() == reflect.Bool {
			rv.SetBool(val.Bool())
		} else {
			return fmt.Errorf("raf: cannot unmarshal bool into %s", rv.Type())
		}
	case TypeMap:
		return u.unmarshalMapOrStruct(val.Map(), rv)
	case TypeArray:
		arr := val.Array()
		if rv.Kind() == reflect.Slice {
			rv.Set(reflect.MakeSlice(rv.Type(), arr.Len(), arr.Len()))
			for i := 0; i < arr.Len(); i++ {
				elemVal := Value{Type: arr.ElemType(), Data: arr.At(i)}
				if err := u.unmarshalValue(elemVal, rv.Index(i)); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("raf: cannot unmarshal array into %s", rv.Type())
		}
	}
	return nil
}

func valueToInterface(val Value) (any, error) {
	switch val.Type {
	case TypeString:
		return val.String(), nil
	case TypeInt64:
		return val.Int64(), nil
	case TypeFloat64:
		return val.Float64(), nil
	case TypeBool:
		return val.Bool(), nil
	case TypeNull:
		return nil, nil
	case TypeMap:
		block := val.Map()
		m := make(map[string]any)
		for i := 0; i < block.NumPairs(); i++ {
			innerVal, err := valueToInterface(block.ValueAt(i))
			if err != nil {
				return nil, err
			}
			m[string(block.KeyAt(i))] = innerVal
		}
		return m, nil
	case TypeArray:
		arr := val.Array()
		var slice []any
		for i := 0; i < arr.Len(); i++ {
			elemVal := Value{Type: arr.ElemType(), Data: arr.At(i)}
			v, err := valueToInterface(elemVal)
			if err != nil {
				return nil, err
			}
			slice = append(slice, v)
		}
		return slice, nil
	default:
		return nil, fmt.Errorf("raf: unknown value type %d", val.Type)
	}
}
