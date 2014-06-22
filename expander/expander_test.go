package expander

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExpander(t *testing.T) {

	Convey("It should return empty key-values if the object is nil", t, func() {
		result := Expand(nil, nil)

		So(result, ShouldBeEmpty)
	})

	Convey("It should return all key-values that user defines", t, func() {
		testController := new(DummyController)
		_ = Expand(testController, nil)

		So(testController.UsingExpandKey, ShouldEqual, true)
	})

	Convey("It should return a map of all the simple key-values that user defines if expand is *", t, func() {
		expectedMap := make(map[string]interface{})
		expectedMap["s"] = "bar"
		expectedMap["b"] = false
		expectedMap["i"] = -1
		expectedMap["f"] = 1.1
		expectedMap["ui"] = 1

		singleLevel := SimpleSingleLevel{s: "bar", b: false, i: -1, f: 1.1, ui: 1}

		testController := DummyController{ExpandingString: expectedMap}
		result := Expand(&testController, singleLevel)

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

		testController := DummyController{ExpandingString: expectedMap}
		result := Expand(&testController, singleMultiLevel)

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

		testController := DummyController{ExpandingString: expectedMap}
		result := Expand(&testController, complexSingleLevel)
		ssl := result["ssl"].(map[string]interface{})

		So(result["s"], ShouldEqual, expectedMap["s"])
		So(ssl["s"], ShouldEqual, simpleMap["s"])
		So(ssl["b"], ShouldEqual, simpleMap["b"])
		So(ssl["i"], ShouldEqual, simpleMap["i"])
		So(ssl["f"], ShouldEqual, simpleMap["f"])
		So(ssl["ui"], ShouldEqual, simpleMap["ui"])
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

type DummyController struct {
	Parametric
	UsingExpandKey  bool
	ExpandingString map[string]interface{}
}

func (c *DummyController) GetString(key string) string {
	predefinedKeys := make(map[string]string)
	predefinedKeys[key] = "*"

	if key == "expand" {
		c.UsingExpandKey = true
	}

	return predefinedKeys[key]
}
