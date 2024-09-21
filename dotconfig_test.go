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
API_VERSION='1.19'
IS_DEV='1'
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
	_, err := dotconfig.FromFileName[myconfig]("DOESN'T EXIST!!!", dotconfig.ReturnFileErrors)
	// Make sure we get a doesn't exist error.
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Expected error: %v. Got: %v.", os.ErrNotExist, err)
	}
}

type AppConfig struct {
	MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
	APIVersion         float64 `env:"API_VERSION"`
	IsDev              bool    `env:"IS_DEV"`
	StripeSecret       string  `env:"STRIPE_SECRET"`
}

const appConfigSample = `
MAX_BYTES_PER_REQUEST='1024'
API_VERSION='1.19'
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
IS_DEV='1'
STRIPE_SECRET='sk_test_insertkeyhere'`

func ExampleFromReader() {
	config, err := dotconfig.FromReader[AppConfig](strings.NewReader(appConfigSample))
	if err != nil {
		fmt.Printf("Didn't expect error. Got %v.", err)
	}
	// Don't do this in the real world, as your config will often
	// have secrets from a secret manager and you don't want to
	// print them to the console.
	fmt.Printf("App config loaded. Ready to serve traffic with following configuration: %#v", config)

	// Output:
	// App config loaded. Ready to serve traffic with following configuration: dotconfig_test.AppConfig{MaxBytesPerRequest:1024, APIVersion:1.19, IsDev:true, StripeSecret:"sk_test_insertkeyhere"}
}
