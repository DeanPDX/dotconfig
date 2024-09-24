// Package dotconfig implements loading/parsing of .env
// files into configs structs.
package dotconfig

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type ErrorCollection struct {
	Errors []error
}

// Add appends a new error to the collection
func (ec *ErrorCollection) Add(err error) {
	if err != nil {
		ec.Errors = append(ec.Errors, err)
	}
}

// HasErrors returns true if the collection contains any errors
func (ec *ErrorCollection) HasErrors() bool {
	return len(ec.Errors) > 0
}

// Error implements the error interface
func (ec *ErrorCollection) Error() string {
	if !ec.HasErrors() {
		return ""
	}
	errorStrings := make([]string, len(ec.Errors))
	for i, err := range ec.Errors {
		errorStrings[i] = err.Error()
	}
	return fmt.Sprintf("Multiple errors occurred:\n- %s", strings.Join(errorStrings, "\n- "))
}

// FromFileName will call [os.Open] on the supplied name and will
// then call [FromReader]. By default this will ignore file access
// errors. This is usually desired behavior because in live
// environments, config will come from a secret manager and [os.Open]
// will fail. If you *want* to return file errors, use opts:
//
//	type myconfig struct{/*...*/}
//	conf, err := dotconfig.FromFileName[myconfig](".env", dotconfig.ReturnFileErrors)
//
// See [FromReader] for supported types and expected file format. And
// if you want to control your own file access or read from something
// other than a file, you can call [FromReader] directly with an [io.Reader].
func FromFileName[T any](name string) (T, ErrorCollection) {

	ec := &ErrorCollection{}

	file, err := os.Open(name)
	if err != nil {
		var config T
		ec.Add(err)
		return config, *ec
	}

	defer file.Close()
	return FromReader[T](file, ec), *ec
}

// FromReader will read from r and call os.Setenv to set
// environment variables based on key value pairs in r.
//
// Expected format for the pairs is:
//
//	KEY='value enclosed in single quotes'
//	# Comments are fine as are blank lines
//
//	STRIPE_SECRET_KEY='sk_test_asDF!'
//	MULTI_LINE='line1\nline2\nline3'
//
// Currently newlines are supported as "\n" in string values.
// In the future might look in to more advanced escaping, etc.
// but this suits our needs for the time being.
func FromReader[T any](r io.Reader, ec *ErrorCollection) T {
	// First, parse all values in our reader and os.Setenv them.
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Empty line or comments, nothing to do. Otherwise, if it doesn't have "='" we don't have a valid line.
		if len(line) == 0 || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		// Turn a line into key/value pair. Example line:
		// STRIPE_SECRET_KEY='sk_test_asDF!'

		key := line[0:strings.Index(line, "=")]
		value := line[len(key)+1:]
		// Trim closing single quote
		value = strings.TrimSuffix(value, "'")
		value = strings.TrimPrefix(value, "'")
		// Turn \n into newlines
		value = strings.ReplaceAll(value, `\n`, "\n")
		os.Setenv(key, value)
	}
	// Next, populate config file based on struct tags and return populated config
	return fromEnv[T](ec)
}

func fromEnv[T any](ec *ErrorCollection) T {
	var config T
	// Reflect into our config
	ct := reflect.TypeOf(config)
	if ct.Kind() != reflect.Struct {
		ec.Add(errors.New("config is no struct"))
		return config
	}
	cv := reflect.ValueOf(&config).Elem()
	// Enumerate fields and grab values via os.Getenv, converting as needed.
	for i := 0; i < ct.NumField(); i++ {
		fieldVal := cv.Field(i)
		// Ensure we can set field
		if !fieldVal.CanSet() {
			continue
		}
		fieldType := ct.Field(i)
		envKey := fieldType.Tag.Get("env")

		// No struct tag
		if envKey == "" {
			ec.Add(errors.New(fmt.Sprint("fieldVal: ", fieldType, " is empty")))
			continue
		}

		envValue, valueIsThere := os.LookupEnv(envKey)
		if !valueIsThere {
			ec.Add(errors.New(fmt.Sprint("envKey: ", envKey, " is not present")))
		}

		// No value is OK now.
		if strings.TrimSpace(envValue) == "" {
			fieldVal.SetString("")
			continue
		}

		// Based on type, parse and set values. This borrows from encoding/json:
		// https://cs.opensource.google/go/go/+/refs/tags/go1.23.1:src/encoding/json/decode.go;l=990
		switch fieldType.Type.Kind() {
		case reflect.Bool:
			val, _ := strconv.ParseBool(envValue)
			fieldVal.SetBool(val)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val, _ := strconv.ParseInt(envValue, 10, 64)
			fieldVal.SetInt(val)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			val, _ := strconv.ParseUint(envValue, 10, 64)
			fieldVal.SetUint(val)
		case reflect.Float32, reflect.Float64:
			val, _ := strconv.ParseFloat(envValue, fieldType.Type.Bits())
			fieldVal.SetFloat(val)
		case reflect.String:
			fieldVal.SetString(envValue)
		default:
			ec.Add(errors.New(fmt.Sprint("unsupported field type: ", fieldType.Type.Name())))
			return config
		}
	}
	return config
}
