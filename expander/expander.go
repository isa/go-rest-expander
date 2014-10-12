package expander

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"net"
	"errors"

	"github.com/golang/groupcache/lru"
)

const (
	REF_KEY        = "ref"
	REL_KEY        = "rel"
	VERB_KEY       = "verb"
	COLLECTION_KEY = "Collection"
)

type Configuration struct {
	UsingCache           bool
	UsingMongo           bool
	IdURIs            map[string]string
	CacheExpInSeconds    int64
	ConnectionTimeoutInS int
}

var ExpanderConfig Configuration = Configuration{
	UsingMongo:        false,
	UsingCache:        false,
	CacheExpInSeconds: 86400, // = 24 hours
	ConnectionTimeoutInS: 2,
}

var Cache *lru.Cache = lru.New(250)
var CacheMutex = sync.Mutex{}
var client http.Client
var timeout = time.Duration(2 * time.Second)
var httpClientIsInitialized = false
var initializingHttpClient = false
var initializerMutex = sync.Mutex{}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func Init() {
	client = http.Client{}

	client.Timeout = time.Duration(ExpanderConfig.ConnectionTimeoutInS)*time.Second

	httpClientIsInitialized = true
}

type CacheEntry struct {
	Timestamp int64
	Data      string
}

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

func resolveFilters(expansion, fields string) (expansionFilter Filters, fieldFilter Filters, recursiveExpansion bool, err error) {
	if !validateFilterFormat(expansion) {
		err = errors.New("expansionFilter for filtering was not correct")
		return
	}
	if !validateFilterFormat(fields) {
		err = errors.New("fieldFilter for filtering was not correct")
		return
	}

	fieldFilter, _ = buildFilterTree(fields)

	if expansion != "*" {
		expansionFilter, _ = buildFilterTree(expansion)
	} else if fields != "*" && fields != "" {
		expansionFilter, _ = buildFilterTree(fields)
	} else {
		recursiveExpansion = true
	}
	return
}

//TODO: TagFields & BSONFields
func Expand(data interface{}, expansion, fields string) map[string]interface{} {
	if ExpanderConfig.UsingMongo && len(ExpanderConfig.IdURIs) == 0 {
		fmt.Println("Warning: Cannot use mongo flag without proper IdURIs given!")
	}
	if ExpanderConfig.UsingCache && ExpanderConfig.CacheExpInSeconds == 0 {
		fmt.Println("Warning: Cannot use Cache with expiration 0, cache will be useless!")
	}

	expansionFilter, fieldFilter, recursiveExpansion, err := resolveFilters(expansion, fields)
	if err != nil {
		expansionFilter = Filters{}
		fieldFilter = Filters{}
		fmt.Printf("Warning: Filter was not correct, expansionFilter: '%v' fieldFilter: '%v', error: %v \n", expansion, fields, err)
	}

	expanded := *walkByExpansion(data, expansionFilter, recursiveExpansion)

	filtered := walkByFilter(expanded, fieldFilter)

	return filtered
}

