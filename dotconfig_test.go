package dotconfig_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/DeanPDX/dotconfig"
)

type sampleConfig struct {
	StripeSecret string `env:"STRIPE_SECRET"`
	IsDevEnv     bool   `env:"IS_DEV"`
	WelcomeEmail string `env:"WELCOME_EMAIL"`
}

const welcomeEmail = `Hello,

Welcome to the app!

-The Team`

func TestFromReaderNewlines(t *testing.T) {
	reader := strings.NewReader(`#just testing
# Stripe secret key
STRIPE_SECRET='sk_test_asDF!'
# Going to leave a file lines blank


IS_DEV='true'
WELCOME_EMAIL='Hello,\n\nWelcome to the app!\n\n-The Team'`)

	config, err := dotconfig.FromReader[sampleConfig](reader)
	if err != nil {
		t.Fatalf("Didn't expect error. Got %v.", err)
	}
	expected := sampleConfig{
		StripeSecret: "sk_test_asDF!",
		IsDevEnv:     true,
		WelcomeEmail: welcomeEmail,
	}
	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v", expected, config)
	}
}

type moreAdvancedConfig struct {
	MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
	APIVersion         float64 `env:"API_VERSION"`
	IsDev              bool    `env:"IS_DEV"`
	LogErrors          bool    `env:"LOG_ERRORS"`
	notExported        string  `env:"NOT_EXPORTED"`
}

// Valid bool values are:
// 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
const moreAdvancedEnv = `
MAX_BYTES_PER_REQUEST='1024'
API_VERSION=1.19
IS_DEV="1"
LOG_ERRORS='t'
NOT_EXPORTED='yikes!'
`

func TestFromReaderDecoding(t *testing.T) {
	reader := strings.NewReader(moreAdvancedEnv)
	config, err := dotconfig.FromReader[moreAdvancedConfig](reader)
	if err != nil {
		t.Fatalf("Didn't expect error. Got %v.", err)
	}
	expected := moreAdvancedConfig{
		MaxBytesPerRequest: 1024,
		APIVersion:         1.19,
		IsDev:              true,
		LogErrors:          true,
		notExported:        "",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v", expected, config)
	}
}

func TestFromFileNameErrs(t *testing.T) {
	type myconfig struct{}
	_, err := dotconfig.FromFileName[myconfig]("DOESN'T EXIST!!!", dotconfig.ReturnFileIOErrors)
	// Make sure we get a doesn't exist error.
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Expected error: %v. Got: %v.", os.ErrNotExist, err)
	}
}

// This is how most things will **actually** work in production type
// environments. The file won't exist, file IO errors will be ignored,
// and values will come from environment variables.
func TestFromFileNameNoFile(t *testing.T) {
	os.Setenv("MAX_BYTES_PER_REQUEST", "1024")
	os.Setenv("API_VERSION", "1.19")
	os.Setenv("IS_DEV", "t")
	os.Setenv("LOG_ERRORS", "1")
	os.Setenv("NOT_EXPORTED", "some value")
	config, err := dotconfig.FromFileName[moreAdvancedConfig]("doesn't exist!")
	if err != nil {
		fmt.Printf("Didn't expect error. Got %v.", err)
	}

	expected := moreAdvancedConfig{
		MaxBytesPerRequest: 1024,
		APIVersion:         1.19,
		IsDev:              true,
		LogErrors:          true,
		notExported:        "",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v", expected, config)
	}
}

const errTestStr = `MAX_BYTES_PER_REQUEST='1024'
# You can do single quotes or not.
API_VERSION=1.19
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
# Also, double quotes are fine if you prefer them:
IS_DEV="1"
STRIPE_SECRET='sk_test_insertkeyhere'
# Right now supporting newlines via "\n" in strings:
WELCOME_MESSAGE='Hello,\nWelcome to the app!\n-The App Dev Team'
UNSUPPORTED_TYPE=asdf`

