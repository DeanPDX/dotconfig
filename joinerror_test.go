package dotconfig

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorsReturnsErr(t *testing.T) {
	// Make sure calling Errors on an error just returns that in collection
	err := errors.New("test error")
	errs := Errors(err)
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error. Got %v", len(errs))
	}
}

func TestErrorsReturnsNil(t *testing.T) {
	// Make sure calling Errors on an error just returns that in collection
	errs := Errors(nil)
	if errs != nil {
		t.Fatal("Expected nil slice")
	}
}

type empty struct{}

type required struct {
	MyInt int `env:"MY_INT,required"`
}

type doubleRequired struct {
	MyInt    int `env:"MY_INT,required"`
	MySecond int `env:"MY_SECOND, required"`
}

func TestErrorsStringer(t *testing.T) {
	// Make sure calling Errors on an error just returns that in collection
	_, err := FromReader[empty](strings.NewReader(""))
	if err != nil {
		t.Fatal("Expected nil slice")
	}
	// Single error should return in common error format
	_, err = FromReader[required](strings.NewReader(""))
	want := "key not present in ENV: MY_INT"
	if err.Error() != want {
		t.Fatalf("Expected %v. Got %v.", want, err.Error())
	}
	_, err = FromReader[doubleRequired](strings.NewReader(""))
	want = `multiple errors:
- key not present in ENV: MY_INT
- key not present in ENV: MY_SECOND`
	if err.Error() != want {
		t.Fatalf("Expected %v. Got %v.", want, err.Error())
	}
	errs := joinError{}
	if errs.Error() != "" {
		t.Fatalf("Expected empty string. Got: %v", errs.Error())
	}
}
