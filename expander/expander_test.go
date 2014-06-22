package expander

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExpander(t *testing.T) {

	Convey("It should return empty key-values if the object is nil", t, func() {
		result := Expand(nil, "", "")

		So(result, ShouldBeEmpty)
	})

	Convey("It should return a map of all the simple key-values that user defines if expand is *", t, func() {
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

	Convey("It should assume expansion is * if no expansion parameter is given and return all the simple key-values that user defines", t, func() {
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

	Convey("It should return a map of all the simple with nested key-values that user defines if expand is *", t, func() {
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

	Convey("It should return a map of all the complex key-values that user defines if expand is *", t, func() {
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

	Convey("It should return an empty expansion list when the expansion is *", t, func() {
		expansion := "*"
		result := buildModifyTree(expansion)

		So(result, ShouldBeEmpty)
	})

	Convey("It should return a list of all fields when the expansion specifies them", t, func() {
		expansion := "a, b"

		result := buildModifyTree(expansion)

		So(len(result), ShouldEqual, 2)
		So(result[0].Value, ShouldEqual, "a")
		So(result[1].Value, ShouldEqual, "b")
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
