package m2

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type valueAdder interface {
	AddString(value string)
	AddInt64(value int64)
	AddFloat64(value float64)
	AddBool(value bool)
	AddNull()
	AddArrayFn(elemType Type, count int, fn func(ab *ArrayBuilder) error) error
	AddBuilderFn(fn func(mb *Builder) error) error
	AddRaw(value []byte)
}

type reflectField struct {
	index      int
	isNullable bool
	kind       reflect.Kind
}

type structFields struct {
	rafFields     []KeyType
	reflectFields []reflectField
}

func (sf *structFields) Len() int {
	return len(sf.rafFields)
}

func (sf *structFields) Swap(i, j int) {
	sf.rafFields[i], sf.rafFields[j] = sf.rafFields[j], sf.rafFields[i]
	sf.reflectFields[i], sf.reflectFields[j] = sf.reflectFields[j], sf.reflectFields[i]
}

func (sf *structFields) Less(i, j int) bool {
	return sf.rafFields[i].Name < sf.rafFields[j].Name
}

func computeStructFields(rt reflect.Type) (*structFields, error) {
	n := rt.NumField()
	rafFields := make([]KeyType, 0, n)
	reflectFields := make([]reflectField, 0, n)

	for i := range n {
		f := rt.Field(i)
		name, skip := fieldName(f)
		if skip {
			continue
		}

		rafType, err := reflectTypeToRAFType(f.Type)
		if err != nil {
			return nil, errors.New("raf: unsupported field type " + f.Type.String() + " for field " + f.Name)
		}

		rafFields = append(rafFields, KeyType{Name: name, Type: rafType})
		reflectFields = append(reflectFields, reflectField{index: i, isNullable: isNullableType(f.Type), kind: f.Type.Kind()})
	}

	// Sort fields by name for deterministic encoding order
	structFields := &structFields{rafFields: rafFields, reflectFields: reflectFields}
	sort.Stable(structFields)
	return structFields, nil
}

func isNullableType(rt reflect.Type) bool {
	for rt.Kind() == reflect.Pointer {
		return true
	}

	// switch rt.Kind() {
	// case reflect.Slice, reflect.Map:
	// 	return true
	// }

	return false
}

func reflectTypeToRAFType(rt reflect.Type) (Type, error) {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	switch rt.Kind() {
	case reflect.Bool:
		return TypeBool, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInt64, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TypeInt64, nil
	case reflect.Float32, reflect.Float64:
		return TypeFloat64, nil
	case reflect.String:
		return TypeString, nil
	case reflect.Slice:
		return TypeArray, nil
	case reflect.Map:
		if rt.Key().Kind() == reflect.String {
			return TypeMap, nil
		} else {
			return 0, errors.New("raf: map key must be string")
		}
	case reflect.Struct:
		return TypeMap, nil
	}

	return 0, errors.New("raf: unsupported type " + rt.String())
}

type Marshaler struct {
	builderPool      *sync.Pool
	arrayBuilderPool *sync.Pool
	structCache      sync.Map // reflect.Type to *structFields
}

func NewMarshaler() *Marshaler {
	return &Marshaler{
		builderPool: &sync.Pool{
			New: func() any {
				return NewBuilder(make([]byte, 0, 1024))
			},
		},
		arrayBuilderPool: &sync.Pool{
			New: func() any {
				return NewArrayBuilder(make([]byte, 0, 256), TypeString, 0)
			},
		},
	}
}

func (m *Marshaler) Marshal(v any) ([]byte, error) {
	builder := m.builderPool.Get().(*Builder)
	defer m.builderPool.Put(builder)
	builder.Reset()

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, errors.New("raf: Marshal(nil)")
	}

	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil, errors.New("raf: Marshal called with nil pointer or interface")
		}

		rv = rv.Elem() // TODO let's have a for loop here
	}

	if rv.Kind() != reflect.Map && rv.Kind() != reflect.Struct {
		return nil, errors.New("raf: Marshal called with unsupported root type, must be struct or map")
	}

	if err := m.marshalStruct(builder, rv); err != nil {
		return nil, err
	}

	data, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// Copy data to a new slice.

	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

