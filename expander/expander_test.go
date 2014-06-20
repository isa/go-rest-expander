package expander

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type DummySingleLevel struct {
	foo string
	baz bool
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

func TestExpander(t *testing.T) {

	Convey("It should return all key-values that user defines", t, func() {
		testController := new(DummyController)
		_ = Expand(testController, nil)

		So(testController.UsingExpandKey, ShouldEqual, true)
	})

	Convey("It should return a map of all the key-values that user defined if expand is *", t, func() {
		expectedMap := make(map[string]interface{})
		expectedMap["foo"] = "bar"
		expectedMap["baz"] = false

		singleLevel := DummySingleLevel{foo: "bar", baz: false}

		testController := DummyController{ExpandingString: expectedMap}
		result := Expand(&testController, singleLevel)

		So(result, ShouldResemble, expectedMap)
	})

}
