package expander

import (
	"fmt"
	"reflect"
)

type Parametric interface {
	GetString(key string) string
}

func Expand(p Parametric, object interface{}) map[string]interface{} {
	_ = p.GetString("expand")
	result := make(map[string]interface{})

	if object == nil {
		return result
	}

	v := reflect.ValueOf(object)

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
	default:
		fmt.Println("unsupported type...")
	}

	return ""
}
