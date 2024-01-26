package client

import (
    // cmn "lom/src/lib/lomcommon"
    "reflect"
    "testing"
)

func TestDropped(t *testing.T) {
    dExpect := map[string]int{
        "foo": 5,
        "bar": 1,
    }

    for k, v := range dExpect {
        for i := 0; i < v; i++ {
            droppedData.inc(k)
        }
    }

    if !reflect.DeepEqual(dExpect, droppedData.data) {
        t.Fatalf("Incorrect data exp(%v) != res(%v)", dExpect, droppedData)
    }
}
