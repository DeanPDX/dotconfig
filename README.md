# Dotconfig [![Go Reference](https://pkg.go.dev/badge/github.com/DeanPDX/dotconfig.svg)](https://pkg.go.dev/github.com/DeanPDX/dotconfig)
This package aims to simplify configuration from environment variables. In local development, we can supply a `.env` file with key/value pairs. When deployed, values come from a secret manager. This is similar to [joho/godotenv](https://github.com/joho/godotenv) but the aim here is to not only read the `.env` file but use reflection to produce a config struct.

## Example
Assume a file in the current working directory called `.env` with the following contents:

```shell
MAX_BYTES_PER_REQUEST='1024'
API_VERSION='1.19'
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
IS_DEV='1'
STRIPE_SECRET='sk_test_insertkeyhere'
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
                fmt.Printf("Didn't expect error. Got %v.", err)
        }
        // Config is ready to use.
        // Float/bool/int values will be correctly parsed.
        // When deploying to the cloud, make sure your env
        // vars are being passed to your container via secret
        // manager.
}
```

If your key value pairs are coming from a source other than a local file, or you want to control file IO yourself, you can call `FromReader` instead and pass in a io.Reader. There is [an example of that in the godoc](https://pkg.go.dev/github.com/DeanPDX/dotconfig#example-FromReader).

## Error Handling
By default, file IO errors in `dotconfig.FromFileName` won't produce an error. This is because when you are running in the cloud with a secret manager, not finding a `.env` file is the happy path. If you want to return errors from `os.Open` you can do so with an option:

```go
config, err := dotconfig.FromFileName[AppConfig](".env", dotconfig.ReturnFileErrors)
```

## Contributing
Contributions are always welcome. This is still in the early stages and is mostly for internal use at the moment. Have a new idea or find a bug? Submit a pull request or create an issue!