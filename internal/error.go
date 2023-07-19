package internal

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/ghdwlsgur/outline-vpn/wraperror"
	"github.com/spf13/cobra"
)

var (
	// ErrInvalidParams is an error type to use when passed arguments are invalid.
	ErrInvalidParams = errors.New("[err] invalid params")
	// ErrUnknown is an error type to use when error reason doesn't know.
	ErrUnknown = errors.New("[err] unknown")
)

// WrapError wraps error.
func WrapError(err error) error {
	if err != nil {
		// Get program counter and line number
		pc, _, line, _ := runtime.Caller(1)
		// Get function name from program counter
		fn := runtime.FuncForPC(pc).Name()
		// Refine function name
		details := strings.Split(fn, "/")
		fn = details[len(details)-1]
		// Build chain
		chainErr := wraperror.Error(err)
		return chainErr.Wrap(fmt.Errorf("[err][%s:%d]", fn, line))
	}
	return nil
}

func WrapArgsError(argFn cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		err := argFn(cmd, args)
		if err == nil {
			return nil
		}

		return fmt.Errorf("usage:  %s %s",
			cmd.CommandPath(),
			cmd.ValidArgs,
		)
	}
}
