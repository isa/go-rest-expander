package expander

import (
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestExpander(t *testing.T) {

	Convey("It should walk the given object and identify it's type:", t, func() {
		Convey("Walking the type should return empty key-values if the object is nil", func() {
			result := Expand(nil, "", "")

			So(result, ShouldBeEmpty)
		})

		Convey("Walking the type should return a map of all the visible simple key-values that user defines if expand is *", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["S"] = "bar"
			expectedMap["B"] = false
			expectedMap["I"] = -1
			expectedMap["F"] = 1.1
			expectedMap["UI"] = 1

			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}
			result := Expand(singleLevel, "*", "")

			So(result["S"], ShouldEqual, expectedMap["S"])
			So(result["B"], ShouldEqual, expectedMap["B"])
			So(result["I"], ShouldEqual, expectedMap["I"])
			So(result["F"], ShouldEqual, expectedMap["F"])
			So(result["UI"], ShouldEqual, expectedMap["UI"])
		})

		Convey("Walking the type should return a map of all the visible simple key-values that user defines if expand is *", func() {
			simpleWithTime := SimpleWithTime{Name: "foo", Time: time.Now()}
			expectedMap := make(map[string]string)
			expectedMap["Name"] = simpleWithTime.Name
			time, _ := simpleWithTime.Time.MarshalJSON()
			expectedMap["Time"] = string(time)

			result := Expand(simpleWithTime, "*", "")

			So(result["Name"], ShouldEqual, expectedMap["Name"])
			So(result["Time"], ShouldEqual, expectedMap["Time"])
		})

		Convey("Walking the type should assume expansion is * if no expansion parameter is given and return all the simple key-values that user defines", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["S"] = "bar"
			expectedMap["B"] = false
			expectedMap["I"] = -1
			expectedMap["F"] = 1.1
			expectedMap["UI"] = 1

			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}
			result := Expand(singleLevel, "*", "")

			So(result["S"], ShouldEqual, expectedMap["S"])
			So(result["B"], ShouldEqual, expectedMap["B"])
			So(result["I"], ShouldEqual, expectedMap["I"])
			So(result["F"], ShouldEqual, expectedMap["F"])
			So(result["UI"], ShouldEqual, expectedMap["UI"])
		})

		Convey("Walking the type should return a map of all the simple with nested key-values that user defines if expand is *", func() {
			expectedMsb := map[string]bool{"key1": true, "key2": false}
			expectedMap := make(map[string]interface{})
			expectedMap["SI"] = []int{1, 2}
			expectedMap["MSB"] = expectedMsb

			singleMultiLevel := SimpleMultiLevel{expectedMap["SI"].([]int), expectedMap["MSB"].(map[string]bool)}
			result := Expand(singleMultiLevel, "*", "")

			So(result["SI"], ShouldContain, 1)
			So(result["SI"], ShouldContain, 2)

			msb := result["MSB"].(map[string]interface{})
			for k, v := range msb {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, expectedMsb[key])
			}
		})

		Convey("Walking the type should return a map of all the complex key-values that user defines if expand is *", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["S"] = "bar"
			simpleMap["B"] = false
			simpleMap["I"] = -1
			simpleMap["F"] = 1.1
			simpleMap["UI"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["SSL"] = simpleMap
			expectedMap["S"] = "a string"

			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}
			complexSingleLevel := ComplexSingleLevel{S: expectedMap["S"].(string), SSL: singleLevel}

			result := Expand(complexSingleLevel, "*", "")
			ssl := result["SSL"].(map[string]interface{})

			So(result["S"], ShouldEqual, expectedMap["S"])
			So(ssl["S"], ShouldEqual, simpleMap["S"])
			So(ssl["B"], ShouldEqual, simpleMap["B"])
			So(ssl["I"], ShouldEqual, simpleMap["I"])
			So(ssl["F"], ShouldEqual, simpleMap["F"])
			So(ssl["UI"], ShouldEqual, simpleMap["UI"])
		})
	})

	Convey("It should create a modification tree:", t, func() {
		Convey("Building a modification tree should be an empty expansion list when the expansion is *", func() {
			expansion := "*"
			result, _ := buildFilterTree(expansion)

			So(result, ShouldBeEmpty)
		})

		Convey("Building a modification tree should be an empty expansion list when the expansion is not specified", func() {
			expansion := ""
			result, _ := buildFilterTree(expansion)

			So(result, ShouldBeEmpty)
		})

		Convey("Building a modification tree should be a list of single field when the expansion specifies only one", func() {
			expansion := "A"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 1)
			So(result[0].Value, ShouldEqual, "A")
		})

		Convey("Building a modification tree should be a list of all fields when the expansion specifies them", func() {
			expansion := "A, B"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "A")
			So(result[1].Value, ShouldEqual, "B")
		})

		Convey("Building a modification tree should be a list of all nested fields when the expansion specifies them", func() {
			expansion := "A, B(C, D)"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "A")
			So(result[1].Value, ShouldEqual, "B")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "C")
			So(result[1].Children[1].Value, ShouldEqual, "D")
		})

		Convey("Building a modification tree should be a list of all nested fields and more when the expansion specifies them", func() {
			expansion := "A, B(C, D), E"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 3)
			So(result[0].Value, ShouldEqual, "A")
			So(result[1].Value, ShouldEqual, "B")
			So(result[2].Value, ShouldEqual, "E")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "C")
			So(result[1].Children[1].Value, ShouldEqual, "D")
		})

		Convey("Building a modification tree should be a list of all deeply-nested fields when the expansion specifies them", func() {
			expansion := "A, B(C(D, E), F), G"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 3)
			So(result[0].Value, ShouldEqual, "A")
			So(result[1].Value, ShouldEqual, "B")
			So(result[2].Value, ShouldEqual, "G")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "C")
			So(result[1].Children[1].Value, ShouldEqual, "F")
			So(len(result[1].Children[0].Children), ShouldEqual, 2)
			So(result[1].Children[0].Children[0].Value, ShouldEqual, "D")
			So(result[1].Children[0].Children[1].Value, ShouldEqual, "E")
		})

		Convey("Building a modification tree should be a list of all confusingly deeply-nested fields when the expansion specifies them", func() {
			expansion := "A(B(C(D))), E"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "A")
			So(result[1].Value, ShouldEqual, "E")
			So(len(result[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Value, ShouldEqual, "B")
			So(len(result[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Value, ShouldEqual, "C")
			So(len(result[0].Children[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Children[0].Value, ShouldEqual, "D")
		})

		Convey("Building a modification tree should be a list of all nested fields when the expansion specifies only nested ones", func() {
			expansion := "A(B(C))"

			result, _ := buildFilterTree(expansion)

			So(len(result), ShouldEqual, 1)
			So(result[0].Value, ShouldEqual, "A")
			So(len(result[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Value, ShouldEqual, "B")
			So(len(result[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Value, ShouldEqual, "C")
		})
	})

	Convey("It should filter out the fields based on the given modification tree:", t, func() {
		Convey("Filtering should return an empty map when no Data is given", func() {
			filters := Filters{}
			result := walkByFilter(nil, filters)

			So(result, ShouldBeEmpty)
		})

		Convey("Filtering should return a map with only selected fields on simple objects based on the modification tree", func() {
			singleLevel := map[string]interface{}{"S": "bar", "B": false, "I": -1, "F": 1.1, "UI": 1}
			filters := Filters{}
			filters = append(filters, Filter{Value: "S"})
			filters = append(filters, Filter{Value: "I"})

			result := walkByFilter(singleLevel, filters)

			So(result["S"], ShouldEqual, singleLevel["S"])
			So(result["I"], ShouldEqual, singleLevel["I"])
			So(result["B"], ShouldBeEmpty)
			So(result["F"], ShouldBeEmpty)
			So(result["UI"], ShouldBeEmpty)
		})

		Convey("Filtering should return a map with only selected fields on multilevel single objects based on the modification tree", func() {
			expectedMsb := map[string]interface{}{"key1": true, "key2": false}
			expectedMap := make(map[string]interface{})
			expectedMap["SI"] = []int{1, 2}
			expectedMap["MSB"] = expectedMsb

			child := Filter{Value: "key1"}
			parent := Filter{Value: "MSB", Children: []Filter{child}}
			filters := Filters{}
			filters = append(filters, parent)

			result := walkByFilter(expectedMap, filters)
			msb := result["MSB"].(map[string]interface{})

			So(len(msb), ShouldEqual, 1)
			for k, v := range msb {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, expectedMsb[key])
			}
		})

		Convey("Filtering should return a map with empty list on multilevel single objects if empty list given", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["SI"] = []int{}
			filters := Filters{}

			result := walkByFilter(expectedMap, filters)
			si := result["SI"].([]int)

			So(len(si), ShouldEqual, 0)
		})

		Convey("Filtering should return a map with only selected fields on simple-multilevel objects based on the modification tree", func() {
			expectedMap := make(map[string]interface{})
			expectedMap["S"] = "a string"
			expectedChildren := make([]map[string]interface{}, 0)
			expectedChildren = append(expectedChildren, map[string]interface{}{
				"key1": "value1",
				"key2": 0,
			})
			expectedChildren = append(expectedChildren, map[string]interface{}{
				"key1": "value2",
				"key2": 1,
			})
			expectedMap["Children"] = expectedChildren

			parent := Filter{Value: "Children", Children: Filters{Filter{Value: "key2"}}}
			filters := Filters{}
			filters = append(filters, Filter{Value: "S"})
			filters = append(filters, parent)

			result := walkByFilter(expectedMap, filters)
			children := result["Children"].([]map[string]interface{})

			So(result["S"], ShouldEqual, expectedMap["S"])
			for i, v := range children {
				So(v["key1"], ShouldBeEmpty)
				So(v["key2"], ShouldEqual, i)
			}
		})

		Convey("Filtering should return a map with only selected fields on complex objects based on the modification tree", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["S"] = "bar"
			simpleMap["B"] = false
			simpleMap["I"] = -1
			simpleMap["F"] = 1.1
			simpleMap["UI"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["SSL"] = simpleMap
			expectedMap["S"] = "a string"

			child1 := Filter{Value: "B"}
			child2 := Filter{Value: "F"}
			parent := Filter{Value: "SSL", Children: Filters{child1, child2}}
			filters := Filters{}
			filters = append(filters, Filter{Value: "S"})
			filters = append(filters, parent)

			result := walkByFilter(expectedMap, filters)
			ssl := result["SSL"].(map[string]interface{})

			So(result["S"], ShouldEqual, expectedMap["S"])
			So(len(ssl), ShouldEqual, 2)
			for k, v := range ssl {
				key := fmt.Sprintf("%v", k)
				So(v, ShouldEqual, simpleMap[key])
			}
		})

	})

	Convey("It should filter out the fields based on the given modification tree during expansion:", t, func() {
		Convey("Filtering should return the full map when no Filters is given", func() {
			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}

			result := Expand(singleLevel, "", "")

			So(result["S"], ShouldEqual, singleLevel.S)
		})

		Convey("Filtering should return the filtered fields in simple object as map when first-level Filters given", func() {
			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}

			result := Expand(singleLevel, "", "S, I")

			So(result["S"], ShouldEqual, singleLevel.S)
			So(result["I"], ShouldEqual, singleLevel.I)
			So(result["B"], ShouldBeEmpty)
			So(result["F"], ShouldBeEmpty)
			So(result["UI"], ShouldBeEmpty)
		})

		Convey("Filtering should return the filtered fields in complex object as map when multi-level Filters given", func() {
			simpleMap := make(map[string]interface{})
			simpleMap["S"] = "bar"
			simpleMap["B"] = false
			simpleMap["I"] = -1
			simpleMap["F"] = 1.1
			simpleMap["UI"] = 1

			expectedMap := make(map[string]interface{})
			expectedMap["SSL"] = simpleMap
			expectedMap["S"] = "a string"

			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}
			complexSingleLevel := ComplexSingleLevel{S: expectedMap["S"].(string), SSL: singleLevel}

			result := Expand(complexSingleLevel, "", "S,SSL(B, F, UI)")
			ssl := result["SSL"].(map[string]interface{})

			So(result["S"], ShouldEqual, complexSingleLevel.S)
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
			singleLevel := SimpleSingleLevel{S: "bar", B: false, I: -1, F: 1.1, UI: 1}
			complexSingleLevel := ComplexSingleLevel{S: "something", SSL: singleLevel}

			v := reflect.ValueOf(complexSingleLevel)

			result := isReference(v.Field(0))

			So(result, ShouldBeFalse)
		})

		Convey("Identifying should return true when field is a hypermedia link", func() {
			singleLevel := SimpleSingleLevel{L: Link{Ref: "http://valid", Rel: "nothing", Verb: "GET"}}

			v := reflect.ValueOf(singleLevel.L)

			result := isReference(v)

			So(result, ShouldBeTrue)
		})

		Convey("Identifying should return false when field doesn't have a hypermedia link", func() {
			m := map[string]interface{}{
				"a_key": "something",
			}

			result := hasReference(m)

			So(result, ShouldBeFalse)
		})

		Convey("Identifying should return true when field has a hypermedia link", func() {
			m := map[string]interface{}{
				"a_key": "something",
				"a_link": map[string]interface{}{
					"ref":  "http://valid",
					"rel":  "a-relation",
					"verb": "GET",
				},
			}

			result := hasReference(m)

			So(result, ShouldBeTrue)
		})

		Convey("Identifying should return true when nested field has a hypermedia link", func() {
			m := map[string]interface{}{
				"a_key": "something",
				"another_key": map[string]interface{}{
					"some-id": "333",
					"a_link": map[string]interface{}{
						"ref":  "http://valid",
						"rel":  "a-relation",
						"verb": "GET",
					},
				},
			}

			result := hasReference(m)

			So(result, ShouldBeTrue)
		})

		Convey("Identifying should return false when nested field doesn't have a hypermedia link", func() {
			m := map[string]interface{}{
				"a_key": "something",
				"another_key": map[string]interface{}{
					"some-id": "333",
					"another_type": map[string]interface{}{
						"something": "yeap",
					},
				},
			}

			result := hasReference(m)

			So(result, ShouldBeFalse)
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
			singleLevel := SimpleSingleLevel{L: Link{Ref: "http://valid", Rel: "nothing", Verb: "GET"}}

			v := reflect.ValueOf(singleLevel.L)

			result := getReferenceURI(v)

			So(result, ShouldEqual, singleLevel.L.Ref)
		})
	})

	Convey("It should fetch the underlying data from the Mongo during expansion:", t, func() {
		Convey("Fetching should return the same value when Mongo flag is not set", func() {
			simple := SimpleWithDBRef{Name: "foo", Ref: DBRef{"a collection", "an id", "a database"}}

			result := Expand(simple, "*", "")
			mongoRef := result["Ref"].(map[string]interface{})
			So(result["name"], ShouldEqual, simple.Name)
			So(mongoRef["Collection"], ShouldEqual, simple.Ref.Collection)
			So(mongoRef["Id"], ShouldEqual, simple.Ref.Id)
			So(mongoRef["Database"], ShouldEqual, simple.Ref.Database)
		})

		Convey("Fetching should return the same value when Mongo flag is set to false", func() {
			simple := SimpleWithDBRef{Name: "foo", Ref: DBRef{"a collection", "an id", "a database"}}

			ExpanderConfig = Configuration{UsingMongo: false}
			result := Expand(simple, "*", "")
			mongoRef := result["Ref"].(map[string]interface{})

			So(result["name"], ShouldEqual, simple.Name)
			So(mongoRef["Collection"], ShouldEqual, simple.Ref.Collection)
			So(mongoRef["Id"], ShouldEqual, simple.Ref.Id)
			So(mongoRef["Database"], ShouldEqual, simple.Ref.Database)
		})

		Convey("Fetching should return the same value when Mongo flag is set to true without IdURIs", func() {
			simple := SimpleWithDBRef{Name: "foo", Ref: DBRef{"a collection", MongoId("123"), "a database"}}

			ExpanderConfig = Configuration{UsingMongo: true}
			result := Expand(simple, "*", "")
			mongoRef := result["Ref"].(map[string]interface{})

			So(result["name"], ShouldEqual, simple.Name)
			So(mongoRef["Collection"], ShouldEqual, simple.Ref.Collection)
			So(mongoRef["Id"], ShouldEqual, simple.Ref.Id)
			So(mongoRef["Database"], ShouldEqual, simple.Ref.Database)
		})

		Convey("Fetching should return the underlying value when Mongo flag is set to true with proper IdURIs", func() {
			simple := SimpleWithDBRef{Name: "foo", Ref: DBRef{"a collection", MongoId("123"), "a database"}}
			info := Info{"A name", 100}
			uris := map[string]string{simple.Ref.Collection: "http://some-uri/id"}

			ExpanderConfig = Configuration{UsingMongo: true, IdURIs: uris}
			mockedFn := getContentFrom
			getContentFrom = func(url *url.URL) string {
				result, _ := json.Marshal(info)
				return string(result)
			}

			result := Expand(simple, "*", "")
			mongoRef := result["Ref"].(map[string]interface{})

			So(result["name"], ShouldEqual, simple.Name)
			So(mongoRef["Name"], ShouldEqual, info.Name)
			So(mongoRef["Age"], ShouldEqual, info.Age)

			getContentFrom = mockedFn
		})

		Convey("Fetching should return a list of underlying values when Mongo flag is set to true with proper IdURIs", func() {
			simple := SimpleWithMultipleDBRefs{
				Name: "foo",
				Refs: []DBRef{
					{"a collection", MongoId("123"), "a database"},
					{"another collection", MongoId("234"), "another database"},
				},
			}
			info := Info{"A name", 100}
			uris := map[string]string{
				"a collection":       "http://some-uri/id",
				"another collection": "http://some-other-uri/id",
			}

			ExpanderConfig = Configuration{UsingMongo: true, IdURIs: uris}
			mockedFn := getContentFrom
			getContentFrom = func(url *url.URL) string {
				result, _ := json.Marshal(info)
				return string(result)
			}

			result := Expand(simple, "*", "")
			mongoRef := result["Refs"].([]interface{})
			child1 := mongoRef[0].(map[string]interface{})
			child2 := mongoRef[1].(map[string]interface{})

			So(result["Name"], ShouldEqual, simple.Name)
			So(child1["Name"], ShouldEqual, info.Name)
			So(child1["Age"], ShouldEqual, info.Age)
			So(child2["Name"], ShouldEqual, info.Name)
			So(child2["Age"], ShouldEqual, info.Age)

			getContentFrom = mockedFn
		})

	})

	Convey("It should fetch the underlying data from the URIs during expansion:", t, func() {
		Convey("Fetching should return the same value when non-URI data structure given", func() {
			singleLevel := SimpleSingleLevel{L: Link{Ref: "non-URI", Rel: "nothing", Verb: "GET"}}

			result := Expand(singleLevel, "*", "")
			actual := result["L"].(map[string]interface{})

			So(actual["ref"], ShouldEqual, singleLevel.L.Ref)
			So(actual["rel"], ShouldEqual, singleLevel.L.Rel)
			So(actual["verb"], ShouldEqual, singleLevel.L.Verb)
		})

		Convey("Fetching should return the same value when non-URI data structure given", func() {
			singleLevel := SimpleSingleLevel{L: Link{Ref: "non-URI", Rel: "nothing", Verb: "GET"}}

			result := Expand(singleLevel, "*", "")
			actual := result["L"].(map[string]interface{})

			So(actual["ref"], ShouldEqual, singleLevel.L.Ref)
			So(actual["rel"], ShouldEqual, singleLevel.L.Rel)
			So(actual["verb"], ShouldEqual, singleLevel.L.Verb)
		})

		Convey("Fetching should replace the value with expanded data structure when valid URI given", func() {
			singleLevel := SimpleSingleLevel{L: Link{Ref: "http://valid", Rel: "nothing", Verb: "GET"}}
			info := Info{"A name", 100}

			mockedFn := getContentFrom
			getContentFrom = func(url *url.URL) string {
				result, _ := json.Marshal(info)
				return string(result)
			}

			result := Expand(singleLevel, "*", "")
			actual := result["L"].(map[string]interface{})

			So(actual["Name"], ShouldEqual, info.Name)
			So(actual["Age"], ShouldEqual, info.Age)

			getContentFrom = mockedFn
		})

		Convey("Fetching should replace an array of values with expanded data structures when valid URIs given", func() {
			links := []Link{
				Link{"http://valid1", "relation1", "GET"},
				Link{"http://valid2", "relation2", "GET"},
			}

			info := []Info{
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

		Convey("Fetching should replace the value recursively with expanded data structure when valid URIs given", func() {
			singleLevel1 := SimpleSingleLevel{S: "one", L: Link{Ref: "http://valid1/ssl", Rel: "nothing1", Verb: "GET"}}
			singleLevel2 := SimpleSingleLevel{S: "two", L: Link{Ref: "http://valid2/info", Rel: "nothing2", Verb: "GET"}}
			info := Info{"A name", 100}

			mockedFn := getContentFrom
			index := 0
			getContentFrom = func(url *url.URL) string {
				var result []byte
				if index > 0 {
					result, _ = json.Marshal(info)
					return string(result)
				}
				result, _ = json.Marshal(singleLevel2)
				index = index + 1
				return string(result)
			}

			result := Expand(singleLevel1, "*", "")
			parent := result["L"].(map[string]interface{})
			child := parent["L"].(map[string]interface{})

			So(result["S"], ShouldEqual, singleLevel1.S)
			So(parent["S"], ShouldEqual, singleLevel2.S)
			So(child["Name"], ShouldEqual, info.Name)
			So(child["Age"], ShouldEqual, info.Age)

			getContentFrom = mockedFn
		})

		Convey("Expanding should replace the value recursively and filter the expanded data structure when valid URIs given", func() {
			singleLevel1 := SimpleSingleLevel{S: "one", L: Link{Ref: "http://valid1/ssl", Rel: "nothing1", Verb: "GET"}}
			singleLevel2 := SimpleSingleLevel{S: "two", L: Link{Ref: "http://valid2/info", Rel: "nothing2", Verb: "GET"}}

			mockedFn := getContentFrom
			getContentFrom = func(url *url.URL) string {
				var result []byte
				result, _ = json.Marshal(singleLevel2)
				return string(result)
			}

			result := Expand(singleLevel1, "L", "")
			parent := result["L"].(map[string]interface{})
			child := parent["L"].(map[string]interface{})

			So(result["S"], ShouldEqual, singleLevel1.S)
			So(parent["S"], ShouldEqual, singleLevel2.S)
			So(child["ref"], ShouldEqual, singleLevel2.L.Ref)
			So(child["rel"], ShouldEqual, singleLevel2.L.Rel)
			So(child["verb"], ShouldEqual, singleLevel2.L.Verb)

			getContentFrom = mockedFn
		})

		Convey("Expanding should replace the value recursively and filter the expanded data structure when data contains a list of nested sub-types", func() {
			link1 := Link{Ref: "http://valid1/ssl", Rel: "nothing1", Verb: "GET"}
			link2 := Link{Ref: "http://valid2/ssl", Rel: "nothing2", Verb: "GET"}
			singleLevel := SimpleSingleLevel{S: "one", L: link1}
			info := Info{"A name", 100}
			simpleWithLinks := SimpleWithLinks{
				Name:    "lorem",
				Members: []Link{link1, link2},
			}

			mockedFn := getContentFrom
			index := 0
			getContentFrom = func(url *url.URL) string {
				var result []byte
				index = index + 1
				if index%2 == 0 {
					result, _ = json.Marshal(info)
					return string(result)
				}
				result, _ = json.Marshal(singleLevel)
				return string(result)
			}

			result := Expand(simpleWithLinks, "Members(L)", "Name,Members(S,L)")
			parent := result["Members"].([]interface{})

			So(len(result), ShouldEqual, 2)

			child1 := parent[0].(map[string]interface{})
			So(child1["S"], ShouldEqual, singleLevel.S)

			actualLink := child1["L"].(map[string]interface{})
			So(actualLink["Name"], ShouldEqual, info.Name)

			getContentFrom = mockedFn
		})

	})
}

type Link struct {
	Ref  string `json:"ref"`
	Rel  string `json:"rel"`
	Verb string `json:"verb"`
}

type MongoId string

func (m MongoId) Hex() string {
	return string(m)
}

type Info struct {
	Name string
	Age  int
}

type SimpleWithLinks struct {
	Name    string
	Members []Link
}

type SimpleWithDBRef struct {
	Name string `json:"name,omitempty"`
	Ref  DBRef  `json: "ref", bson: "ref"`
}

type SimpleWithTime struct {
	Name string
	Time time.Time
}

type SimpleWithMultipleDBRefs struct {
	Name string
	Refs []DBRef
}

type SimpleSingleLevel struct {
	S  string
	B  bool
	I  int
	F  float64
	UI uint
	// hidden bool
	L Link
}

type SimpleMultiLevel struct {
	SI  []int
	MSB map[string]bool
}

type ComplexSingleLevel struct {
	SSL SimpleSingleLevel
	S   string
}
