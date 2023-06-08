package lib_test

import (
    "fmt"
    "github.com/stretchr/testify/assert"
    . "lom/src/lib/lomcommon"
    "testing"
)

/* Validates GetFloatConfigurationFromJson for various input json strings and config key */
func Test_GetFloatConfigFromMapping(t *testing.T) {
	map1 := map[string]interface{}{"abc": 123.99}
	map2 := map[string]interface{}{"abc": "ghi"}
	map3 := map1
	map4 := make(map[string]interface{})
	expectedResult := []float64{123.99, 222.22, 222.22, 222.22}
	configKey := []string{"abc", "abc", "xyz", "abc"}
	arrayOfMaps := []map[string]interface{}{map1, map2, map3, map4}

	assert := assert.New(t)
	for index := 0; index < len(arrayOfMaps); index++ {
		result := GetFloatConfigFromMapping(arrayOfMaps[index], configKey[index], 222.22)
		assert.Equal(expectedResult[index], result, fmt.Sprintf("GetFloatConfigFromMapping Failed for test index : %d", index))
	}
}
