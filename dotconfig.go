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
	ReturnFileIOErrors  DecodeOption = iota // Return file IO errors
	EnforceStructTags                       // Make sure all fields in config struct have `env` struct tags
	AllowWhitespace                         // Allow leading/trailing whitespace in string values
	SkipNewlineDecoding                     // Don't turn "\n" into newlines
	SkipCommentStrip                        // Don't exclude the comment from value that ends with #some_inline_comment
)

type options struct {
	ReturnFileIOErrors  bool
	EnforceStructTags   bool
	AllowWhitespace     bool
	SkipNewlineDecoding bool
	SkipCommentStrip    bool
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
		case SkipNewlineDecoding:
			v.SkipNewlineDecoding = true
		case SkipCommentStrip:
			v.SkipCommentStrip = true
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
	// Get options struct from variadic input
	decodedOpts := optsFromVariadic(opts)

	type pToken int
	const (
		KEY pToken = iota
		VALUE
	)

	type pState int
	const (
		NORMAL pState = iota
		QSTRING
		DQSTRING
		COMMENT
	)

	rr := bufio.NewReader(r)

	p := 0
	var pt pToken
	var ps pState

	var key, value strings.Builder

	eat := func(ch rune) {
		if pt == KEY {
			key.WriteRune(ch)
		} else if pt == VALUE {
			value.WriteRune(ch)
		} else {
			panic("newer")
		}
	}

	reset := func(ppt pToken, pps pState) {
		pt = ppt
		ps = pps
		key.Reset()
		value.Reset()
	}

	flush := func() error {
		if key.Len() > 0 {
			keyStr := key.String()
			if !decodedOpts.SkipNewlineDecoding {
				keyStr = strings.ReplaceAll(keyStr, "\\n", "\n")
			}

			valueStr := value.String()
			if len(valueStr) > 0 {
				if !decodedOpts.SkipNewlineDecoding {
					valueStr = strings.ReplaceAll(valueStr, "\\n", "\n")
				}
			}

			keyStr = strings.TrimSpace(keyStr)
			if len(keyStr) > 0 {
				return os.Setenv(keyStr, valueStr)
			}
		}
		return nil
	}

	reset(KEY, NORMAL)
	for ; ; p++ {
		ch, _, err := rr.ReadRune()
		if err != nil {
			if err == io.EOF {
				if ps == QSTRING || ps == DQSTRING {
					var defaultT T
					return defaultT, fmt.Errorf("invalid key-value sequence at index %d", p)
				}

				if pt == VALUE {
					_ = flush()
				}
				return fromEnv[T](decodedOpts)
			}

			var defaultT T
			return defaultT, fmt.Errorf("invalid unicode sequence at index %d: %w", p, err)
		}

		if ch == '#' {
			if ps == DQSTRING || ps == QSTRING {
				eat(ch)
			} else if pt == KEY {
				ps = COMMENT
			} else if pt == VALUE {
				if decodedOpts.SkipCommentStrip {
					eat(ch)
				} else {
					_ = flush()
					reset(KEY, COMMENT)
				}
			} else {
				break
			}
			continue
		}

		if ch == '\n' {
			if ps == DQSTRING || ps == QSTRING {
				eat(ch)
			} else if ps == COMMENT {
				reset(KEY, NORMAL)
			} else if pt == KEY {
				// empty line
			} else if pt == VALUE {
				_ = flush()
				reset(KEY, NORMAL)
			} else {
				break
			}

			continue
		}

		if ch == '"' {
			if ps == COMMENT {
				// skip
			} else if ps == DQSTRING {
				ps = NORMAL
			} else if ps == QSTRING {
				eat(ch)
			} else if ps == NORMAL {
				ps = DQSTRING
			} else {
				break
			}
			continue
		}

		if ch == '\'' {
			if ps == COMMENT {
				// skip
			} else if ps == QSTRING {
				ps = NORMAL
			} else if ps == DQSTRING {
				eat(ch)
			} else if ps == NORMAL {
				ps = QSTRING
			} else {
				break
			}
			continue
		}

		if ch == '=' {
			if ps == COMMENT {
				// skip
			} else if ps == QSTRING || ps == DQSTRING {
				eat(ch)
			} else if pt == KEY {
				pt = VALUE
			} else if pt == VALUE {
				eat(ch)
			} else {
				break
			}
			continue
		}

		if ps != COMMENT {
			eat(ch)
		}
	}

	var defaultT T
	return defaultT, fmt.Errorf("invalid parser state at index: %d", p)
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
