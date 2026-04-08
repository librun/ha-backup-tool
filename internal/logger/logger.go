package logger

import (
	"fmt"
	"os"
)

func Fatalf(format string, a ...any) {
	fmt.Printf("❌❌❌ Fatal Error ❌❌❌ "+format+"\n", a...)

	os.Exit(1)
}
