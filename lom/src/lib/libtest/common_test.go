package libtest

import (
    "fmt"
)

func fatalFmt(s string, vals ...any) string {
    return fmt.Sprintf("***Fatal failure****: "+s, vals...)
}

func errorFmt(s string, vals ...any) string {
    return fmt.Sprintf("***Error failure****: "+s, vals...)
}

func logFmt(s string, vals ...any) string {
    return fmt.Sprintf("***Log****: "+s, vals...)
}
