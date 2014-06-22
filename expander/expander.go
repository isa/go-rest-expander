package expander

import (
	"fmt"
	"reflect"
	"strings"
)

type Modification struct {
	Children []Modification
	Value string
}

func Expand(object interface{}, expansion, fields string) map[string]interface{} {
	return typeWalker(object)
}

func typeWalker(object interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if object == nil {
		return result
	}

	v := reflect.ValueOf(object)
	switch object.(type) {
	case reflect.Value:
		v = object.(reflect.Value)
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := v.Type().Field(i)

		result[ft.Name] = getValueFrom(f)
	}

	return result
}

func getValueFrom(t reflect.Value) interface{} {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return t.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return t.Uint()
	case reflect.Float32, reflect.Float64:
		return t.Float()
	case reflect.Bool:
		return t.Bool()
	case reflect.String:
		return t.String()
	case reflect.Slice:
		var result []interface{}

		for i := 0; i < t.Len(); i++ {
			result = append(result, getValueFrom(t.Index(i)))
		}

		return result
	case reflect.Map:
		result := make(map[interface{}]interface{})

		for _, v := range t.MapKeys() {
			result[v] = getValueFrom(t.MapIndex(v))
		}

		return result
	case reflect.Struct:
		return typeWalker(t)
	case reflect.Interface:
		fmt.Println("interfaces are not supported...")
	case reflect.Ptr:
		fmt.Println("pointers are not supported...")
	case reflect.Array:
		fmt.Println("arrays are not supported...")
	default:
		fmt.Println("ugh.. unsupported type...")
	}

	return ""
}

func buildModifyTree(expansion string) []Modification {
	var result []Modification
	const comma rune = ','

	if expansion == "*" {
		return result
	}

	expansion = strings.Replace(expansion, " ", "", -1)

	lastCommaIndex := 0
	for i, b := range expansion {
		if b == comma {
			result = append(result, Modification{Value: string(expansion[lastCommaIndex:i])})
			lastCommaIndex = i + 1
		}
	}
	result = append(result, Modification{Value: string(expansion[lastCommaIndex:])})

	return result
}
