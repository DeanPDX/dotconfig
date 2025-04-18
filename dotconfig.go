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

type DecodeOption int

const (
	ReturnFileIOErrors DecodeOption = iota // Return file IO errors
	EnforceStructTags                      // Make sure all fields in config struct have `env` struct tags
	AllowWhitespace                        // Allow leading/trailing whitespace in string values
)

type options struct {
	ReturnFileIOErrors bool
	EnforceStructTags  bool
	AllowWhitespace    bool
}

func optsFromVariadic(opts []DecodeOption) options {
	v := options{}
	for _, opt := range opts {
		switch opt {
		case ReturnFileIOErrors:
			v.ReturnFileIOErrors = true
		case EnforceStructTags:
			v.EnforceStructTags = true
		case AllowWhitespace:
			v.AllowWhitespace = true
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
//	conf, err := dotconfig.FromFileName[myconfig](".env", dotconfig.ReturnFileIOErrors)
//
// See [FromReader] for supported types and expected file format. And
// if you want to control your own file access or read from something
// other than a file, you can call [FromReader] directly with an [io.Reader].
func FromFileName[T any](name string, opts ...DecodeOption) (T, error) {
	file, err := os.Open(name)
	if err != nil {
		ops := optsFromVariadic(opts)
		// Our consumer wants to just stop on file errors. This is unusual
		// but it's the case where they always want to ensure an .env file
		// exists and is successfully read.
		if ops.ReturnFileIOErrors {
			var config T
			return config, err
		} else {
			// No env file but we will still extract our config from the env
			// variables.
			return fromEnv[T](ops)
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
//	# You also don't need single/double quotes at all if you prefer
//	DATA_SOURCE_NAME=postgres://username:password@localhost:5432/database_name
//	DOUBLE_QUOTES="sk_test_asDF!"
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
		if len(line) == 0 || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}

		// Turn a line into key/value pair. Example lines:
		// STRIPE_SECRET_KEY='sk_test_asDF!'
		// STRIPE_SECRET_KEY=sk_test_asDF!
		// STRIPE_SECRET_KEY="sk_test_asDF!"
		key := line[0:strings.Index(line, "=")]
		value := line[len(key)+1:]

		// If there is a inline comment, so a space and then a #, exclude the comment.
		if strings.Contains(value, " #") {
			value = value[0:strings.Index(value, " #")]
		}

		// Determine if our string is single quoted, double quoted, or just raw value.
		if strings.HasPrefix(value, "'") {
			// Trim closing single quote
			value = strings.TrimSuffix(value, "'")
			// And trim starting single quote
			value = strings.TrimPrefix(value, "'")
		} else if strings.HasPrefix(value, `"`) {
			// Trim closing double quote
			value = strings.TrimSuffix(value, `"`)
			// And trim starting double quote
			value = strings.TrimPrefix(value, `"`)
		}
		// Turn \n into newlines
		value = strings.ReplaceAll(value, `\n`, "\n")
		// Finally, set our env variable.
		os.Setenv(key, value)
	}
	// Next, populate config file based on struct tags and return populated config
	return fromEnv[T](optsFromVariadic(opts))
}

var (
	ErrConfigMustBeStruct   = errors.New("config must be struct")
	ErrMissingStructTag     = errors.New("missing struct tag on field")
	ErrMissingEnvVar        = errors.New("key not present in ENV")
	ErrMissingRequiredField = errors.New("field must have non-zero value")
	ErrUnsupportedFieldType = errors.New("unsupported field type")
)

func fromEnv[T any](opts options) (T, error) {
	var config T
	errs := joinError{}
	// Reflect into our config
	ct := reflect.TypeOf(config)
	// If config is not a struct, that's a hard stop.
	if ct.Kind() != reflect.Struct {
		return config, ErrConfigMustBeStruct
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
		// Get the env struct tag
		envTag := fieldType.Tag.Get("env")
		// No struct tag
		if envTag == "" {
			// By default we just assume the consumers of this library have
			// a mixture of fields with env struct tags and some they want
			// this library to ignore. But consumers can opt in to no struct
			// tag = error with config setting.
			if opts.EnforceStructTags {
				errs.Add(fmt.Errorf("%w: %v", ErrMissingStructTag, fieldType.Name))
			}
			continue
		}
		// Parse env tag into environment variable key and options
		envKey, tagOpts := parseTag(envTag)
		envValue, keyExists := os.LookupEnv(envKey)
		// Missing env var
		if !keyExists {
			// Check to see if we have a default value
			defaultVal := fieldType.Tag.Get("default")
			if defaultVal != "" {
				envValue = defaultVal
			} else if tagOpts.Contains("optional") {
				// Optional so skip missing error
				continue
			} else {
				errs.Add(fmt.Errorf("%w: %v", ErrMissingEnvVar, envKey))
				continue
			}
		}
		// If the consumer hasn't explicitely allowed whitespace, we trim it by default
		if !opts.AllowWhitespace {
			envValue = strings.TrimSpace(envValue)
		}
		// Empty value
		if envValue == "" {
			// If required option is set, this is an error
			if tagOpts.Contains("required") {
				errs.Add(fmt.Errorf("%w: %v", ErrMissingRequiredField, envKey))
			}
			// Otherwise zero-values are fine
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
			errs.Add(fmt.Errorf("%w: %v", ErrUnsupportedFieldType, fieldType.Type.Name()))
		}
	}
	if errs.HasErrors() {
		return config, errs
	}
	return config, nil

}
