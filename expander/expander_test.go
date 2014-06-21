package expander

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExpander(t *testing.T) {

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

	Convey("It should return a map of all the complex key-values that user defines if expand is *", t, func() {
		expectedMap := make(map[string]interface{})
		expectedMap["si"] = []int{1, 2}
		expectedMap["msb"] = map[string]bool{"key1": true, "key2": false}

		singleLevel := ComplexSingleLevel{expectedMap["si"].([]int), expectedMap["msb"].(map[string]bool)}

		testController := DummyController{ExpandingString: expectedMap}
		result := Expand(&testController, singleLevel)

		So(result["si"], ShouldContain, 1)
		So(result["si"], ShouldContain, 2)

		msb := result["msb"].(map[interface{}]interface{})
		for k, v := range msb {
			So(v, ShouldEqual, msb[k])
		}
	})

}

type SimpleSingleLevel struct {
	s  string
	b  bool
	i  int
	f  float64
	ui uint
}

type ComplexSingleLevel struct {
	si  []int
	msb map[string]bool
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
