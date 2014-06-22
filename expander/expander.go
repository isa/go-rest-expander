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

type Modifications []Modification

func Expand(data interface{}, expansion, fields string) map[string]interface{} {
	return typeWalker(data)
}

func typeWalker(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if data == nil {
		return result
	}

	v := reflect.ValueOf(data)
	switch data.(type) {
	case reflect.Value:
		v = data.(reflect.Value)
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

func buildModifyTree(expansion string) ([]Modification, int) {
	var result []Modification
	const comma uint8 = ','
	const openBracket uint8 = '('
	const closeBracket uint8 = ')'

	if expansion == "*" {
		return result, -1
	}

	expansion = strings.Replace(expansion, " ", "", -1)
	indexAfterSeparation := 0
	closeIndex := 0

	for i := 0; i < len(expansion); i++ {
		switch expansion[i] {
			case openBracket:
				modification := Modification{Value: string(expansion[indexAfterSeparation:i])}
				modification.Children, closeIndex = buildModifyTree(expansion[i + 1:])
				result = append(result, modification)
				i = i + closeIndex
				indexAfterSeparation = i + 1
				closeIndex = indexAfterSeparation
			case comma:
				modification := Modification{Value: string(expansion[indexAfterSeparation:i])}
				if modification.Value != "" {
					result = append(result, modification)
				}
				indexAfterSeparation = i + 1
			case closeBracket:
				modification := Modification{Value: string(expansion[indexAfterSeparation:i])}
				if modification.Value != "" {
					result = append(result, modification)
				}

				return result, i + 1
		}
	}

	if indexAfterSeparation > closeIndex {
		result = append(result, Modification{Value: string(expansion[indexAfterSeparation:])})
	}

	return result, -1
}

func filterOut(data interface{}, modifications Modifications) map[string]interface{} {
	result := make(map[string]interface{})

	return result
}
