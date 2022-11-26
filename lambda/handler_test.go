package lambda

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func testa()                                  {}
func testb() error                            { return nil }
func testc(a int) error                       { return nil }
func testd() (int, error)                     { return 0, nil }
func teste(a int) (int, error)                { return 0, nil }
func testf(context.Context) error             { return nil }
func testg(context.Context, int) error        { return nil }
func testh(context.Context) (int, error)      { return 0, nil }
func testi(context.Context, int) (int, error) { return 0, nil }

func Test_Validate(t *testing.T) {
	fn := []any{testa, testb, testc, testd, teste, testf, testg, testh, testi}
	for _, f := range fn {
		_, err := validateArguments(reflect.TypeOf(f))
		assert.NoError(t, err)
	}
}
