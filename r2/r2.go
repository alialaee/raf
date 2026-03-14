package r2

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
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
	return fmt.Sprintf("type mismatch for key %s: expected %d, got %d", e.Key, e.ExpectedType, e.ActualType)
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

	ops, ok := u.opsCache.Load(typeOf)
	if !ok {
		ops = u.compileOPs(typeOf)
		u.opsCache.Store(typeOf, ops)
	}

	return u.unmarshalWithOps(
		ops.([]unmarshalOP),
		raf.NewBlock(data),
		unsafe.Pointer(valueOf.Pointer()),
	)
}

func (u *Unmarshaler) compileOPs(typ reflect.Type) []unmarshalOP {
	ops := make([]unmarshalOP, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		targetType := field.Type
		targetKind := field.Type.Kind()
		if targetKind == reflect.Pointer {
			targetType = field.Type.Elem()
			targetKind = targetType.Kind()
		}

		ops[i] = unmarshalOP{
			offset:     int(field.Offset),
			kind:       field.Type.Kind(),
			fieldType:  field.Type,
			targetType: targetType,
			targetKind: targetKind,
			rafName:    []byte(field.Tag.Get("raf")),
		}

		if targetKind == reflect.Struct {
			ops[i].nested = u.compileOPs(targetType)
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
		return targetKind >= reflect.Int && targetKind <= reflect.Uint64
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
	opsI := 0
	for i := 0; i < data.NumPairs() && opsI < len(ops); i++ {
		op := ops[opsI]
		if !bytes.Equal(data.KeyAt(i), op.rafName) {
			continue
		}

		fieldPtr := unsafe.Add(base, op.offset)
		val := data.ValueAt(i)
		targetKind := op.targetKind

		if op.kind == reflect.Pointer {
			fieldValue := reflect.NewAt(op.fieldType, fieldPtr).Elem()
			if val.IsNull() {
				fieldValue.SetZero()
				opsI++
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
		ops, ok := u.opsCache.Load(dst.Type())
		if !ok {
			ops = u.compileOPs(dst.Type())
			u.opsCache.Store(dst.Type(), ops)
		}
		return u.unmarshalWithOps(ops.([]unmarshalOP), val.Map(), unsafe.Pointer(dst.Addr().Pointer()))
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
		return fmt.Errorf("unsupported value type %d", val.Type)
	}
}
