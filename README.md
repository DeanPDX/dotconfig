# Dotconfig [![Go Reference](https://pkg.go.dev/badge/github.com/DeanPDX/dotconfig.svg)](https://pkg.go.dev/github.com/DeanPDX/dotconfig)
This package aims to simplify configuration from environment variables. In local development, we can supply a `.env` file with key/value pairs. When deployed, values come from a secret manager. This is similar to [joho/godotenv](https://github.com/joho/godotenv) but the aim here is to not only read the `.env` file but use reflection to produce a config struct.

## Usage
Create a `.env` file in your current working directory with the following contents:

```shell
MAX_BYTES_PER_REQUEST='1024'
# Double quotes are fine
API_VERSION="1.19"
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
IS_DEV='1'
# Raw values with no quotes are also fine
STRIPE_SECRET=sk_test_insertkeyhere
# Right now supporting newlines via "\n" in strings:
WELCOME_MESSAGE='Hello,\nWelcome to the app!\n-The App Dev Team'
```

You can read from this file and initialize your config with values with the following code:

```go
package main

import (
	"fmt"

	"github.com/DeanPDX/dotconfig"
)

// Our AppConfig with env struct tags:
type AppConfig struct {
	MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
	APIVersion         float64 `env:"API_VERSION"`
	IsDev              bool    `env:"IS_DEV"`
	StripeSecret       string  `env:"STRIPE_SECRET"`
	WelcomeMessage     string  `env:"WELCOME_MESSAGE"`
}

func Main() {
	config, err := dotconfig.FromFileName[AppConfig](".env")
	if err != nil {
		fmt.Printf("Error: %v.", err)
	}
	// Config is ready to use. Don't print to console in a real 
	// app. But for the purposes of testing:
	fmt.Println(config)
}
```

So for local dev we can use this `.env` file. But when you deploy your app, you set these values from environment variables / secret managers. Your app that consumes this config struct doesn't have to concern itself with where the values came from.

If your key value pairs are coming from a source other than a file, or you want to control file IO yourself, you can call `FromReader` instead and pass in a `io.Reader`. There is [a runnable example of that in the godoc](https://pkg.go.dev/github.com/DeanPDX/dotconfig#example-FromReader).

## Error Handling

By default, file IO errors in `dotconfig.FromFileName` won't produce an error. This is because when you are running in the cloud with a secret manager, not finding a `.env` file is the happy path. If you want to return errors from `os.Open` you can do so with an option:

```go
config, err := dotconfig.FromFileName[AppConfig](".env", dotconfig.ReturnFileErrors)
```

By default, if your struct contains fields that don't have an `env:"MY_ENV"` tag, we assume you want us to ignore those fields. If you want missing `env` tags to produce errors, use the `dotconfig.EnforceStructTags` option:

```
config, err := dotconfig.FromFileName[AppConfig](".env", dotconfig.EnforceStructTags)
```

`dotconfig.FromFileName` and `dotconfig.FromReader` both return multiple wrapped errors. If you want to print all errors to the console you can do that:

```go
type AppConfig struct {
	ForgotToAddStructTag string
	UnsupportedType   	 complex64 `env:"UNSUPPORTED_TYPE"`
}
config, err := dotconfig.FromFileName[AppConfig](".env", dotconfig.EnforceStructTags)
if err != nil {
	fmt.Printf("Error %v.", err)
}
// Output:
// Error: multiple errors:
//  - missing struct tag on field: ForgotToAddStructTag
//  - unsupported field type: complex64
```

Sometimes you want more fine-grained control of error handling (because certain states you can recover from). If you want to handle each error type, you can use `dotconfig.Errors` in conjunction with `errors.Unwrap` and `errors.Is`. Here's an example where each error type is being handled:

```go
type MyConfig struct {}
_, err := dotconfig.FromFileName[MyConfig](".env", dotconfig.EnforceStructTags)
if err != nil {
	// Get error slice from err
	errs := dotconfig.Errors(err)
	for _, err := range errs {
		// Handle various error types however you want
		switch {
		case errors.Is(dotconfig.ErrMissingEnvVar, errors.Unwrap(err)):
			// Handle missing environment variable
		case errors.Is(dotconfig.ErrMissingStructTag, errors.Unwrap(err)):
			// Handle missing struct tag
		case errors.Is(dotconfig.ErrUnsupportedFieldType, errors.Unwrap(err)):
			// Handle unsupported field type
		}
	}
}
```

## Contributing
Contributions are always welcome. This is still in the early stages and is mostly for internal use at the moment. Have a new idea or find a bug? Submit a pull request or create an issue!
