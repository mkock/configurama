package configurama_test

import (
	"fmt"

	"github.com/mkock/configurama"
)

// Example_Prefix provides a usage example for package configurama using
// the "prefix" parameter to match configuration parameters where their
// names are grouped together using a common prefix.
func Example_Prefix() {
	config := configurama.New(map[string]map[string]string{
		"Database": {
			"db.master.type":     "mysql",
			"db.master.Host":     "192.168.0.1",
			"db.master.Port":     "3306",
			"db.master.User":     "master-admin",
			"db.master.Password": "master-secret",
			"db.slave.type":      "mysql",
			"db.slave.Host":      "192.168.0.2",
			"db.slave.Port":      "3306",
			"db.slave.User":      "slave-admin",
			"db.slave.Password":  "slave-secret",
		},
	})

	type dbConfig struct {
		Type, Host, Port, User, Password string
	}
	var masterDBConf, slaveDBConf dbConfig

	config.Extract("Database", "db.master.", &masterDBConf)
	config.Extract("Database", "db.slave.", &slaveDBConf)

	fmt.Println("Master")
	fmt.Println("Type: " + masterDBConf.Type)
	fmt.Println("Host: " + masterDBConf.Host)
	fmt.Println("Port: " + masterDBConf.Port)
	fmt.Println("User: " + masterDBConf.User)
	fmt.Println("Password: " + masterDBConf.Password)
	fmt.Println("Slave")
	fmt.Println("Type: " + slaveDBConf.Type)
	fmt.Println("Host: " + slaveDBConf.Host)
	fmt.Println("Port: " + slaveDBConf.Port)
	fmt.Println("User: " + slaveDBConf.User)
	fmt.Println("Password: " + slaveDBConf.Password)

	// Output:
	// Master
	// Type: mysql
	// Host: 192.168.0.1
	// Port: 3306
	// User: master-admin
	// Password: master-secret
	// Slave
	// Type: mysql
	// Host: 192.168.0.2
	// Port: 3306
	// User: slave-admin
	// Password: slave-secret
}