func ExpandArray(data interface{}, expansion, fields string) []interface{} {
	if ExpanderConfig.UsingMongo && len(ExpanderConfig.IdURIs) == 0 {
		fmt.Println("Warning: Cannot use mongo flag without proper IdURIs given!")
	}
	if ExpanderConfig.UsingCache && ExpanderConfig.CacheExpInSeconds == 0 {
		fmt.Println("Warning: Cannot use Cache with expiration 0, cache will be useless!")
	}

	expansionFilter, fieldFilter, recursiveExpansion, err := resolveFilters(expansion, fields)
	if err != nil {
		expansionFilter = Filters{}
		fieldFilter = Filters{}
		fmt.Printf("Warning: Filter was not correct, expansionFilter: '%v' fieldFilter: '%v', error: %v \n", expansionFilter, fieldFilter, err)
	}

	var result []interface{}

	if data == nil {
		return result
	}

	v := reflect.ValueOf(data)
	switch data.(type) {
	case reflect.Value:
		v = data.(reflect.Value)
	}

	if v.Kind() != reflect.Slice {
		return result
	}

	v = v.Slice(0, v.Len())
	for i := 0; i < v.Len(); i++ {
		arrayItem := *walkByExpansion(v.Index(i), expansionFilter, recursiveExpansion)
		arrayItem = walkByFilter(arrayItem, fieldFilter)
		result = append(result, arrayItem)
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
			ft := reflect.ValueOf(v)

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

func walkByExpansion(data interface{}, filters Filters, recursive bool) *map[string]interface{} {
	result := make(map[string]interface{})

	if data == nil {
		return &result
	}

	v := reflect.ValueOf(data)
	switch data.(type) {
	case reflect.Value:
		v = data.(reflect.Value)
	}
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	//	var resultWriteMutex = sync.Mutex{}
	var writeToResult = func(key string, value interface{}) {
		//resultWriteMutex.Lock()
		result[key] = value
		//resultWriteMutex.Unlock()
	}

	// check if root is db ref
	if isMongoDBRef(v) && recursive {
		uri := buildReferenceURI(v)
		key := v.Type().Field(1).Name
		placeholder := make(map[string]interface{})
		resource, _ := getResourceFrom(uri, filters.Get(key).Children, recursive)
		for k, v := range resource {
			placeholder[k] = v
		}
		return &placeholder
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
					writeToResult(key, resource)
				}else {
					writeToResult(key, f.Interface())
				}
			} else {
				writeToResult(key, f.Interface())
			}
		} else {
			val := getValue(f, filters, options)
			writeToResult(key, val)
			switch val.(type) {
			case string:
				unquoted, err := strconv.Unquote(val.(string))
				if err == nil {
					writeToResult(key, unquoted)
				}
			}

			if isReference(f) {
				if filters.Contains(key) || recursive {
					uri := getReferenceURI(f)
					resource, ok := getResourceFrom(uri, filters.Get(key).Children, recursive)
					if ok {
						writeToResult(key, resource)
					}
				}
			}
		}

	}

	return &result
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
		var result = []interface{}{}

		for i := 0; i < t.Len(); i++ {
			current := t.Index(i)

			if filters.Contains(parentKey) || recursive {
				if isReference(current) {
					uri := getReferenceURI(current)

					//TODO: this fails in case the resource cannot be resolved, because current is DBRef not map[string]interface{}
					result = append(result, current.Interface())
					resource, ok := getResourceFrom(uri, filters.Get(parentKey).Children, recursive)
					if ok {
						result[i] = resource
					}
				} else if isMongoDBRef(current) {
					uri := buildReferenceURI(current)

					//TODO: this fails in case the resource cannot be resolved, because current is DBRef not map[string]interface{}
					result = append(result, current.Interface())
					resource, ok := getResourceFrom(uri, filters.Get(parentKey).Children, recursive)
					if ok {
						result[i] = resource
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
		val, ok := t.Interface().(json.Marshaler)
		if ok {
			bytes, err := val.(json.Marshaler).MarshalJSON()
			if err != nil {
				fmt.Println(err)
			}

			return string(bytes)
		}

		return *walkByExpansion(t, filters, recursive)
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
		err := json.Unmarshal([]byte(content), &m)
		if err != nil {
			return m, false
		}
		ok = true
		if hasReference(m) {
			return *expandChildren(m, filters, recursive), ok
		}
	}

	return m, ok
}

func expandChildren(m map[string]interface{}, filters Filters, recursive bool) *map[string]interface{} {
	result := make(map[string]interface{})

	for key, v := range m {
		ft := reflect.TypeOf(v)
		result[key] = v
		if v == nil {
			continue
		}
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

	return &result
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
					uri = base+"/"+objectId.Hex()
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

var makeGetCall = func(uri *url.URL) string {
	if !httpClientIsInitialized {
		initializerMutex.Lock()
		if !initializingHttpClient {
			initializingHttpClient = true
			Init()
		}
		initializerMutex.Unlock()
	}

	response, err := client.Get(uri.String())
	if err != nil {
		fmt.Println(err)
		return ""
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Println("Error while reading content of response body. It was: ", err)
	}

	return string(contents)
}

var makeGetCallAndAddToCache = func(uri *url.URL) string {
	valueToReturn := makeGetCall(uri)

	var responseMap map[string]interface{}
	err := json.Unmarshal([]byte(valueToReturn), &responseMap)

	_, ok := responseMap["error"]
	if err != nil || ok {
		return ""
	}

	cacheEntry := CacheEntry{
		Timestamp: time.Now().Unix(),
		Data:      valueToReturn,
	}
	CacheMutex.Lock()
	Cache.Add(uri.String(), cacheEntry)
	CacheMutex.Unlock()
	return valueToReturn
}


var getContentFrom = func(uri *url.URL) string {
	if ExpanderConfig.UsingCache {
		CacheMutex.Lock()
		value, ok := Cache.Get(uri.String())
		CacheMutex.Unlock()
		if !ok {
			//no data found in cache
			return makeGetCallAndAddToCache(uri)
		}

		cachedData := value.(CacheEntry)
		nowInMillis := time.Now().Unix()

		if nowInMillis-cachedData.Timestamp > ExpanderConfig.CacheExpInSeconds {
			//data older then Expiration
			CacheMutex.Lock()
			Cache.Remove(uri.String())
			CacheMutex.Unlock()
			return makeGetCallAndAddToCache(uri)
		}

		return cachedData.Data
	}

	return makeGetCall(uri)
}

func validateFilterFormat(filter string) bool {
	runes := []rune(filter)

	var bracketCounter = 0

	for i := range runes {
		if runes[i] == '(' {
			bracketCounter++
		}else if runes[i] == ')' {
			bracketCounter--
			if bracketCounter < 0 {
				return false
			}
		}
	}
	return bracketCounter == 0

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
			i = i+closeIndex
			indexAfterSeparation = i+1
			closeIndex = indexAfterSeparation
		case comma:
			filter := Filter{Value: string(statement[indexAfterSeparation:i])}
			if filter.Value != "" {
				result = append(result, filter)
			}
			indexAfterSeparation = i+1
		case closeBracket:
			filter := Filter{Value: string(statement[indexAfterSeparation:i])}
			if filter.Value != "" {
				result = append(result, filter)
			}

			return result, i+1
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
