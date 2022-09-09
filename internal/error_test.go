/*
Copyright © 2020 gjbae1212
Released under the MIT license.
(https://github.com/gjbae1212/gossm)
*/

package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapError(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		err error
	}{
		"error": {err: fmt.Errorf("[err] obj error")},
	}

	for _, t := range tests {
		err := WrapError(t.err)
		switch t.err.(type) {
		case error:
			assert.True(errors.Is(err, t.err))
		}
		fmt.Println(err)
	}
}
