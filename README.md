# configurama

[![GoDoc](https://godoc.org/github.com/mkock/configurama?status.svg)](https://godoc.org/github.com/mkock/configurama)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GoReportCard](https://goreportcard.com/badge/github.com/mkock/configurama)](https://goreportcard.com/report/github.com/mkock/configurama)

Package configurama provides the ability to store app-wide configuration
parameters and extract different sections with the purpose of decoupling configuration from the packages that use it.

V1 (deprecated) allows extracting multiple parameters into predefined structs where the fields names of the structs
must match the parameter names. Hook functions can be utilized to provide pre-processing, validation etc.

V2 dispenses with the hooks and the "magical" struct assignments and instead provides simple methods for marking
individual parameters as required, having defaults and having to pass validation using regular expressions. V2 is more
explicit and doesn't use reflection, so it's recommended over V1.

## Installation

This package supports Go Modules. Simply run:

```bash
go get github.com/mkock/configurama/V2
```

## V2

### Usage

First, call `configurama.New()` to create a new config pool. It takes as its
only argument the configuration to use, of type `map[string]map[string]string`.

The outermost map represents sections in your configuration file. These are just
names, so it's up to you what you want to do with them, but common strategies are:

1. Having no sections. Just use an empty string, or a useful name, eg. "default".
2. Having a section per service, eg. "database", "cache", "stripe" etc.
3. Having a section per environment, eg. "dev", "prod" etc.

Retrieving parameters is achieved via the helper methods:

* `Pool.String(section, key string, options ...Option) (string, error)`
* `Pool.Int(section, key string, options ...Option) (int, error)`
* `Pool.Float(section, key string, options ...Option) (float64, error)`
* `Pool.Duration(section, key string, options ...Option) (time.Duration, error)`

`options` can be omitted altogether. They are helpful when you need to indicate that a parameter is
required, should be validated or if it should use a default value for unknown/empty parameters.

### Application of Options

1. Fetching an unknown parameter always returns the zero value unless the `Required` option is given.
2. If the `Required` option is given, `NoKeyError` is returned for unknown parameters.
3. If the `Validate` option is given in combination with `Default`, then the default value will also be validated.
4. The `Default` option only applies for missing/empty parameters, _not_ for failed validations or required parameters.

### Other Options

You can always call `Merge()` on an existing pool if you wish to add/overwrite
one or more configuration parameters.

`Merge()` can also be used to unset parameters:

```go
config.Merge(map[string]map[string]string{"dev": {"db.password": ""}}, Overwrite)
```

Or you can remove the parameter altogether:

```go
config.Unset("dev", "db.password")
```

Please see the included example test files for more usage examples.

### Example

```go
params := map[string]map[string]string{
    "dev": {
        "db.service": "mysql",
        "db.host": "localhost",
        "db.port": "3306",
        "db.username": "me",
        "db.password": "secret",
        "db.db": "movies_dev",
    },
    "prod": {
        "db.service": "mysql",
        "db.host": "250.250.250.1",
        "db.port": "3306",
        "db.username": "root",
        "db.password": "secret",
        "db.db": "movies_prod",
    },
}
config := configurama.New(params)

// In your respective services/packages, parameters can be retrieved as such:
service, err := config.String("dev", "db.service", Required())
host, err := config.String("dev", "db.host", Required())
port, err := config.Int("dev", "db.port", Default("3306"))
username, err := config.String("dev", "db.username", Default("root"))
password, err := config.String("dev", "db.password")
db, err := config.String("dev", "db.db", Required())
```

### Notes

This package is _not_ concurrency-safe as I didn't deem it necessary for my
current needs. The common scenario is to simply load configuration from disk,
populate a configurama struct with it and then calling `Extract()` from various
other packages to extract local copies of each section. If you are using this
package and need an implementation that is concurrency-safe, let me know and
I'll implement it. Or better, provide a PR.

## V1

### Usage

V1 is initialized in the same was as V2.

## Example

Given the configuration file in YAML format:

```yaml
dev:
  db.service: mysql
  db.host: localhost
  db.username: me
  db.password: secret
  db.db: movies_dev
  
prod:
  db.service: mysql
  db.host: 250.250.250.1
  db.username: root
  db.password: secret
  db.db: movies_prod
```   

It's up to you to read this data from the file, but you'll probably want to end
up with this structure:

```go
params := map[string]map[string]string{
    "dev": {
        "db.service": "mysql",
        "db.host": "localhost",
        "db.username": "me",
        "db.password": "secret",
        "db.db": "movies_dev",
    },
    "prod": {
        "db.service": "mysql",
        "db.host": "250.250.250.1",
        "db.username": "root",
        "db.password": "secret",
        "db.db": "movies_prod",
    },
}
config := configurama.New(params)
```

You can, of course, also choose to only load the configuration that matches your
environment. But given the configuration above, you would then populate your local
configuration struct with all the database configuration using the appropriate prefix:

```go
type DBConfig struct {
    Service, Host, Username, Password, DB string
}

var dbConfig DBConfig

// This would fail to match parameters without the "db." prefix.
config.Extract("dev", "db.", &dbConfig)

// Access parameters as you wish:
fmt.Println(dbConfig.Username)
```

The `prefix` parameter for `Extract` (or `ExtractWithHooks`) is particularly useful when you still
need configuration for different services without getting name conflicts.

### Parameter extraction

A parameter P (key/value pair) should successfully be extracted into a struct S
when _1)_ S contains a field F that matches the key name in a case-insensitive
comparison, and _2)_ the data type of F can hold the value of P.

String values should extract as you would expect. For example, values such as
`"true"`, `"1"` will evaluate to boolean `true`, `"0.8"` evaluates to the
corresponding float value etc.  

### Using Hooks

You may run custom functions at two points during parameter extraction:

1. The _pre-hook_ runs before the provided struct is filled with parameters. It
   allows you to validate data, set defaults etc. If you return an error in this
   hook, extraction will fail.
2. The _post-hook_ runs after the provided struct has been filled with
   parameters. It's useful if you want to do extra work such as adding some
   non-config fields to your struct, for example.



## License

This software package is released under the MIT License.
