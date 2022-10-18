package slog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Slog(t *testing.T) {
	var out bytes.Buffer
	l := New(&out).SetLevel("debug").Str("init", "run")
	l.Debug().Msgf("aa")
	l.Debug().Msgf("bb")

	assert.Equal(t, strings.Count(out.String(), "init"), 2)
}
