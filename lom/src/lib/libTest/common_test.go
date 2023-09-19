package libTest

import (
    "fmt"
    "testing"
)

func fatalF(t *testing.T, s string) {
    t.Fatalf(fmt.Sprintf("***Fatal failure****: %s", s))
}

func errorF(t *testing.T, s string) {
    t.Errorf(fmt.Sprintf("***Error failure****: %s", s))
}

func logF(t *testing.T, s string) {
    t.Logf(fmt.Sprintf("***Log****: %s", s))
}


