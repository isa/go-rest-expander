package expander

import (
	"fmt"
	"net/url"
	"encoding/json"
	"reflect"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExpander(t *testing.T) {

	Convey("It should walk the given object and identify it's type:", t, func() {
		Convey("Walking the type should return empty key-values if the object is nil", func() {
			result := Expand(nil, "", "")

			So(result, ShouldBeEmpty)
		})

		Convey("Walking the type should return a map of all the simple key-values that user defines if expand is *", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["s"] = "bar"
			expectedMap["b"] = false
			expectedMap["i"] = -1
			expectedMap["f"] = 1.1
			expectedMap["ui"] = 1

			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			result := Expand(singleLevel, "*", "")

			So(result["s"], ShouldEqual, expectedMap["s"])
			So(result["b"], ShouldEqual, expectedMap["b"])
			So(result["i"], ShouldEqual, expectedMap["i"])
			So(result["f"], ShouldEqual, expectedMap["f"])
			So(result["ui"], ShouldEqual, expectedMap["ui"])
		})

		Convey("Walking the type should assume expansion is * if no expansion parameter is given and return all the simple key-values that user defines", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["s"] = "bar"
			expectedMap["b"] = false
			expectedMap["i"] = -1
			expectedMap["f"] = 1.1
			expectedMap["ui"] = 1

			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			result := Expand(singleLevel, "*", "")

			So(result["s"], ShouldEqual, expectedMap["s"])
			So(result["b"], ShouldEqual, expectedMap["b"])
			So(result["i"], ShouldEqual, expectedMap["i"])
			So(result["f"], ShouldEqual, expectedMap["f"])
			So(result["ui"], ShouldEqual, expectedMap["ui"])
		})

		Convey("Walking the type should return a map of all the simple with nested key-values that user defines if expand is *", func() {
			expectedMsb := map[string]bool{"key1": true, "key2": false}
			expectedMap := make(map[string]interface{})
			expectedMap["si"] = []int{1, 2}
			expectedMap["msb"] = expectedMsb

			singleMultiLevel := SimpleMultiLevel{expectedMap["si"].([]int), expectedMap["msb"].(map[string]bool)}
			result := Expand(singleMultiLevel, "*", "")

			So(result["si"], ShouldContain, 1)
			So(result["si"], ShouldContain, 2)

			msb := result["msb"].(map[interface{}]interface{})
			for k, v := range msb {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, expectedMsb[key])
			}
		})

		Convey("Walking the type should return a map of all the complex key-values that user defines if expand is *", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["s"] = "bar"
			simpleMap["b"] = false
			simpleMap["i"] = -1
			simpleMap["f"] = 1.1
			simpleMap["ui"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["ssl"] = simpleMap
			expectedMap["s"] = "a string"

			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			complexSingleLevel := ComplexSingleLevel{s: expectedMap["s"].(string), ssl: singleLevel}

			result := Expand(complexSingleLevel, "*", "")
			ssl := result["ssl"].(map[string]interface{})

			So(result["s"], ShouldEqual, expectedMap["s"])
			So(ssl["s"], ShouldEqual, simpleMap["s"])
			So(ssl["b"], ShouldEqual, simpleMap["b"])
			So(ssl["i"], ShouldEqual, simpleMap["i"])
			So(ssl["f"], ShouldEqual, simpleMap["f"])
			So(ssl["ui"], ShouldEqual, simpleMap["ui"])
		})
	})

	Convey("It should create a modification tree:", t, func() {
		Convey("Building a modification tree should be an empty expansion list when the expansion is *", func() {
			expansion := "*"
			result, _ := buildModificationTree(expansion)

			So(result, ShouldBeEmpty)
		})

		Convey("Building a modification tree should be a list of all fields when the expansion specifies them", func() {
			expansion := "a, b"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
		})

		Convey("Building a modification tree should be a list of all nested fields when the expansion specifies them", func() {
			expansion := "a, b(c, d)"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "c")
			So(result[1].Children[1].Value, ShouldEqual, "d")
		})

		Convey("Building a modification tree should be a list of all nested fields and more when the expansion specifies them", func() {
			expansion := "a, b(c, d), e"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 3)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
			So(result[2].Value, ShouldEqual, "e")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "c")
			So(result[1].Children[1].Value, ShouldEqual, "d")
		})

		Convey("Building a modification tree should be a list of all deeply-nested fields when the expansion specifies them", func() {
			expansion := "a, b(c(d, e), f), g"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 3)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
			So(result[2].Value, ShouldEqual, "g")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "c")
			So(result[1].Children[1].Value, ShouldEqual, "f")
			So(len(result[1].Children[0].Children), ShouldEqual, 2)
			So(result[1].Children[0].Children[0].Value, ShouldEqual, "d")
			So(result[1].Children[0].Children[1].Value, ShouldEqual, "e")
		})

		Convey("Building a modification tree should be a list of all confusingly deeply-nested fields when the expansion specifies them", func() {
			expansion := "a(b(c(d))), e"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "e")
			So(len(result[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Value, ShouldEqual, "b")
			So(len(result[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Value, ShouldEqual, "c")
			So(len(result[0].Children[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Children[0].Value, ShouldEqual, "d")
		})

		Convey("Building a modification tree should be a list of all nested fields when the expansion specifies only nested ones", func() {
			expansion := "a(b(c))"

			result, _ := buildModificationTree(expansion)

			So(len(result), ShouldEqual, 1)
			So(result[0].Value, ShouldEqual, "a")
			So(len(result[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Value, ShouldEqual, "b")
			So(len(result[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Value, ShouldEqual, "c")
		})
	})

	Convey("It should filter out the fields based on the given modification tree:", t, func() {
		Convey("Filtering should return an empty map when no Modifications is given", func() {
			result := filterOut("", nil)

			So(result, ShouldBeEmpty)
		})

		Convey("Filtering should return an empty map when no Data is given", func() {
			modifications := Modifications{}
			result := filterOut(nil, modifications)

			So(result, ShouldBeEmpty)
		})

		Convey("Filtering should return a map with only selected fields on simple objects based on the modification tree", func() {
			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			modifications := Modifications{}
			modifications = append(modifications, Modification{Value: "s"})
			modifications = append(modifications, Modification{Value: "i"})

			result := filterOut(singleLevel, modifications)

			So(result["s"], ShouldEqual, singleLevel.s)
			So(result["i"], ShouldEqual, singleLevel.i)
			So(result["b"], ShouldBeEmpty)
			So(result["f"], ShouldBeEmpty)
			So(result["ui"], ShouldBeEmpty)
		})

		Convey("Filtering should return a map with only selected fields on multilevel single objects based on the modification tree", func() {
			expectedMsb := map[string]bool{"key1": true, "key2": false}
			expectedMap := make(map[string]interface{})
			expectedMap["si"] = []int{1, 2}
			expectedMap["msb"] = expectedMsb

			singleMultiLevel := SimpleMultiLevel{expectedMap["si"].([]int), expectedMap["msb"].(map[string]bool)}

			child := Modification{Value: "key1"}
			parent := Modification{Value: "msb", Children: []Modification{child}}
			modifications := Modifications{}
			modifications = append(modifications, parent)

			result := filterOut(singleMultiLevel, modifications)
			msb := result["msb"].(map[interface{}]interface{})

			So(len(msb), ShouldEqual, 1)
			for k, v := range msb {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, expectedMsb[key])
			}
		})

		Convey("Filtering should return a map with only selected fields on complex objects based on the modification tree", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["s"] = "bar"
			simpleMap["b"] = false
			simpleMap["i"] = -1
			simpleMap["f"] = 1.1
			simpleMap["ui"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["ssl"] = simpleMap
			expectedMap["s"] = "a string"

			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			complexSingleLevel := ComplexSingleLevel{s: expectedMap["s"].(string), ssl: singleLevel}

			child1 := Modification{Value: "b"}
			child2 := Modification{Value: "f"}
			parent := Modification{Value: "ssl", Children: Modifications{child1, child2}}
			modifications := Modifications{}
			modifications = append(modifications, Modification{Value: "s"})
			modifications = append(modifications, parent)

			result := filterOut(complexSingleLevel, modifications)
			ssl := result["ssl"].(map[string]interface{})

			So(result["s"], ShouldEqual, complexSingleLevel.s)
			So(len(ssl), ShouldEqual, 2)
			for k, v := range ssl {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, simpleMap[key])
			}
		})

	})

	Convey("It should filter out the fields based on the given modification tree during expansion:", t, func() {
		Convey("Filtering should return the full map when no Modifications is given", func() {
			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}

			result := Expand(singleLevel, "", "")

			So(result["s"], ShouldEqual, singleLevel.s)
		})

		Convey("Filtering should return the filtered fields in simple object as map when first-level Modifications given", func() {
			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}

			result := Expand(singleLevel, "", "s, i")

			So(result["s"], ShouldEqual, singleLevel.s)
			So(result["i"], ShouldEqual, singleLevel.i)
			So(result["b"], ShouldBeEmpty)
			So(result["f"], ShouldBeEmpty)
			So(result["ui"], ShouldBeEmpty)
		})

		Convey("Filtering should return the filtered fields in complex object as map when multi-level Modifications given", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["s"] = "bar"
			simpleMap["b"] = false
			simpleMap["i"] = -1
			simpleMap["f"] = 1.1
			simpleMap["ui"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["ssl"] = simpleMap
			expectedMap["s"] = "a string"

			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			complexSingleLevel := ComplexSingleLevel{s: expectedMap["s"].(string), ssl: singleLevel}

			result := Expand(complexSingleLevel, "", "s,ssl(b, f, ui)")
			ssl := result["ssl"].(map[string]interface{})

			So(result["s"], ShouldEqual, complexSingleLevel.s)
			So(len(ssl), ShouldEqual, 3)
			for k, v := range ssl {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, simpleMap[key])
			}
		})
	})

	Convey("It should identify if given field is a reference field:", t, func() {
		Convey("Identifying should return false when field is not a struct", func() {
			info := Info{"A name", 100}
			v := reflect.ValueOf(info)

			result := isReference(v.Field(0))

			So(result, ShouldBeFalse)
		})

		Convey("Identifying should return false when field is not a hypermedia link", func() {
			singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}
			complexSingleLevel := ComplexSingleLevel{s: "something", ssl: singleLevel}

			v := reflect.ValueOf(complexSingleLevel)

			result := isReference(v.Field(0))

			So(result, ShouldBeFalse)
		})

		Convey("Identifying should return true when field is a hypermedia link", func() {
			singleLevel := SimpleSingleLevel{l: Link{ref: "http://valid", rel: "nothing", verb: "GET"}}

			v := reflect.ValueOf(singleLevel.l)

			result := isReference(v)

			So(result, ShouldBeTrue)
		})
	})

	Convey("It should find expansion URI if given field is a reference field:", t, func() {
		Convey("Identifying should return empty string when no ref field", func() {
			info := Info{"A name", 100}
			v := reflect.ValueOf(info)

			result := getReferenceURI(v)

			So(result, ShouldBeEmpty)
		})

		Convey("Identifying should return full URI when field is a ref field", func() {
			singleLevel := SimpleSingleLevel{l: Link{ref: "http://valid", rel: "nothing", verb: "GET"}}

			v := reflect.ValueOf(singleLevel.l)

			result := getReferenceURI(v)

			So(result, ShouldEqual, singleLevel.l.ref)
		})
	})

	Convey("It should fetch the underlying data from the URIs during expansion:", t, func() {
		Convey("Fetching should return the same value when non-URI data structure given", func() {
			singleLevel := SimpleSingleLevel{l: Link{ref: "non-URI", rel: "nothing", verb: "GET"}}

			result := Expand(singleLevel, "*", "")
			actual := result["l"].(map[string]interface{})

			So(actual["ref"], ShouldEqual, singleLevel.l.ref)
			So(actual["rel"], ShouldEqual, singleLevel.l.rel)
			So(actual["verb"], ShouldEqual, singleLevel.l.verb)
		})

		Convey("Fetching should return the same value when non-URI data structure given", func() {
			singleLevel := SimpleSingleLevel{l: Link{ref: "non-URI", rel: "nothing", verb: "GET"}}

			result := Expand(singleLevel, "*", "")
			actual := result["l"].(map[string]interface{})

			So(actual["ref"], ShouldEqual, singleLevel.l.ref)
			So(actual["rel"], ShouldEqual, singleLevel.l.rel)
			So(actual["verb"], ShouldEqual, singleLevel.l.verb)
		})

		Convey("Fetching should replace the value with expanded data structure when valid URI given", func() {
			singleLevel := SimpleSingleLevel{l: Link{ref: "http://valid", rel: "nothing", verb: "GET"}}
			info := Info{"A name", 100}

			mockedFn := getContentFrom
			getContentFrom = func(url *url.URL) string {
				result, _ := json.Marshal(info)
				return string(result)
			}

			result := Expand(singleLevel, "*", "")
			actual := result["l"].(map[string]interface{})

			So(actual["Name"], ShouldEqual, info.Name)
			So(actual["Age"], ShouldEqual, info.Age)

			getContentFrom = mockedFn
		})

		Convey("Fetching should replace an array of values with expanded data structures when valid URIs given", func() {
			links := []Link{
				Link{"http://valid1", "relation1", "GET"},
				Link{"http://valid2", "relation2", "GET"},
			}

			info := []Info {
				Info{"A name", 100},
				Info{"Another name", 200},
			}

			mockedFn := getContentFrom
			index := 0
			getContentFrom = func(url *url.URL) string {
				result, _ := json.Marshal(info[index])
				index = index + 1
				return string(result)
			}

			simpleWithLinks := SimpleWithLinks{"something", links}

			result := Expand(simpleWithLinks, "*", "")
			members := result["Members"].([]interface{})

			So(result["Name"], ShouldEqual, simpleWithLinks.Name)

			for i, v := range members {
				member := v.(map[string]interface{})

				So(member["Name"], ShouldEqual, info[i].Name)
				So(member["Age"], ShouldEqual, info[i].Age)
			}

			getContentFrom = mockedFn
		})
	})
}

type Link struct {
	ref string
	rel string
	verb string
}

type Info struct {
	Name string
	Age int
}

type SimpleWithLinks struct {
	Name string
	Members []Link
}

type SimpleSingleLevel struct {
	s  string
	b  bool
	i  int
	f  float64
	ui uint
	l Link
}

type SimpleMultiLevel struct {
	si  []int
	msb map[string]bool
}

type ComplexSingleLevel struct {
	ssl SimpleSingleLevel
	s   string
}
