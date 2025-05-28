package logger

import (
	"fmt"
	"os"
)

func Fatal(format string, a ...any) {
	fmt.Printf("❌❌❌ Fatal Error ❌❌❌ "+format+"\n", a)

	os.Exit(1)
}
