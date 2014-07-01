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
	REF_KEY        = "ref"
	REL_KEY        = "rel"
	VERB_KEY       = "verb"
	COLLECTION_KEY = "Collection"
)

type Configuration struct {
	UsingMongo bool
	IdURIs     map[string]string
}

var ExpanderConfig Configuration = Configuration{UsingMongo: false}

type DBRef struct {
	Collection string
	Id         interface{}
	Database   string
}

type ObjectId interface {
	Hex() string
}

type Filter struct {
	Children Filters
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

//TODO: TagFields & BSONFields
func Expand(data interface{}, expansion, fields string) map[string]interface{} {
	if ExpanderConfig.UsingMongo && len(ExpanderConfig.IdURIs) == 0 {
		fmt.Println("Warning: Cannot use mongo flag without proper IdURIs given!")
	}

	var recursiveExpansion bool
	fieldFilter, _ := buildFilterTree(fields)
	expansionFilter, _ := buildFilterTree(expansion)

	if expansion == "*" {
		recursiveExpansion = true
	}

	expanded := walkByExpansion(data, expansionFilter, recursiveExpansion)
	filtered := walkByFilter(expanded, fieldFilter)

	return filtered
}

func walkByFilter(data map[string]interface{}, filters Filters) map[string]interface{} {
	result := make(map[string]interface{})

	if data == nil {
		return result
	}

	for k, v := range data {
		if filters.IsEmpty() || filters.Contains(k) {
			ft := reflect.ValueOf(v)
			// if ft.Kind() == reflect.Ptr {
			// 	ft = ft.Elem()
			// 	v = ft.Interface()
			// }

			result[k] = v
			subFilters := filters.Get(k).Children

			if v == nil {
				continue
			}

			switch ft.Type().Kind() {
			case reflect.Map:
				result[k] = walkByFilter(v.(map[string]interface{}), subFilters)
			case reflect.Slice:
				if ft.Len() == 0 {
					continue
				}

				switch ft.Index(0).Kind() {
				case reflect.Map:
					children := make([]map[string]interface{}, 0)
					for _, child := range v.([]map[string]interface{}) {
						item := walkByFilter(child, subFilters)
						children = append(children, item)
					}
					result[k] = children
				default:
					children := make([]interface{}, 0)
					for _, child := range v.([]interface{}) {
						cft := reflect.TypeOf(child)

						if cft.Kind() == reflect.Map {
							item := walkByFilter(child.(map[string]interface{}), subFilters)
							children = append(children, item)
						} else {
							children = append(children, child)
						}
					}
					result[k] = children
				}
			}
		}
	}

	return result
}

func walkByExpansion(data interface{}, filters Filters, recursive bool) map[string]interface{} {
	result := make(map[string]interface{})

	if data == nil {
		return result
	}

	v := reflect.ValueOf(data)
	switch data.(type) {
	case reflect.Value:
		v = data.(reflect.Value)
	}
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := v.Type().Field(i)

		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}

		key := ft.Name
		tag := ft.Tag.Get("json")
		if tag != "" {
			key = strings.Split(tag, ",")[0]
		}

		options := func() (bool, string) {
			return recursive, key
		}

		if isMongoDBRef(f) {
			if filters.Contains(key) || recursive {
				uri := buildReferenceURI(f)
				resource, ok := getResourceFrom(uri, filters.Get(key).Children, recursive)

				if ok && len(resource) > 0 {
					result[key] = resource
				} else {
					result[key] = f.Interface()
				}
			} else {
				result[key] = f.Interface()
			}
		} else {
			val := getValue(f, filters, options)
			result[key] = val

			if isReference(f) {
				if filters.Contains(key) || recursive {
					uri := getReferenceURI(f)
					resource, ok := getResourceFrom(uri, filters.Get(key).Children, recursive)

					if ok {
						result[key] = resource
					}
				}
			}
		}

	}

	return result
}

