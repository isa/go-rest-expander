package expander

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const (
	REF_KEY  = "ref"
	REL_KEY  = "rel"
	VERB_KEY = "verb"
)

type Filter struct {
	Children []Filter
	Value    string
}

type Filters []Filter

func (m Filters) Contains(v string) bool {
	for _, m := range m {
		if v == m.Value {
			return true
		}
	}

	return false
}

func (m Filters) IsEmpty() bool {
	return len(m) == 0
}

func (m Filters) Get(v string) Filter {
	var result Filter

	if m.IsEmpty() {
		return result
	}

	for _, m := range m {
		if v == m.Value {
			return m
		}
	}

	return result
}

func Expand(data interface{}, expansion, fields string) map[string]interface{} {
	fieldFilter, _ := buildFilterTree(fields)
	expansionFilter, _ := buildFilterTree(expansion)

	expanded := walkByExpansion(data, expansionFilter)
	return walkByFilter(expanded, fieldFilter)
}

func walkByExpansion(data interface{}, filters Filters) map[string]interface{} {
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

		val := getValue(f, filters.Get(ft.Name).Children)

		key := ft.Name
		if ft.Tag.Get("json") != "" {
			key = ft.Tag.Get("json")
		}
		result[key] = val

		if isReference(f) {
			uri := getReferenceURI(f)
			resource, ok := getResourceFrom(uri)
			if ok {
				result[key] = resource
			}
		}
	}

	return result
}

func walkByFilter(data map[string]interface{}, filters Filters) map[string]interface{} {
	result := make(map[string]interface{})

	if data == nil {
		return result
	}

	for k, v := range data {
		if filters.IsEmpty() || filters.Contains(k) {
			result[k] = v
			ft := reflect.ValueOf(v)

			if isReference(ft) {
				uri := getReferenceURI(ft)
				resource, ok := getResourceFrom(uri)
				if ok {
					result[k] = resource
				}
			} else {
				switch ft.Type().Kind() {
				case reflect.Map:
					result[k] = walkByFilter(v.(map[string]interface{}), filters.Get(k).Children)
				}
			}
		}
	}

	return result
}

func getValue(t reflect.Value, filters Filters) interface{} {
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
			current := t.Index(i)

			if isReference(current) {
				uri := getReferenceURI(current)
				resource, ok := getResourceFrom(uri)

				if ok {
					result = append(result, resource)
				}
			} else {
				result = append(result, getValue(current, filters))
			}
		}

		return result
	case reflect.Map:
		result := make(map[string]interface{})

		for _, v := range t.MapKeys() {
			result[v.Interface().(string)] = getValue(t.MapIndex(v), filters)
		}

		return result
	case reflect.Struct:
		return walkByExpansion(t, filters)
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

func getResourceFrom(u string) (map[string]interface{}, bool) {
	ok := false
	uri, err := url.ParseRequestURI(u)
	var m map[string]interface{}

	if err == nil {
		content := getContentFrom(uri)
		_ = json.Unmarshal([]byte(content), &m)
		ok = true

		if hasReference(m) {
			m = expandChildren(m)
		}
	}

	return m, ok
}

func expandChildren(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, v := range m {
		ft := reflect.TypeOf(v)
		result[key] = v

		if ft.Kind() == reflect.Map {
			child := v.(map[string]interface{})
			uri, found := child[REF_KEY]

			if found {
				resource, ok := getResourceFrom(uri.(string))
				if ok {
					result[key] = resource
				}
			}
		}
	}

	return result
}

func isRefKey(ft reflect.StructField) bool {
	return ft.Name == REF_KEY || ft.Tag.Get("json") == REF_KEY
}

func isReference(t reflect.Value) bool {
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			ft := t.Type().Field(i)

			if isRefKey(ft) {
				return true
			}
		}
	}

	return false
}

func hasReference(m map[string]interface{}) bool {
	for _, v := range m {
		ft := reflect.TypeOf(v)

		// what about lists?
		if ft.Kind() == reflect.Map {
			child := v.(map[string]interface{})
			_, ok := child[REF_KEY]

			if ok {
				return true
			}

			return hasReference(child)
		}
	}

	return false
}

func getReferenceURI(t reflect.Value) string {
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			ft := t.Type().Field(i)

			if isRefKey(ft) {
				return t.Field(i).String()
			}
		}
	}

	return ""
}

var getContentFrom = func(uri *url.URL) string {
	response, err := http.Get(uri.String())

	if err != nil {
		fmt.Println(err)
		return ""
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(contents)
}

func buildFilterTree(statement string) ([]Filter, int) {
	var result []Filter
	const comma uint8 = ','
	const openBracket uint8 = '('
	const closeBracket uint8 = ')'

	if statement == "*" {
		return result, -1
	}

	statement = strings.Replace(statement, " ", "", -1)
	indexAfterSeparation := 0
	closeIndex := 0

	for i := 0; i < len(statement); i++ {
		switch statement[i] {
		case openBracket:
			filter := Filter{Value: string(statement[indexAfterSeparation:i])}
			filter.Children, closeIndex = buildFilterTree(statement[i+1:])
			result = append(result, filter)
			i = i + closeIndex
			indexAfterSeparation = i + 1
			closeIndex = indexAfterSeparation
		case comma:
			filter := Filter{Value: string(statement[indexAfterSeparation:i])}
			if filter.Value != "" {
				result = append(result, filter)
			}
			indexAfterSeparation = i + 1
		case closeBracket:
			filter := Filter{Value: string(statement[indexAfterSeparation:i])}
			if filter.Value != "" {
				result = append(result, filter)
			}

			return result, i + 1
		}
	}

	if indexAfterSeparation > closeIndex {
		result = append(result, Filter{Value: string(statement[indexAfterSeparation:])})
	}

	return result, -1
}
