package raf

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"unsafe"
)

var (
	ErrInvalidRAFData = errors.New("invalid RAF data")
)

type ErrTypeMismatch struct {
	Key          string
	ExpectedType Type
	ActualType   reflect.Kind
	Inner        error
}

func (e *ErrTypeMismatch) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("type mismatch for key %s: expected %s, got %s: %v", e.Key, e.ExpectedType, e.ActualType, e.Inner)
	}
	return fmt.Sprintf("type mismatch for key %s: expected %s, got %s", e.Key, e.ExpectedType, e.ActualType)
}

func (e *ErrTypeMismatch) Unwrap() error {
	return e.Inner
}

func (e *ErrTypeMismatch) WithWrap(err error) error {
	e.Inner = err
	return e
}

func (e *ErrTypeMismatch) Is(target error) bool {
	if e.Inner != nil && errors.Is(e.Inner, target) {
		return true
	}

	t, ok := target.(*ErrTypeMismatch)
	if !ok {
		return false
	}
	return e.Key == t.Key && e.ExpectedType == t.ExpectedType && e.ActualType == t.ActualType
}

func newErrTypeMismatch(key string, expected Type, actual reflect.Kind) *ErrTypeMismatch {
	return &ErrTypeMismatch{
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

	// If the field is a slice
	elemKind reflect.Kind
	elemSize uintptr

	// If the field is a struct or a slice of structs, nested contains
	// the ops for the nested struct(s) and nil for recursive types,
	// resolved lazily at unmarshal time via loadOPs.
	nested []unmarshalOP

	rafName []byte
}

func (u *Unmarshaler) Unmarshal(data []byte, v any) error {
	block := NewBlock(data)
	if !block.Valid() {
		return ErrInvalidRAFData
	}

	if m, ok := v.(*map[string]any); ok {
		*m = unmarshalBlockToMap(block)
		return nil
	}

	valueOf := reflect.ValueOf(v)
	if valueOf.Kind() != reflect.Pointer || valueOf.IsNil() {
		return fmt.Errorf("Unmarshal expects a non-nil pointer")
	}

	typeOf := valueOf.Type().Elem()
	return u.unmarshal(
		u.loadOPs(typeOf),
		block,
		unsafe.Pointer(valueOf.Pointer()),
	)
}

func unmarshalBlockToMap(block Block) map[string]any {
	n := block.NumPairs()
	m := make(map[string]any, n)
	for i := range n {
		m[string(block.KeyAt(i))] = unmarshalValueToAny(block.ValueAt(i))
	}
	return m
}

func unmarshalValueToAny(val Value) any {
	switch val.Type {
	case TypeString:
		return val.String()
	case TypeInt64:
		return val.Int64()
	case TypeFloat64:
		return val.Float64()
	case TypeBool:
		return val.Bool()
	case TypeMap:
		return unmarshalBlockToMap(val.Block())
	case TypeArray:
		arr := val.Array()
		n := arr.Len()
		switch arr.ElemType() {
		case TypeString:
			s := make([]string, n)
			for i := range n {
				s[i] = string(arr.At(i))
			}
			return s
		case TypeInt64:
			s := make([]int64, n)
			for i := range n {
				s[i] = arr.AtInt64(i)
			}
			return s
		case TypeFloat64:
			s := make([]float64, n)
			for i := range n {
				s[i] = arr.AtFloat64(i)
			}
			return s
		case TypeBool:
			s := make([]bool, n)
			for i := range n {
				s[i] = arr.AtBool(i)
			}
			return s
		case TypeMap:
			s := make([]map[string]any, n)
			for i := range n {
				s[i] = unmarshalBlockToMap(NewBlock(arr.At(i)))
			}
			return s
		default:
			s := make([]any, n)
			for i := range n {
				s[i] = unmarshalValueToAny(Value{Type: arr.ElemType(), Data: arr.At(i)})
			}
			return s
		}
	default:
		return nil
	}
}

func (u *Unmarshaler) compileOPs(typ reflect.Type) []unmarshalOP {
	return u.compileOPsWithSeen(typ, nil)
}

func (u *Unmarshaler) compileOPsWithSeen(typ reflect.Type, seen map[reflect.Type]bool) []unmarshalOP {
	if seen == nil {
		seen = make(map[reflect.Type]bool)
	}
	seen[typ] = true
	defer delete(seen, typ)

	ops := make([]unmarshalOP, 0, typ.NumField())
	for field := range typ.Fields() {
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

		switch targetKind {
		case reflect.Struct:
			if !seen[targetType] {
				ops[len(ops)-1].nested = u.compileOPsWithSeen(targetType, seen)
			}
		case reflect.Slice:
			elemType := targetType.Elem()
			ops[len(ops)-1].elemKind = elemType.Kind()
			ops[len(ops)-1].elemSize = elemType.Size()
			if elemType.Kind() == reflect.Struct {
				if !seen[elemType] {
					ops[len(ops)-1].nested = u.compileOPsWithSeen(elemType, seen)
				}
			}
		}
	}

	// Order keys by raf tag to match the order in the RAF block
	sort.SliceStable(ops, func(i, j int) bool {
		return bytes.Compare(ops[i].rafName, ops[j].rafName) < 0
	})

	return ops
}

func typeCompatible(valType Type, targetKind reflect.Kind) bool {
	switch valType {
	case TypeString:
		return targetKind == reflect.String
	case TypeInt64:
		return targetKind >= reflect.Int && targetKind <= reflect.Uint64
	case TypeFloat64:
		return targetKind == reflect.Float32 || targetKind == reflect.Float64
	case TypeBool:
		return targetKind == reflect.Bool
	case TypeMap:
		return targetKind == reflect.Struct || targetKind == reflect.Map
	case TypeArray:
		return targetKind == reflect.Slice
	default:
		return false
	}
}

func (u *Unmarshaler) unmarshal(ops []unmarshalOP, data Block, base unsafe.Pointer) error {
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

		if len(val.Data) == 0 {
			if op.kind == reflect.Pointer {
				opsI++
				dataI++
				continue
			}
		}

		if op.kind == reflect.Pointer {
			fieldValue := reflect.NewAt(op.fieldType, fieldPtr).Elem()
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(op.targetType))
			}
			fieldPtr = unsafe.Pointer(fieldValue.Pointer())
		}

		if !typeCompatible(val.Type, targetKind) {
			return newErrTypeMismatch(string(op.rafName), val.Type, targetKind)
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
			if len(val.Data) == 0 {
				break
			}
			if err := u.unmarshal(op.nested, val.Block(), fieldPtr); err != nil {
				return err
			}
		case reflect.Map:
			if len(val.Data) == 0 {
				break
			}
			m := unmarshalBlockToMap(val.Block())
			reflect.NewAt(op.targetType, fieldPtr).Elem().Set(reflect.ValueOf(m))
		case reflect.Slice:
			if len(val.Data) == 0 {
				break
			}
			arr := val.Array()
			fieldValue := reflect.NewAt(op.targetType, fieldPtr).Elem()
			arrLen := arr.Len()
			if fieldValue.IsNil() || fieldValue.Cap() < arrLen {
				fieldValue.Set(reflect.MakeSlice(op.targetType, arrLen, arrLen))
			} else {
				fieldValue.SetLen(arrLen)
			}
			if arrLen == 0 {
				break
			}
			sliceBase := unsafe.Pointer(fieldValue.Pointer())
			for i := range arrLen {
				elemPtr := unsafe.Add(sliceBase, uintptr(i)*op.elemSize)
				switch op.elemKind {
				case reflect.Int:
					*(*int)(elemPtr) = int(arr.AtInt64(i))
				case reflect.Int8:
					*(*int8)(elemPtr) = int8(arr.AtInt64(i))
				case reflect.Int16:
					*(*int16)(elemPtr) = int16(arr.AtInt64(i))
				case reflect.Int32:
					*(*int32)(elemPtr) = int32(arr.AtInt64(i))
				case reflect.Int64:
					*(*int64)(elemPtr) = arr.AtInt64(i)
				case reflect.Uint:
					*(*uint)(elemPtr) = uint(arr.AtInt64(i))
				case reflect.Uint8:
					*(*uint8)(elemPtr) = uint8(arr.AtInt64(i))
				case reflect.Uint16:
					*(*uint16)(elemPtr) = uint16(arr.AtInt64(i))
				case reflect.Uint32:
					*(*uint32)(elemPtr) = uint32(arr.AtInt64(i))
				case reflect.Uint64:
					*(*uint64)(elemPtr) = uint64(arr.AtInt64(i))
				case reflect.Float32:
					*(*float32)(elemPtr) = float32(arr.AtFloat64(i))
				case reflect.Float64:
					*(*float64)(elemPtr) = arr.AtFloat64(i)
				case reflect.Bool:
					*(*bool)(elemPtr) = arr.AtBool(i)
				case reflect.String:
					*(*string)(elemPtr) = string(arr.At(i))
				case reflect.Struct:
					if err := u.unmarshal(op.nested, NewBlock(arr.At(i)), elemPtr); err != nil {
						return err
					}
				case reflect.Map:
					m := unmarshalBlockToMap(NewBlock(arr.At(i)))
					reflect.NewAt(op.targetType.Elem(), elemPtr).Elem().Set(reflect.ValueOf(m))
				default:
					return fmt.Errorf("unsupported slice element kind: %s", op.elemKind)
				}
			}
		default:
			return fmt.Errorf("%w: unsupported target kind %s", newErrTypeMismatch(string(op.rafName), val.Type, targetKind), targetKind)
		}

		opsI++
		dataI++
	}
	return nil
}

func (u *Unmarshaler) loadOPs(typ reflect.Type) []unmarshalOP {
	ops, ok := u.opsCache.Load(typ)
	if !ok {
		ops = u.compileOPs(typ)
		u.opsCache.Store(typ, ops)
		// Resolve any nil nested ops left by recursive type cycles
		u.resolveLazyOPs(ops.([]unmarshalOP))
	}
	return ops.([]unmarshalOP)
}

// resolveLazyOPs fills in nil nested fields that were deferred during
// compilation due to recursive types.
func (u *Unmarshaler) resolveLazyOPs(ops []unmarshalOP) {
	for i := range ops {
		switch ops[i].targetKind {
		case reflect.Struct:
			if ops[i].nested == nil {
				ops[i].nested = u.loadOPs(ops[i].targetType)
			}
		case reflect.Slice:
			if ops[i].elemKind == reflect.Struct && ops[i].nested == nil {
				ops[i].nested = u.loadOPs(ops[i].targetType.Elem())
			}
		}
	}
}
