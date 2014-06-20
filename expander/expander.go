package expander

import (
	"fmt"
	"reflect"
	"strconv"
)

type Parametric interface {
	GetString(key string) string
}

func Expand(p Parametric, object interface{}) map[string]interface{} {
	_ = p.GetString("expand")
	result := make(map[string]interface{})

	klazz := reflect.TypeOf(object)

	// if a pointer to a struct is passed, get the type of the dereferenced object
	if klazz.Kind() == reflect.Ptr {
		klazz = klazz.Elem()
	}

	if klazz.Kind() != reflect.Struct {
		//TODO: change it with logger
		fmt.Printf("%v type can't have attributes inspected\n", klazz.Kind())
		panic("wooow")
	}

	for i := 0; i < klazz.NumField(); i++ {
		p := klazz.Field(i)

		if !p.Anonymous {
			switch p.Type.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				result[p.Name] = strconv.FormatInt(p)
			case reflect.String:
				result[p.Name] = p.String()
				// etc...
			}
		}
	}

	return result
}