func TestMultiError(t *testing.T) {
	type MultiErrorConfig struct {
		MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
		APIVersion         float64 `env:"API_VERSION"`
		IsDev              bool    `env:"IS_DEV"`
		StripeSecret       string  `env:"STRIPE_SECRET"`
		MissingKey         string  `env:"MISSING_KEY"`
		WelcomeMessage     string  `env:"WELCOME_MESSAGE"`
		EmptyStructTag     string
		UnsupportedType    complex64 `env:"UNSUPPORTED_TYPE"`
	}

	_, err := dotconfig.FromReader[MultiErrorConfig](strings.NewReader(errTestStr), dotconfig.EnforceStructTags)
	knownErrors := 0
	if err != nil {
		// Get error slice from err
		errs := dotconfig.Errors(err)
		for _, err := range errs {
			// Handle various error types however you want
			switch {
			case errors.Is(dotconfig.ErrMissingEnvVar, errors.Unwrap(err)):
				// Handle missing environment variable
				knownErrors++
				fmt.Printf("Error: %v\n", err)
			case errors.Is(dotconfig.ErrMissingStructTag, errors.Unwrap(err)):
				// Handle missing struct tag
				knownErrors++
				fmt.Printf("Error: %v\n", err)
			case errors.Is(dotconfig.ErrUnsupportedFieldType, errors.Unwrap(err)):
				// Handle unsupported field
				knownErrors++
				fmt.Printf("Error: %v\n", err)
			}
		}
		if knownErrors != 3 {
			t.Errorf("Expecting 3 errors.")
		}
	}
}

func TestSingleError(t *testing.T) {
	type AppConfig struct {
		ForgotToAddStructTag string
	}
	_, err := dotconfig.FromFileName[AppConfig](".env", dotconfig.EnforceStructTags)
	// Just make sure we get exactly 1 error.
	errs := dotconfig.Errors(err)
	if len(errs) != 1 {
		t.Errorf("Expecting exactly 1 error")
	}
}

type AppConfig struct {
	MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
	APIVersion         float64 `env:"API_VERSION"`
	IsDev              bool    `env:"IS_DEV"`
	StripeSecret       string  `env:"STRIPE_SECRET"`
	WelcomeMessage     string  `env:"WELCOME_MESSAGE"`
}

const appConfigSample = `
MAX_BYTES_PER_REQUEST="1024"
API_VERSION=1.19
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
IS_DEV='1'
STRIPE_SECRET='sk_test_insertkeyhere'
# Right now supporting newlines via "\n" in strings:
WELCOME_MESSAGE='Hello,\nWelcome to the app!\n-The App Dev Team'`

func ExampleFromReader() {
	config, err := dotconfig.FromReader[AppConfig](strings.NewReader(appConfigSample))
	if err != nil {
		fmt.Printf("Didn't expect error. Got %v.", err)
	}
	// Don't do this in the real world, as your config will
	// have secrets from a secret manager and you don't want
	// to print them to the console.
	fmt.Printf("App config loaded.\nMax Bytes: %v. Version: %v. Dev? %v. Stripe Secret: %v.\nWelcome Message:\n%v",
		config.MaxBytesPerRequest, config.APIVersion, config.IsDev, config.StripeSecret, config.WelcomeMessage)
	// Output:
	// App config loaded.
	// Max Bytes: 1024. Version: 1.19. Dev? true. Stripe Secret: sk_test_insertkeyhere.
	// Welcome Message:
	// Hello,
	// Welcome to the app!
	// -The App Dev Team
}

type ConfigWithErrors struct {
	StripeSecret   string     `env:"SHOULD_BE_MISSING"`
	Complex        complex128 `env:"COMPLEX"`
	WelcomeMessage string
}

func ExampleErrors() {
	r := strings.NewReader(`COMPLEX=asdf`)
	_, err := dotconfig.FromReader[ConfigWithErrors](r, dotconfig.EnforceStructTags)
	if err != nil {
		// Get error slice from err
		errs := dotconfig.Errors(err)
		for _, err := range errs {
			// Handle various error types however you want
			switch {
			case errors.Is(dotconfig.ErrMissingEnvVar, errors.Unwrap(err)):
				// Handle missing environment variable
				fmt.Printf("Missing env variable: %v\n", err)
			case errors.Is(dotconfig.ErrMissingStructTag, errors.Unwrap(err)):
				// Handle missing struct tag
				fmt.Printf("Missing struct tag: %v\n", err)
			case errors.Is(dotconfig.ErrUnsupportedFieldType, errors.Unwrap(err)):
				// Handle unsupported field
				fmt.Printf("Unsupported type: %v\n", err)
			}
		}
	}
	// Output:
	// Missing env variable: value not present in env: SHOULD_BE_MISSING
	// Unsupported type: unsupported field type: complex128
	// Missing struct tag: missing struct tag on field: WelcomeMessage
}
