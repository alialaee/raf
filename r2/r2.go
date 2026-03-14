package r2

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/alialaee/raf"
)

var defaultUnmarshaler = NewUnmarshaler()

func Unmarshal(data []byte, v any) error {
	return defaultUnmarshaler.Unmarshal(data, v)
}

type TypeMismatchError struct {
	Key          string
	ExpectedType raf.Type
	ActualType   raf.Type
	Inner        error
}

func (e *TypeMismatchError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("type mismatch for key %s: expected %s, got %s: %v", e.Key, e.ExpectedType, e.ActualType, e.Inner)
	}
	return fmt.Sprintf("type mismatch for key %s: expected %s, got %s", e.Key, e.ExpectedType, e.ActualType)
}

func (e *TypeMismatchError) Unwrap() error {
	return e.Inner
}

func (e *TypeMismatchError) WithWrap(err error) error {
	e.Inner = err
	return e
}

func (e *TypeMismatchError) Is(target error) bool {
	if e.Inner != nil && errors.Is(e.Inner, target) {
		return true
	}

	t, ok := target.(*TypeMismatchError)
	if !ok {
		return false
	}
	return e.Key == t.Key && e.ExpectedType == t.ExpectedType && e.ActualType == t.ActualType
}

func newTypeMismatchError(key string, expected raf.Type, actual raf.Type) *TypeMismatchError {
	return &TypeMismatchError{
		Key:          key,
		ExpectedType: expected,
		ActualType:   actual,
	}
}

type Unmarshaler struct {
	opsCache sync.Map
}

func NewUnmarshaler() *Unmarshaler {
	return &Unmarshaler{}
}

type unmarshalOP struct {
	offset     int
	kind       reflect.Kind
	fieldType  reflect.Type
	targetType reflect.Type
	targetKind reflect.Kind
	rafName    []byte
	nested     []unmarshalOP
}

func (u *Unmarshaler) Unmarshal(data []byte, v any) error {
	valueOf := reflect.ValueOf(v)
	if valueOf.Kind() != reflect.Pointer || valueOf.IsNil() {
		return fmt.Errorf("Unmarshal expects a non-nil pointer")
	}

	typeOf := valueOf.Type().Elem()
	return u.unmarshalWithOps(
		u.loadOPs(typeOf),
		raf.NewBlock(data),
		unsafe.Pointer(valueOf.Pointer()),
	)
}

func (u *Unmarshaler) compileOPs(typ reflect.Type) []unmarshalOP {
	ops := make([]unmarshalOP, 0, typ.NumField())
	for field := range typ.Fields() {
		field := field
		fieldRafName, skip := fieldName(field)
		if skip {
			continue
		}

		targetType := field.Type
		targetKind := field.Type.Kind()
		if targetKind == reflect.Pointer {
			targetType = field.Type.Elem()
			targetKind = targetType.Kind()
		}

		ops = append(ops, unmarshalOP{
			offset:     int(field.Offset),
			kind:       field.Type.Kind(),
			fieldType:  field.Type,
			targetType: targetType,
			targetKind: targetKind,
			rafName:    []byte(fieldRafName),
		})

		if targetKind == reflect.Struct {
			ops[len(ops)-1].nested = u.compileOPs(targetType)
		}
	}

	// Order keys by raf tag to match the order in the RAF block
	sort.SliceStable(ops, func(i, j int) bool {
		return bytes.Compare(ops[i].rafName, ops[j].rafName) < 0
	})

	return ops
}

func typeCompatible(valType raf.Type, targetKind reflect.Kind) bool {
	switch valType {
	case raf.TypeNull:
		return true
	case raf.TypeString:
		return targetKind == reflect.String
	case raf.TypeInt64:
		if targetKind != reflect.Int && targetKind != reflect.Int8 && targetKind != reflect.Int16 && targetKind != reflect.Int32 && targetKind != reflect.Int64 &&
			targetKind != reflect.Uint && targetKind != reflect.Uint8 && targetKind != reflect.Uint16 && targetKind != reflect.Uint32 && targetKind != reflect.Uint64 &&
			targetKind != reflect.Float32 && targetKind != reflect.Float64 {
			return false
		}
		return true
	case raf.TypeFloat64:
		return targetKind == reflect.Float32 || targetKind == reflect.Float64
	case raf.TypeBool:
		return targetKind == reflect.Bool
	case raf.TypeMap:
		return targetKind == reflect.Struct
	case raf.TypeArray:
		return targetKind == reflect.Slice
	default:
		return false
	}
}

