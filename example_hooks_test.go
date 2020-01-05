package configurama_test

import (
	"errors"
	"fmt"

	"github.com/mkock/configurama"
)

// Example_Hooks provides a usage example for package configurama using
// hooks to validate the configuration parameters.
func Example_Hooks() {
	config := configurama.New(map[string]map[string]string{
		"Database": {
			"type":     "mysql",
			"host":     "localhost",
			"port":     "3306",
			"user":     "admin",
			"password": "secret",
		},
	})

	type dbConfig struct {
		Type, Host, Port, User, Password string
	}
	var dbConf dbConfig

	validate := func(m map[string]string) error {
		if m["host"] == "localhost" {
			return errors.New("localhost connections disallowed")
		}
		return nil
	}

	err := config.ExtractWithHooks("Database", "", &dbConf, validate, nil)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("DB")
		fmt.Println("Type: " + dbConf.Type)
		fmt.Println("Host: " + dbConf.Host)
		fmt.Println("Port: " + dbConf.Port)
		fmt.Println("User: " + dbConf.User)
		fmt.Println("Password: " + dbConf.Password)
	}

	// Output:
	// Error: localhost connections disallowed
}
