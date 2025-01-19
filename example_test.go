package dotconfig_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DeanPDX/dotconfig"
)

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
	RequiredField  string `env:"REQUIRED_FIELD,required"` // Can't be zero-value
}

const exampleErrorsEnv = `
COMPLEX=asdf
REQUIRED_FIELD="" # Will cause error because zero-value`

func ExampleErrors() {
	r := strings.NewReader(exampleErrorsEnv)
	_, err := dotconfig.FromReader[ConfigWithErrors](r, dotconfig.EnforceStructTags)
	if err != nil {
		// Get error slice from err
		errs := dotconfig.Errors(err)
		for _, err := range errs {
			// Handle various error types however you want
			switch {
			case errors.Is(errors.Unwrap(err), dotconfig.ErrMissingEnvVar):
				// Handle missing environment variable
				fmt.Printf("Missing env variable: %v\n", err)
			case errors.Is(errors.Unwrap(err), dotconfig.ErrMissingStructTag):
				// Handle missing struct tag
				fmt.Printf("Missing struct tag: %v\n", err)
			case errors.Is(errors.Unwrap(err), dotconfig.ErrUnsupportedFieldType):
				// Handle unsupported field
				fmt.Printf("Unsupported type: %v\n", err)
			case errors.Is(errors.Unwrap(err), dotconfig.ErrMissingRequiredField):
				// Handle required field
				fmt.Printf("Required field can't be zero value: %v\n", err)
			}
		}
	}
	// Output:
	// Missing env variable: key not present in ENV: SHOULD_BE_MISSING
	// Unsupported type: unsupported field type: complex128
	// Missing struct tag: missing struct tag on field: WelcomeMessage
	// Required field can't be zero value: field must have non-zero value: REQUIRED_FIELD
}

type ConfigWithDefaults struct {
	MaxBytesPerRequest int     `env:"MAX_BYTES" default:"2048"`
	IsDev              bool    `env:"DEVELOPMENT,optional"` // will default to zero value (false)
	WelcomeMessage     string  `env:"APP_HELLO" default:"Hey!"`
	AppVersion         float64 `env:"APP_VERSION" default:"1.0"`
}

func Example_defaultValues() {
	r := strings.NewReader(`APP_VERSION=2.38`)
	conf, err := dotconfig.FromReader[ConfigWithDefaults](r, dotconfig.EnforceStructTags)
	if err != nil {
		fmt.Printf("Didn't expect error. Got %v.", err)
	}
	fmt.Println("App config loaded")
	fmt.Println("Max bytes:", conf.MaxBytesPerRequest)   // 2048 from default tag
	fmt.Println("Is dev?", conf.IsDev)                   // False because optional so zero value
	fmt.Println("Welcome message:", conf.WelcomeMessage) // "Hey!" from default tag
	fmt.Println("App version:", conf.AppVersion)         // 2.38 because ENV value overrides default
	// Output:
	// App config loaded
	// Max bytes: 2048
	// Is dev? false
	// Welcome message: Hey!
	// App version: 2.38
}
