package safe

import (
	"fmt"
	"runtime/debug"

	"github.com/rambollwong/rainbowlog"
)

// Go is a utility function that executes a function concurrently in a goroutine.
// It recovers from any panics that occur within the goroutine and prints the panic error.
func Go(f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("panic: %+v", err)
				debug.PrintStack()
			}
		}()
		f()
	}()
}

// LoggerGo is a utility function that executes a function concurrently in a goroutine.
// It recovers from any panics that occur within the goroutine and logs the panic error using the provided logger.
func LoggerGo(logger *rainbowlog.Logger, f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().WithCallerSkip(4).Msgf("panic: %+v, stack: \n%s\n", err, string(debug.Stack())).Done()
			}
		}()
		f()
	}()
}
