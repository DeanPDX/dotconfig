# Dotconfig [![Go Reference](https://pkg.go.dev/badge/github.com/DeanPDX/dotconfig.svg)](https://pkg.go.dev/github.com/DeanPDX/dotconfig)
This package aims to simplify configuration from environment variables. In local development, we can supply a `.env` file with key/value pairs. When deployed, values come from a secret manager. This is similar to [joho/godotenv](https://github.com/joho/godotenv) but the aim here is to not only read the `.env` file but use reflection to produce a config struct.

## Usage
Create a `.env` file in your current working directory with the following contents:

```shell
MAX_BYTES_PER_REQUEST='1024'
# You can do single quotes or not.
API_VERSION=1.19
# All of these are valie for booleans:
# 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
IS_DEV='1'
STRIPE_SECRET='sk_test_insertkeyhere'
# You can do single quotes or not.
ANOTHER_SECRET=withoutsinglequotes
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

func main() {

	type AppConfig struct {
		MaxBytesPerRequest int     `env:"MAX_BYTES_PER_REQUEST"`
		APIVersion         float64 `env:"API_VERSION"`
		IsDev              bool    `env:"IS_DEV"`
		StripeSecret       string  `env:"STRIPE_SECRET"`
		AnotherSecret      string  `env:"ANOTHER_SECRET"`
		MissingKey         string  `env:"MISSING_KEY"`
		EnvEmpty           string  `env:""`
		WelcomeMessage     string  `env:"WELCOME_MESSAGE"`
	}

	config, errC := dotconfig.FromFileName[AppConfig](".env")
	if errC.HasErrors() {
		for _, err := range errC.Errors {
			fmt.Println(err) // handle errors. Decide yourself what to do.
		}
		// This will return:
		// envKey:MISSING_KEY is not present
		// fieldVal:{EnvEmpty  string env:"" 72 [6] false} is empty

		fmt.Println(errC.Error()) // Prints all erros as string.
	}

	fmt.Println("RESULT:###########")

	fmt.Println(config.MaxBytesPerRequest)
	fmt.Println(config.APIVersion)
	fmt.Println(config.IsDev)
	fmt.Println(config.StripeSecret)
	fmt.Println(config.AnotherSecret)
	fmt.Println(config.MissingKey)
	fmt.Println(config.EnvEmpty)
	fmt.Println(config.WelcomeMessage)

}
```

So for local dev we can use this `.env` file. But when you deploy your app, you set these values from environment variables / secret managers. Your app that consumes this config struct doesn't have to concern itself with where the values came from.

If your key value pairs are coming from a source other than a file, or you want to control file IO yourself, you can call `FromReader` instead and pass in a `io.Reader`. There is [a runnable example of that in the godoc](https://pkg.go.dev/github.com/DeanPDX/dotconfig#example-FromReader).

## Error Handling
Dotconfig returns an Error collection. And you can handle every entry as you need. This is useful, because sometimes it is OK to have a PORT empty or whatnot.


## Contributing
Contributions are always welcome. This is still in the early stages and is mostly for internal use at the moment. Have a new idea or find a bug? Submit a pull request or create an issue!