func getValue(t reflect.Value, filters Filters, options func() (bool, string)) interface{} {
	recursive, parentKey := options()

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

			if filters.Contains(parentKey) || recursive {
				if isReference(current) {
					uri := getReferenceURI(current)
					resource, ok := getResourceFrom(uri, filters.Get(parentKey).Children, recursive)

					if ok {
						result = append(result, resource)
					} else {
						result = append(result, current.Interface())
					}
				} else if isMongoDBRef(current) {
					uri := buildReferenceURI(current)
					resource, ok := getResourceFrom(uri, filters.Get(parentKey).Children, recursive)

					if ok {
						result = append(result, resource)
					} else {
						result = append(result, current.Interface())
					}
				} else {
					result = append(result, getValue(current, filters.Get(parentKey).Children, options))
				}
			} else {
				result = append(result, getValue(current, filters.Get(parentKey).Children, options))
			}
		}

		return result
	case reflect.Map:
		result := make(map[string]interface{})

		for _, v := range t.MapKeys() {
			key := v.Interface().(string)
			result[key] = getValue(t.MapIndex(v), filters.Get(key).Children, options)
		}

		return result
	case reflect.Struct:
		val, ok := t.Interface().(fmt.Stringer)
		if ok {
			return val.String()
		}

		return walkByExpansion(t, filters, recursive)
	default:
		return t.Interface()
	}

	return ""
}

func getResourceFrom(u string, filters Filters, recursive bool) (map[string]interface{}, bool) {
	ok := false
	uri, err := url.ParseRequestURI(u)
	var m map[string]interface{}

	if err == nil {
		content := getContentFrom(uri)
		_ = json.Unmarshal([]byte(content), &m)
		ok = true

		if hasReference(m) {
			return expandChildren(m, filters, recursive), ok
		}
	}

	return m, ok
}

func expandChildren(m map[string]interface{}, filters Filters, recursive bool) map[string]interface{} {
	result := make(map[string]interface{})

	for key, v := range m {
		ft := reflect.TypeOf(v)
		result[key] = v

		if ft.Kind() == reflect.Map && (recursive || filters.Contains(key)) {
			child := v.(map[string]interface{})
			uri, found := child[REF_KEY]

			if found {
				resource, ok := getResourceFrom(uri.(string), filters, recursive)

				if ok {
					result[key] = resource
				}
			}
		}
	}

	return result
}

func buildReferenceURI(t reflect.Value) string {
	var uri string

	if t.Kind() == reflect.Struct {
		collection := ""
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			ft := t.Type().Field(i)

			if ft.Name == COLLECTION_KEY {
				collection = f.String()
			} else {
				objectId, ok := f.Interface().(ObjectId)
				if ok {
					base := ExpanderConfig.IdURIs[collection]
					uri = base + "/" + objectId.Hex()
				}
			}
		}
	}

	return uri
}

func isMongoDBRef(t reflect.Value) bool {
	mongoEnabled := ExpanderConfig.UsingMongo && len(ExpanderConfig.IdURIs) > 0

	if !mongoEnabled {
		return false
	}

	if t.Kind() == reflect.Struct {
		if t.NumField() != 3 {
			return false
		}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.CanInterface() {
				_, ok := f.Interface().(ObjectId)
				if ok {
					return true
				}
			}
		}
	}

	return false
}

func isRefKey(ft reflect.StructField) bool {
	tag := strings.Split(ft.Tag.Get("json"), ",")[0]
	return ft.Name == REF_KEY || tag == REF_KEY
}

func isReference(t reflect.Value) bool {
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			ft := t.Type().Field(i)

			if isRefKey(ft) && t.NumField() > 1 { // at least relation & ref should be given
				return true
			}
		}
	}

	return false
}

func hasReference(m map[string]interface{}) bool {
	for _, v := range m {
		ft := reflect.TypeOf(v)

		if ft != nil && ft.Kind() == reflect.Map {
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
	if len(statement) == 0 {
		return result, -1
	}

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

	if indexAfterSeparation == 0 {
		result = append(result, Filter{Value: statement})
	}

	return result, -1
}
