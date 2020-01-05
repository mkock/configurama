package configurama_test

import (
	"fmt"

	"github.com/mkock/configurama"
)

// Example_Basic provides a basic usage example for package configurama.
func Example_Basic() {
	config := configurama.New(map[string]map[string]string{
		"Database": {
			"Type":     "mysql",
			"Host":     "localhost",
			"Port":     "3306",
			"User":     "root",
			"Password": "secret",
		},
	})

	type dbConfig struct {
		Type, Host, Port, User, Password string
	}
	dbConf := dbConfig{}

	config.Extract("Database", "", &dbConf)

	fmt.Println("Type: " + dbConf.Type)
	fmt.Println("Host: " + dbConf.Host)
	fmt.Println("Port: " + dbConf.Port)
	fmt.Println("User: " + dbConf.User)
	fmt.Println("Password: " + dbConf.Password)

	// Output:
	// Type: mysql
	// Host: localhost
	// Port: 3306
	// User: root
	// Password: secret
}
