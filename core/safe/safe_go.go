package safe

import (
	"fmt"

	"github.com/rambollwong/rainbowlog"
)

// Go is a utility function that executes a function concurrently in a goroutine.
// It recovers from any panics that occur within the goroutine and prints the panic error.
func Go(f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("panic: %+v", err)
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
				logger.Error().Msgf("panic: %+v", err).Done()
			}
		}()
		f()
	}()
}