func (m *Marshaler) marshalStruct(builder *Builder, rv reflect.Value) error {
	structFields, err := m.structFieldsForType(rv.Type())
	if err != nil {
		return err
	}

	// Write keys
	builder.AddKeys(structFields.rafFields...)

	// Write values
	for _, rf := range structFields.reflectFields {
		fieldValue := rv.Field(rf.index)
		if rf.isNullable && fieldValue.IsNil() {
			builder.AddNull()
			continue
		}

		switch rf.kind {
		case reflect.String:
			builder.AddString(fieldValue.String())
			continue
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			builder.AddInt64(fieldValue.Int())
			continue
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			builder.AddInt64(int64(fieldValue.Uint()))
			continue
		case reflect.Float32, reflect.Float64:
			builder.AddFloat64(fieldValue.Float())
			continue
		case reflect.Bool:
			builder.AddBool(fieldValue.Bool())
			continue
		}

		if err := m.marshalValue(builder, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

func (m *Marshaler) marshalValue(va valueAdder, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Bool:
		va.AddBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		va.AddInt64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		va.AddInt64(int64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		va.AddFloat64(rv.Float())
	case reflect.String:
		va.AddString(rv.String())
	case reflect.Struct:
		innerBuilder := m.builderPool.Get().(*Builder)
		innerBuilder.Reset()

		err := m.marshalStruct(innerBuilder, rv)
		if err != nil {
			m.builderPool.Put(innerBuilder)
			return err
		}

		data, err := innerBuilder.Build()
		if err != nil {
			m.builderPool.Put(innerBuilder)
			return err
		}
		va.AddRaw(data)
		m.builderPool.Put(innerBuilder)
	case reflect.Slice:
		count := rv.Len()
		elemType := rv.Type().Elem()
		isNullable := false
		if elemType.Kind() == reflect.Pointer {
			isNullable = true
		}

		rafElemType, err := reflectTypeToRAFType(elemType)
		if err != nil {
			return err
		}

		innerArrayBuilder := m.arrayBuilderPool.Get().(*ArrayBuilder)
		innerArrayBuilder.Reset(rafElemType, count)

		elemKind := elemType.Kind()
		for elemKind == reflect.Pointer {
			elemKind = elemType.Elem().Kind()
		}

		switch elemKind {
		case reflect.String:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}
				innerArrayBuilder.AddString(elemValue.String())
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}
				innerArrayBuilder.AddInt64(elemValue.Int())
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}
				innerArrayBuilder.AddInt64(int64(elemValue.Uint()))
			}
		case reflect.Float32, reflect.Float64:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}
				innerArrayBuilder.AddFloat64(elemValue.Float())
			}
		case reflect.Bool:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}
				innerArrayBuilder.AddBool(elemValue.Bool())
			}
		default:
			for i := range count {
				elemValue := rv.Index(i)
				if isNullable && elemValue.IsNil() {
					innerArrayBuilder.AddNull()
					continue
				}
				for elemValue.Kind() == reflect.Pointer {
					elemValue = elemValue.Elem()
				}

				if err := m.marshalValue(innerArrayBuilder, elemValue); err != nil {
					m.arrayBuilderPool.Put(innerArrayBuilder)
					return err
				}
			}
		}

		innerArrayBuilderData, err := innerArrayBuilder.Build()
		if err != nil {
			m.arrayBuilderPool.Put(innerArrayBuilder)
			return err
		}

		va.AddRaw(innerArrayBuilderData)
		m.arrayBuilderPool.Put(innerArrayBuilder)

	default:
		return fmt.Errorf("raf: unsupported value type %s", rv.Type().String())
	}

	return nil
}

func (m *Marshaler) structFieldsForType(rt reflect.Type) (*structFields, error) {
	if cached, ok := m.structCache.Load(rt); ok {
		return cached.(*structFields), nil
	}

	fields, err := computeStructFields(rt)
	if err != nil {
		return nil, err
	}

	m.structCache.Store(rt, fields)
	return fields, nil
}

func fieldName(f reflect.StructField) (name string, skip bool) {
	tag := f.Tag.Get("raf")
	if !f.IsExported() {
		return "", true
	}
	if tag == "-" {
		return "", true
	}
	name = tag
	if name == "" {
		name = strings.ToLower(f.Name)
	}
	return name, false
}