func (u *Unmarshaler) unmarshalWithOps(ops []unmarshalOP, data raf.Block, base unsafe.Pointer) error {
	for dataI, opsI := 0, 0; dataI < data.NumPairs() && opsI < len(ops); {
		op := ops[opsI]
		cmp := bytes.Compare(data.KeyAt(dataI), op.rafName)
		if cmp > 0 {
			opsI++
			continue
		}
		if cmp < 0 {
			dataI++
			continue
		}

		fieldPtr := unsafe.Add(base, op.offset)
		val := data.ValueAt(dataI)
		targetKind := op.targetKind

		if op.kind == reflect.Pointer {
			fieldValue := reflect.NewAt(op.fieldType, fieldPtr).Elem()
			if val.IsNull() {
				fieldValue.SetZero()
				opsI++
				dataI++
				continue
			}
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(op.targetType))
			}
			fieldPtr = unsafe.Pointer(fieldValue.Pointer())
		}

		if !typeCompatible(val.Type, targetKind) {
			return newTypeMismatchError(string(op.rafName), val.Type, raf.Type(targetKind))
		}

		switch targetKind {
		case reflect.Int:
			*(*int)(fieldPtr) = int(val.Int64())
		case reflect.Int8:
			*(*int8)(fieldPtr) = int8(val.Int64())
		case reflect.Int16:
			*(*int16)(fieldPtr) = int16(val.Int64())
		case reflect.Int32:
			*(*int32)(fieldPtr) = int32(val.Int64())
		case reflect.Int64:
			*(*int64)(fieldPtr) = val.Int64()
		case reflect.Uint:
			*(*uint)(fieldPtr) = uint(val.Int64())
		case reflect.Uint8:
			*(*uint8)(fieldPtr) = uint8(val.Int64())
		case reflect.Uint16:
			*(*uint16)(fieldPtr) = uint16(val.Int64())
		case reflect.Uint32:
			*(*uint32)(fieldPtr) = uint32(val.Int64())
		case reflect.Uint64:
			*(*uint64)(fieldPtr) = uint64(val.Int64())
		case reflect.Float32:
			*(*float32)(fieldPtr) = float32(val.Float64())
		case reflect.Float64:
			*(*float64)(fieldPtr) = val.Float64()
		case reflect.Bool:
			*(*bool)(fieldPtr) = val.Bool()
		case reflect.String:
			*(*string)(fieldPtr) = val.String()
		case reflect.Struct:
			if err := u.unmarshalWithOps(op.nested, val.Map(), fieldPtr); err != nil {
				return err
			}
		case reflect.Slice:
			fieldValue := reflect.NewAt(op.fieldType, fieldPtr).Elem()
			if err := u.unmarshalValueInto(fieldValue, val); err != nil {
				return newTypeMismatchError(string(op.rafName), val.Type, raf.Type(targetKind)).WithWrap(err)
			}
		default:
			return fmt.Errorf("%w: unsupported target kind %s", newTypeMismatchError(string(op.rafName), val.Type, raf.Type(targetKind)), targetKind)
		}

		opsI++
		dataI++
	}
	return nil
}

func (u *Unmarshaler) unmarshalValueInto(dst reflect.Value, val raf.Value) error {
	if dst.Kind() == reflect.Pointer {
		if val.IsNull() {
			dst.SetZero()
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return u.unmarshalValueInto(dst.Elem(), val)
	}

	if val.IsNull() {
		dst.SetZero()
		return nil
	}

	switch val.Type {
	case raf.TypeString:
		if dst.Kind() != reflect.String {
			return fmt.Errorf("cannot unmarshal string into %s", dst.Type())
		}
		dst.SetString(val.String())
		return nil
	case raf.TypeInt64:
		switch dst.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dst.SetInt(val.Int64())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dst.SetUint(uint64(val.Int64()))
		case reflect.Float32, reflect.Float64:
			dst.SetFloat(float64(val.Int64()))
		default:
			return fmt.Errorf("cannot unmarshal int64 into %s", dst.Type())
		}
		return nil
	case raf.TypeFloat64:
		switch dst.Kind() {
		case reflect.Float32, reflect.Float64:
			dst.SetFloat(val.Float64())
		default:
			return fmt.Errorf("cannot unmarshal float64 into %s", dst.Type())
		}
		return nil
	case raf.TypeBool:
		if dst.Kind() != reflect.Bool {
			return fmt.Errorf("cannot unmarshal bool into %s", dst.Type())
		}
		dst.SetBool(val.Bool())
		return nil
	case raf.TypeMap:
		if dst.Kind() != reflect.Struct {
			return fmt.Errorf("cannot unmarshal map into %s", dst.Type())
		}
		ops := u.loadOPs(dst.Type())
		return u.unmarshalWithOps(ops, val.Map(), unsafe.Pointer(dst.Addr().Pointer()))
	case raf.TypeArray:
		if dst.Kind() != reflect.Slice {
			return fmt.Errorf("cannot unmarshal array into %s", dst.Type())
		}
		arr := val.Array()
		if dst.IsNil() || dst.Cap() < arr.Len() {
			dst.Set(reflect.MakeSlice(dst.Type(), arr.Len(), arr.Len()))
		} else {
			dst.SetLen(arr.Len()) // TODO: consider removing this. Let's always allocate.
		}
		for i := 0; i < arr.Len(); i++ {
			elemVal := raf.Value{Type: arr.ElemType(), Data: arr.At(i)}
			if err := u.unmarshalValueInto(dst.Index(i), elemVal); err != nil {
				return fmt.Errorf("index %d: %w", i, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported value type %s", val.Type)
	}
}

func (u *Unmarshaler) loadOPs(typ reflect.Type) []unmarshalOP {
	ops, ok := u.opsCache.Load(typ)
	if !ok {
		ops = u.compileOPs(typ)
		u.opsCache.Store(typ, ops)
	}
	return ops.([]unmarshalOP)
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
