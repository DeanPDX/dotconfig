// Package dotconfig implements loading/parsing of .env
// files into configs structs.
package dotconfig

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func readEnvFile() {
	envFile, err := os.ReadFile(".env")
	// We don't really need to worry about handling errors here. If something
	// happens and there's no file that probably means we are in a prod-like environment
	// and everything will be set via actual environment variables.
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(envFile))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Empty line or comments, nothing to do. Otherwise, if it doesn't have "='" we don't have a valid line.
		if len(line) == 0 || strings.HasPrefix(line, "#") || !strings.Contains(line, "='") {
			continue
		}
		// Turn a line into key/value pair. Example line:
		// APP_DSN='postgres://username:password123=@localhost:5432/database_name'
		key := line[0:strings.Index(line, "='")]
		value := line[len(key)+2:]
		// Trim closing single quote
		value = strings.TrimSuffix(value, "'")
		// Turn \n into newlines
		value = strings.ReplaceAll(value, `\n`, "\n")
		os.Setenv(key, value)
	}
}

type DecodeOption int

const (
	ReturnFileErrors     DecodeOption = iota // Return file access errors
	ReturnDecodingErrors                     // Return encoding errors
)

type options struct {
	ReturnFileErrors     bool
	ReturnDecodingErrors bool
}

func optsFromVariadic(opts []DecodeOption) options {
	v := options{}
	for _, opt := range opts {
		switch opt {
		case ReturnFileErrors:
			v.ReturnFileErrors = true
		case ReturnDecodingErrors:
			v.ReturnDecodingErrors = true
		}
	}
	return v
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
func FromFileName[T any](name string, opts ...DecodeOption) (T, error) {
	file, err := os.Open(name)
	ops := optsFromVariadic(opts)
	if err != nil {
		var config T
		if ops.ReturnFileErrors {
			return config, err
		} else {
			return config, nil
		}
	}
	defer file.Close()
	return FromReader[T](file)
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
func FromReader[T any](r io.Reader, opts ...DecodeOption) (T, error) {
	// First, parse all values in our reader and os.Setenv them.
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Empty line or comments, nothing to do. Otherwise, if it doesn't have "='" we don't have a valid line.
		if len(line) == 0 || strings.HasPrefix(line, "#") || !strings.Contains(line, "='") {
			continue
		}
		// Turn a line into key/value pair. Example line:
		// STRIPE_SECRET_KEY='sk_test_asDF!'
		key := line[0:strings.Index(line, "='")]
		value := line[len(key)+2:]
		// Trim closing single quote
		value = strings.TrimSuffix(value, "'")
		// Turn \n into newlines
		value = strings.ReplaceAll(value, `\n`, "\n")
		os.Setenv(key, value)
	}
	// Next, populate config file based on struct tags and return populated config
	return fromEnv[T]()
}

func fromEnv[T any](opts ...DecodeOption) (T, error) {
	var config T
	// Reflect into our config
	ct := reflect.TypeOf(config)
	if ct.Kind() != reflect.Struct {
		return config, errors.New("only structs are supported")
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
			continue
		}
		envValue := os.Getenv(envKey)
		// No value
		if strings.TrimSpace(envValue) == "" {
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
			return config, fmt.Errorf("unsupported field type: %v", fieldType.Type.Name())
		}
	}
	return config, nil
}
