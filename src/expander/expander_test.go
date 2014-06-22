package expander

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	// "fmt"
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
			expectedMap := make(map[string]interface{})
			expectedMap["si"] = []int{1, 2}
			expectedMap["msb"] = map[string]bool{"key1": true, "key2": false}

			singleMultiLevel := SimpleMultiLevel{expectedMap["si"].([]int), expectedMap["msb"].(map[string]bool)}
			result := Expand(singleMultiLevel, "*", "")

			So(result["si"], ShouldContain, 1)
			So(result["si"], ShouldContain, 2)

			msb := result["msb"].(map[interface{}]interface{})
			for k, v := range msb {
				So(v, ShouldEqual, msb[k])
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
			result, _ := buildModifyTree(expansion)

			So(result, ShouldBeEmpty)
		})

		Convey("Building a modification tree should be a list of all fields when the expansion specifies them", func() {
			expansion := "a, b"

			result, _ := buildModifyTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
		})

		Convey("Building a modification tree should be a list of all nested fields when the expansion specifies them", func() {
			expansion := "a, b(c, d)"

			result, _ := buildModifyTree(expansion)

			So(len(result), ShouldEqual, 2)
			So(result[0].Value, ShouldEqual, "a")
			So(result[1].Value, ShouldEqual, "b")
			So(len(result[1].Children), ShouldEqual, 2)
			So(result[1].Children[0].Value, ShouldEqual, "c")
			So(result[1].Children[1].Value, ShouldEqual, "d")
		})

		Convey("Building a modification tree should be a list of all nested fields and more when the expansion specifies them", func() {
			expansion := "a, b(c, d), e"

			result, _ := buildModifyTree(expansion)

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

			result, _ := buildModifyTree(expansion)

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

			result, _ := buildModifyTree(expansion)

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

			result, _ := buildModifyTree(expansion)

			So(len(result), ShouldEqual, 1)
			So(result[0].Value, ShouldEqual, "a")
			So(len(result[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Value, ShouldEqual, "b")
			So(len(result[0].Children[0].Children), ShouldEqual, 1)
			So(result[0].Children[0].Children[0].Value, ShouldEqual, "c")
		})
	})
}

type SimpleSingleLevel struct {
	s  string
	b  bool
	i  int
	f  float64
	ui uint
}

type SimpleMultiLevel struct {
	si  []int
	msb map[string]bool
}

type ComplexSingleLevel struct {
	ssl SimpleSingleLevel
	s   string
}
