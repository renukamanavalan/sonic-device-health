package libtest

import (
    "fmt"
    "time"
)

func fatalFmt(s string, vals ...any) string {
    prefix := fmt.Sprintf("***Fatal failure****:(%v): (%s)", time.Now().UnixMilli(), s)
    return fmt.Sprintf(prefix, vals...)
}

func errorFmt(s string, vals ...any) string {
    prefix := fmt.Sprintf("***Error failure****:(%v): (%s)", time.Now().UnixMilli(), s)
    return fmt.Sprintf(prefix, vals...)
}

func logFmt(s string, vals ...any) string {
    prefix := fmt.Sprintf("***Log****:(%v): (%s)", time.Now().UnixMilli(), s)
    return fmt.Sprintf(prefix, vals...)
}
