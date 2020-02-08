package configurama

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// empty is a convenience map that refers to the empty configuration pool.
var empty = map[string]map[string]string{}

func TestTypes(t *testing.T) {
	type Types struct {
		NegInt, ZeroInt, PosInt       int
		NegFloat, ZeroFloat, PosFloat float32
		ZeroBool, IntBool, TrueBool   bool
		ZeroString, MixedString       string
		InRange                       uint8
	}

	params := map[string]map[string]string{
		"default": {
			"NegInt":      "-50",
			"ZeroInt":     "0",
			"PosInt":      "9000",
			"NegFloat":    "-0.5",
			"ZeroFloat":   "0",
			"PosFloat":    "0.5",
			"ZeroBool":    "false",
			"IntBool":     "1",
			"TrueBool":    "true",
			"ZeroString":  "",
			"MixedString": "abc123ÆØÅÖ -- 9.0True",
			"InRange":     "12",
		},
	}

	c := New(params)
	var types Types
	err := c.Extract("default", "", &types)
	verifyNil(t, err)

	if types.NegInt != -50 {
		t.Errorf("expected value %d, got %d", -50, types.NegInt)
	}
	if types.ZeroInt != 0 {
		t.Errorf("expected value %d, got %d", 0, types.ZeroInt)
	}
	if types.PosInt != 9000 {
		t.Errorf("expected value %d, got %d", 9000, types.PosInt)
	}
	if types.NegFloat != -0.5 {
		t.Errorf("expected value %.2f, got %.2f", -0.5, types.NegFloat)
	}
	if types.ZeroFloat != 0.0 {
		t.Errorf("expected value %.2f, got %.2f", 0.0, types.ZeroFloat)
	}
	if types.PosFloat != 0.5 {
		t.Errorf("expected value %.2f, got %.2f", 0.5, types.PosFloat)
	}
	if types.ZeroBool != false {
		t.Errorf("expected value %t, got %t", false, types.ZeroBool)
	}
	if types.IntBool != true {
		t.Errorf("expected value %t, got %t", true, types.IntBool)
	}
	if types.TrueBool != true {
		t.Errorf("expected value %t, got %t", true, types.TrueBool)
	}
	if types.ZeroString != "" {
		t.Errorf("expected value %q, got %q", "", types.ZeroString)
	}
	if types.MixedString != "abc123ÆØÅÖ -- 9.0True" {
		t.Errorf("expected value %q, got %q", "abc123ÆØÅÖ -- 9.0True", types.MixedString)
	}
	if types.InRange != 12 {
		t.Errorf("expected value %d, got %d", 12, types.InRange)
	}
}

func TestRaw(t *testing.T) {
	params := map[string]map[string]string{
		"dev": {
			"db.host":       "localhost",
			"db.username":   "root",
			"db.password":   "secret",
			"db.connstring": "",
		},
	}
	cnf := New(params)
	actual := cnf.Raw()
	if !reflect.DeepEqual(params, actual) {
		t.Error("expected raw pool data to equal given pool data")
	}
}

func TestGet(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"db.host":       "localhost",
			"db.username":   "root",
			"db.password":   "secret",
			"db.connstring": "",
		},
	})

	tt := map[string]struct {
		section, key, expected string
		ok                     bool
	}{
		"empty case": {
			"", "", "", false,
		},
		"missing key": {
			"dev", "unknown", "", false,
		},
		"matching key": {
			"dev", "db.username", "root", true,
		},
		"matching key, empty value": {
			"dev", "db.connstring", "", true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual, ok := cnf.Get(tc.section, tc.key)
			if ok != tc.ok {
				t.Errorf("expected boolean %t, got %t", tc.ok, ok)
			}
			if actual != tc.expected {
				t.Errorf("expected value %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestUnset(t *testing.T) {
	type Hero struct {
		Name, Alias    string
		WorldSavedLast int
		Score          float32
	}

	c := New(map[string]map[string]string{"Hero": {
		"name":           "Peter Parker",
		"alias":          "Spiderman",
		"score":          "0.8",
		"worldSavedLast": "2018",
	}, "Enemy": {
		"name":  "Harry Osborne",
		"alias": "The Green Goblin",
		"score": "0.7",
	}})

	ok := c.Unset("Hero", "worldSavedLast")
	if !ok {
		t.Error("expected c.Unset() to return true, got false")
	}
	var h Hero
	err := c.Extract("Hero", "", &h)
	verifyNil(t, err)
	if h.Name != "Peter Parker" {
		t.Fatalf("expected Name to equal %q, got %q", "Peter Parker", h.Name)
	}
	if h.Alias != "Spiderman" {
		t.Fatalf("expected Alias to equal %q, got %q", "Spiderman", h.Alias)
	}
	if h.WorldSavedLast != 0 {
		t.Fatalf("expected WorldSavedLast to equal %d, got %d", 0, h.WorldSavedLast)
	}
	if h.Score != 0.8 {
		t.Fatalf("expected Score to equal %.5f, got %.5f", 0.8, h.Score)
	}
}

func TestExtract(t *testing.T) {
	type Hero struct {
		Name, Alias    string
		WorldSavedLast int
		Score          float32
		WearsMask      bool
	}

	t.Run("it handles the case of a missing section", func(t *testing.T) {
		var h Hero

		c := New(map[string]map[string]string{"Hero": {
			"name":           "Peter Parker",
			"alias":          "Spiderman",
			"worldSavedLast": "2018",
			"score":          "0.12345",
		}})

		err := c.Extract("Bad guy", "", &h)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expected := "unknown section name: Bad guy"
		if err.Error() != expected {
			t.Fatalf("expected error to equal %q, got %q", expected, err.Error())
		}
	})

	t.Run("it correctly extracts all parameters", func(t *testing.T) {
		var h Hero

		c := New(map[string]map[string]string{"Hero": {
			"name":           "Peter Parker",
			"alias":          "Spiderman",
			"worldSavedLast": "2018",
			"score":          "0.12345",
			"wearsmask":      "true",
		}})

		err := c.Extract("Hero", "", &h)
		verifyNil(t, err)
		if h.Name != "Peter Parker" {
			t.Errorf("expected Name to equal %q, got %q", "Peter Parker", h.Name)
		}
		if h.Alias != "Spiderman" {
			t.Errorf("expected Alias to equal %q, got %q", "Spiderman", h.Alias)
		}
		if h.WorldSavedLast != 2018 {
			t.Errorf("expected WorldSavedLast to equal %d, got %d", 2018, h.WorldSavedLast)
		}
		if h.Score != 0.12345 {
			t.Errorf("expected Score to equal %.5f, got %.5f", 0.12345, h.Score)
		}
		if !h.WearsMask {
			t.Error("expected WearsMask to be true, got false")
		}
	})

	t.Run("it ignores unmatched parameters", func(t *testing.T) {
		var h Hero

		c := New(map[string]map[string]string{"Hero": {
			"name":           "Peter Parker",
			"alias":          "Spiderman",
			"worldSavedLast": "2018",
			"score":          "0.12345",
			"other":          "Nothing to see here",
		}})

		err := c.Extract("Hero", "", &h)
		verifyNil(t, err)
		if h.Name != "Peter Parker" {
			t.Fatalf("expected Name to equal %q, got %q", "Peter Parker", h.Name)
		}
		if h.Alias != "Spiderman" {
			t.Fatalf("expected Alias to equal %q, got %q", "Spiderman", h.Alias)
		}
		if h.WorldSavedLast != 2018 {
			t.Fatalf("expected WorldSavedLast to equal %d, got %d", 2018, h.WorldSavedLast)
		}
		if h.Score != 0.12345 {
			t.Fatalf("expected Score to equal %.5f, got %.5f", 0.12345, h.Score)
		}
	})

	t.Run("it leaves unset parameters at their zero values", func(t *testing.T) {
		var h Hero

		c := New(map[string]map[string]string{"Hero": {
			"name":           "Peter Parker",
			"worldSavedLast": "2018",
		}})

		err := c.Extract("Hero", "", &h)
		verifyNil(t, err)
		if h.Name != "Peter Parker" {
			t.Fatalf("expected Name to equal %q, got %q", "Peter Parker", h.Name)
		}
		if h.Alias != "" {
			t.Fatalf("expected Alias to equal %q, got %q", "", h.Alias)
		}
		if h.WorldSavedLast != 2018 {
			t.Fatalf("expected WorldSavedLast to equal %d, got %d", 2018, h.WorldSavedLast)
		}
		if h.Score != 0.0 {
			t.Fatalf("expected Score to equal %.5f, got %.5f", 0.0, h.Score)
		}
	})

	t.Run("it errors for parameters that cannot be parsed", func(t *testing.T) {
		var h Hero

		c := New(map[string]map[string]string{"Hero": {
			"worldSavedLast": "yesterday", // Not a number.
		}})

		err := c.Extract("Hero", "", &h)
		if err == nil {
			t.Fatal("expected a parsing error, got nil")
		}
		if h.WorldSavedLast != 0 {
			t.Fatalf("expected WorldSavedLast to equal %d, got %d", 0, h.WorldSavedLast)
		}
	})

	t.Run("it handles sections correctly", func(t *testing.T) {
		var h1, h2 Hero

		c := New(map[string]map[string]string{"Hero": {
			"name":           "Peter Parker",
			"alias":          "Spiderman",
			"score":          "0.8",
			"worldSavedLast": "2018",
		}, "Enemy": {
			"name":  "Harry Osborne",
			"alias": "The Green Goblin",
			"score": "0.7",
		}})

		err1 := c.Extract("Hero", "", &h1)
		verifyNil(t, err1)
		if h1.Name != "Peter Parker" {
			t.Fatalf("expected Name to equal %q, got %q", "Peter Parker", h1.Name)
		}
		if h1.Alias != "Spiderman" {
			t.Fatalf("expected Alias to equal %q, got %q", "Spiderman", h1.Alias)
		}
		if h1.WorldSavedLast != 2018 {
			t.Fatalf("expected WorldSavedLast to equal %d, got %d", 2018, h1.WorldSavedLast)
		}
		if h1.Score != 0.8 {
			t.Fatalf("expected Score to equal %.5f, got %.5f", 0.8, h1.Score)
		}

		err2 := c.Extract("Enemy", "", &h2)
		verifyNil(t, err2)
		if h2.Name != "Harry Osborne" {
			t.Fatalf("expected Name to equal %q, got %q", "Harry Osborne", h2.Name)
		}
		if h2.Alias != "The Green Goblin" {
			t.Fatalf("expected Alias to equal %q, got %q", "The Green Goblin", h2.Alias)
		}
		if h2.WorldSavedLast != 0 {
			t.Fatalf("expected WorldSavedLast to equal %d, got %d", 0, h2.WorldSavedLast)
		}
		if h2.Score != 0.7 {
			t.Fatalf("expected Score to equal %.5f, got %.5f", 0.8, h2.Score)
		}
	})

	t.Run("it does case insensitive matching", func(t *testing.T) {
		c := New(map[string]map[string]string{"MyConfig": {
			"serviceName":  "MySQL",       // Lowercased first letter.
			"PORTNUMBER":   "3306",        // Uppercase letters.
			"username":     "root",        // Lowercase letters.
			"dbpassword":   "secret",      // Mismatched case.
			"realpassword": "more secret", // Unexported field in struct.
		}})

		type TitleCaseConfig struct {
			ServiceName  string
			PortNumber   int
			UserName     string
			DBPassword   string
			realPassword string
		}

		var tcConfig TitleCaseConfig
		err1 := c.Extract("MyConfig", "", &tcConfig)
		verifyNil(t, err1)
		if tcConfig.ServiceName != "MySQL" {
			t.Errorf("expected ServiceName to equal %q, got %q", "MySQL", tcConfig.ServiceName)
		}
		if tcConfig.PortNumber != 3306 {
			t.Errorf("expected PortNumber to equal %d, got %d", 3306, tcConfig.PortNumber)
		}
		if tcConfig.UserName != "root" {
			t.Errorf("expected UserName to equal %q, got %q", "root", tcConfig.UserName)
		}
		if tcConfig.DBPassword != "secret" {
			t.Errorf("expected DBPassword to equal %q, got %q", "secret", tcConfig.DBPassword)
		}
		if tcConfig.realPassword != "" {
			t.Errorf("expected realPassword to equal %q, got %q", "", tcConfig.realPassword)
		}
	})

	t.Run("it handles prefixes correctly", func(t *testing.T) {
		c := New(map[string]map[string]string{"DBConfig": {
			"db.mysql.servicename":       "MySQL",
			"db.mysql.portnumber":        "3306",
			"db.mysql.username":          "root",
			"db.mysql.password":          "secret",
			"unrelated.service.name":     "MyService",
			"unrelated.service.username": "Martin",
		}})

		type DBConfig struct {
			ServiceName string
			PortNumber  int
			UserName    string
			Password    string
		}

		var dbConfig DBConfig
		err1 := c.Extract("DBConfig", "db.mysql.", &dbConfig)
		verifyNil(t, err1)
		if dbConfig.ServiceName != "MySQL" {
			t.Errorf("expected ServiceName to equal %q, got %q", "MySQL", dbConfig.ServiceName)
		}
		if dbConfig.PortNumber != 3306 {
			t.Errorf("expected PortNumber to equal %d, got %d", 3306, dbConfig.PortNumber)
		}
		if dbConfig.UserName != "root" {
			t.Errorf("expected UserName to equal %q, got %q", "root", dbConfig.UserName)
		}
		if dbConfig.Password != "secret" {
			t.Errorf("expected Password to equal %q, got %q", "secret", dbConfig.Password)
		}
	})
}

func TestExtractWithHooks(t *testing.T) {
	type Contact struct {
		Name, Gender, Email, Phone, Address, Zip, City, Country string
	}

	params := map[string]map[string]string{"main": {
		"name":    "Guy Friendly",
		"gender":  "male",
		"email":   "somebody@somewhere.com",
		"phone":   "+45 25205016",
		"address": "Overthere Street 18",
		"zip":     "2200",
		"city":    "Copenhagen",
		"country": "Denmark",
	}, "extra": {
		"name":    "Kim Kindly",
		"gender":  "female",
		"email":   "someone@elsewhere.com",
		"phone":   "",
		"address": "Elsewhere Street 22",
		"zip":     "2200",
		"city":    "Copenhagen",
		"country": "Denmark",
	}, "prefixed": {
		"one.var1": "var1.1",
		"one.var2": "var1.2",
		"two.var1": "var2.1",
		"two.var2": "var2.2",
	}}

	t.Run("it works with nil hooks", func(t *testing.T) {
		var con Contact
		c := New(params)
		err := c.ExtractWithHooks("main", "", &con, nil, nil)
		verifyNil(t, err)
	})

	t.Run("it calls the pre hook with correct parameters", func(t *testing.T) {
		var con Contact
		var called bool

		c := New(params)
		verifyContact := func(t *testing.T, m map[string]string) {
			t.Helper()
			if len(m) != 8 {
				t.Errorf("expected %d parameters, got %d", 8, len(m))
			}
			if !reflect.DeepEqual(params["main"], m) {
				t.Errorf("expected pre-hook parameters to equal parameters from section %q", "main")
			}
		}
		pre := func(m map[string]string) error {
			verifyContact(t, m)
			called = true
			return nil
		}
		err := c.ExtractWithHooks("main", "", &con, pre, nil)
		verifyNil(t, err)
		if !called {
			t.Errorf("expected pre-hook to have been called")
		}
	})

	t.Run("it calls the pre hook with only prefix parameters", func(t *testing.T) {
		var con Contact
		var called bool

		c := New(params)
		verifyContact := func(t *testing.T, m map[string]string) {
			t.Helper()
			if len(m) != 2 {
				t.Errorf("expected %d parameters, got %d", 2, len(m))
			}
			if m["var1"] != "var1.1" {
				t.Errorf("expected pre-hook parameter %q to equal %q, got %q", "var1", "var1.1", m["var1"])
			}
			if m["var2"] != "var1.2" {
				t.Errorf("expected pre-hook parameter %q to equal %q, got %q", "var2", "var1.2", m["var2"])
			}
		}
		pre := func(m map[string]string) error {
			verifyContact(t, m)
			called = true
			return nil
		}
		err := c.ExtractWithHooks("prefixed", "one.", &con, pre, nil)
		verifyNil(t, err)
		if !called {
			t.Errorf("expected pre-hook to have been called")
		}
	})

	t.Run("it returns the same error as the pre-hook", func(t *testing.T) {
		var con Contact
		var called bool

		c := New(params)
		pre := func(m map[string]string) error {
			called = true
			return errors.New("I made a boo-boo")
		}
		err := c.ExtractWithHooks("main", "", &con, pre, nil)
		if !called {
			t.Errorf("expected pre-hook to have been called")
		}
		if err == nil {
			t.Error("expected an error")
		}
		if err.Error() != "I made a boo-boo" {
			t.Errorf("expected error to contain %q, got %q", "I made a boo-boo", err.Error())
		}
	})

	t.Run("it calls the post hook", func(t *testing.T) {
		var con Contact
		var called bool

		c := New(params)
		pre := func(m map[string]string) error {
			return errors.New("I made a boo-boo")
		}
		post := func() {
			called = true
		}
		err := c.ExtractWithHooks("main", "", &con, pre, post)
		if err == nil {
			t.Error("expected an error")
		}
		if called {
			t.Error("expected post-hook to not have been called")
		}
	})

	t.Run("it does not call the post hook when the pre hook returns an error", func(t *testing.T) {
		var con Contact
		var called bool

		c := New(params)
		post := func() {
			called = true
		}
		err := c.ExtractWithHooks("main", "", &con, nil, post)
		verifyNil(t, err)
		if !called {
			t.Errorf("expected post-hook to have been called")
		}
	})
}

func TestMergeOverwriteStrategy(t *testing.T) {
	tt := map[string]struct {
		first    map[string]map[string]string
		second   map[string]map[string]string
		expected map[string]map[string]string
	}{
		"empty case": {
			first:    empty,
			second:   empty,
			expected: empty,
		},
		"trivial case, first": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   empty,
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, second": {
			first:    empty,
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, overlapping": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, overlapping, diff value": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value two"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value two"}},
		},
		"simple case, overlapping, diff keys": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two", "Field three": "Value three"}},
		},
		"simple case, overlapping reverse, diff keys": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two", "Field three": "Value three"}},
		},
		"complex case, sections with overlapping keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two"},
				"Section two":   {"Field six": "Value two-six-new"},
				"Section three": {"Field one": "Value two-one"},
			},
			expected: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two":   {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six-new"},
				"Section three": {"Field one": "Value two-one"},
			},
		},
		"complex case, sections with disjoint keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field two": "Value two-two", "Field four": "Value one-four"},
				"Section two":   {"Field two": "Value two-two", "Field four": "Value two-four", "Field five": "Value two-five"},
				"Section three": {"Field one": "Value three-one"},
			},
			expected: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two", "Field three": "Value one-three", "Field four": "Value one-four", "Field five": "Value one-five"},
				"Section two":   {"Field one": "Value two-one", "Field two": "Value two-two", "Field three": "Value two-three", "Field four": "Value two-four", "Field five": "Value two-five", "Field six": "Value two-six"},
				"Section three": {"Field one": "Value three-one"},
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual, err := merge(tc.first, tc.second, Overwrite)
			verifyNil(t, err)
			verifyEqual(t, tc.expected, actual)
		})
	}
}

func TestMergeKeepStrategy(t *testing.T) {
	tt := map[string]struct {
		first    map[string]map[string]string
		second   map[string]map[string]string
		expected map[string]map[string]string
	}{
		"empty case": {
			first:    empty,
			second:   empty,
			expected: empty,
		},
		"trivial case, first": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   empty,
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, second": {
			first:    empty,
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, overlapping": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"trivial case, overlapping, diff value": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value two"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
		},
		"simple case, overlapping, diff keys": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two", "Field three": "Value three"}},
		},
		"simple case, overlapping reverse, diff keys": {
			first:    map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			second:   map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			expected: map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two", "Field three": "Value three"}},
		},
		"complex case, sections with overlapping keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two"},
				"Section two":   {"Field six": "Value two-six-new"},
				"Section three": {"Field one": "Value two-one"},
			},
			expected: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two":   {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
				"Section three": {"Field one": "Value two-one"},
			},
		},
		"complex case, sections with disjoint keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field two": "Value two-two", "Field four": "Value one-four"},
				"Section two":   {"Field two": "Value two-two", "Field four": "Value two-four", "Field five": "Value two-five"},
				"Section three": {"Field one": "Value three-one"},
			},
			expected: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two", "Field three": "Value one-three", "Field four": "Value one-four", "Field five": "Value one-five"},
				"Section two":   {"Field one": "Value two-one", "Field two": "Value two-two", "Field three": "Value two-three", "Field four": "Value two-four", "Field five": "Value two-five", "Field six": "Value two-six"},
				"Section three": {"Field one": "Value three-one"},
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual, err := merge(tc.first, tc.second, Keep)
			verifyNil(t, err)
			verifyEqual(t, tc.expected, actual)
		})
	}
}

func TestMergeReportStrategy(t *testing.T) {
	tt := map[string]struct {
		first         map[string]map[string]string
		second        map[string]map[string]string
		expected      map[string]map[string]string
		expectedError error
	}{
		"empty case": {
			first:         empty,
			second:        empty,
			expected:      empty,
			expectedError: nil,
		},
		"trivial case, first": {
			first:         map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:        empty,
			expected:      map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expectedError: nil,
		},
		"trivial case, second": {
			first:         empty,
			second:        map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected:      map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expectedError: nil,
		},
		"trivial case, overlapping, diff value": {
			first:         map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			second:        map[string]map[string]string{"Section one": {"Field one": "Value two"}},
			expected:      empty,
			expectedError: fmt.Errorf("section %q, key %q already exists", "Section one", "Field one"),
		},
		"simple case, overlapping, diff keys": {
			first:         map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			second:        map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			expected:      empty,
			expectedError: fmt.Errorf("section %q, key %q already exists", "Section one", "Field one"),
		},
		"simple case, overlapping reverse, diff keys": {
			first:         map[string]map[string]string{"Section one": {"Field one": "Value one", "Field three": "Value three"}},
			second:        map[string]map[string]string{"Section one": {"Field one": "Value one", "Field two": "Value two"}},
			expected:      empty,
			expectedError: fmt.Errorf("section %q, key %q already exists", "Section one", "Field one"),
		},
		"complex case, sections with overlapping keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field two": "Value two-two", "Field four": "Value one-four"},
				"Section two":   {"Field six": "Value two-six-new"},
				"Section three": {"Field one": "Value two-one"},
			},
			expected:      empty,
			expectedError: fmt.Errorf("section %q, key %q already exists", "Section two", "Field six"),
		},
		"complex case, sections with disjoint keys": {
			first: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field three": "Value one-three", "Field five": "Value one-five"},
				"Section two": {"Field one": "Value two-one", "Field three": "Value two-three", "Field six": "Value two-six"},
			},
			second: map[string]map[string]string{
				"Section one":   {"Field two": "Value two-two", "Field four": "Value one-four"},
				"Section two":   {"Field two": "Value two-two", "Field four": "Value two-four", "Field five": "Value two-five"},
				"Section three": {"Field one": "Value three-one"},
			},
			expected: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value two-two", "Field three": "Value one-three", "Field four": "Value one-four", "Field five": "Value one-five"},
				"Section two":   {"Field one": "Value two-one", "Field two": "Value two-two", "Field three": "Value two-three", "Field four": "Value two-four", "Field five": "Value two-five", "Field six": "Value two-six"},
				"Section three": {"Field one": "Value three-one"},
			},
			expectedError: nil,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual, err := merge(tc.first, tc.second, Report)
			if tc.expectedError != nil {
				if err == nil {
					t.Error("expected an error, got nil")
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("expected error message to match %q, got %q", tc.expectedError.Error(), err.Error())
				}
			} else {
				verifyNil(t, err)
				verifyEqual(t, tc.expected, actual)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tt := map[string]struct {
		p1, p2, expected map[string]map[string]string
	}{
		"empty case": {
			p1:       empty,
			p2:       empty,
			expected: empty,
		},
		"trivial case": {
			p1:       map[string]map[string]string{"default": {"key": "val"}},
			p2:       map[string]map[string]string{"default": {"key": "val"}},
			expected: empty,
		},
		"new keys": {
			p1:       map[string]map[string]string{"default": {}},
			p2:       map[string]map[string]string{"default": {"key1": "val1", "key2": "val2"}},
			expected: map[string]map[string]string{"default": {"key1": "val1", "key2": "val2"}},
		},
		"empty values": {
			p1:       map[string]map[string]string{"default": {"key3": "val3"}},
			p2:       map[string]map[string]string{"default": {"key1": "", "key2": ""}},
			expected: map[string]map[string]string{"default": {"key1": "", "key2": ""}},
		},
		"one diff in one section": {
			p1:       map[string]map[string]string{"default": {"key": "val"}},
			p2:       map[string]map[string]string{"default": {"key": "val2"}},
			expected: map[string]map[string]string{"default": {"key": "val2"}},
		},
		"two diffs in one section": {
			p1:       map[string]map[string]string{"default": {"key": "val"}},
			p2:       map[string]map[string]string{"default": {"key1": "val1", "key2": "val2"}},
			expected: map[string]map[string]string{"default": {"key1": "val1", "key2": "val2"}},
		},
		"two diffs in one section, reverse": {
			p1:       map[string]map[string]string{"default": {"key1": "val1", "key2": "val2"}},
			p2:       map[string]map[string]string{"default": {"key": "val"}},
			expected: map[string]map[string]string{"default": {"key": "val"}},
		},
		"two diffs in two sections": {
			p1:       map[string]map[string]string{"section1": {"key1": "val1"}, "section2": {"key2": "val2", "key3": "val3"}},
			p2:       map[string]map[string]string{"section1": {"KEY1": "val1"}, "section2": {"key2": "VAL2", "key3": "val3", "key4": "val4"}},
			expected: map[string]map[string]string{"section1": {"KEY1": "val1"}, "section2": {"key2": "VAL2", "key4": "val4"}},
		},
		"two identical sections": {
			p1:       map[string]map[string]string{"section1": {"key1": "val1"}, "section2": {"key2": "val2", "key3": "val3"}},
			p2:       map[string]map[string]string{"section1": {"key1": "val1"}, "section2": {"key2": "val2", "key3": "val3"}},
			expected: empty,
		},
	}

	for name, tc := range tt {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			c1 := New(tc.p1)
			c2 := New(tc.p2)
			actual := c1.Compare(*c2)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected c1 to equal c2, got diff: %v", actual)
			}
		})
	}
}

func TestMustPrettyPrint(t *testing.T) {
	tt := map[string]struct {
		config   map[string]map[string]string
		expected string
	}{
		"empty case": {
			config:   empty,
			expected: "",
		},
		"trivial case": {
			config: map[string]map[string]string{"Section one": {"Field one": "Value one"}},
			expected: `[Section one]
  Field one: Value one`,
		},
		"trivial edge case": {
			config:   map[string]map[string]string{"Section one": {}},
			expected: `[Section one]`,
		},
		"two-section case": {
			config: map[string]map[string]string{
				"Section one": {"Field one": "Value one-one", "Field two": "Value one-two", "Field three": "Value one-three"},
				"Section two": {"Field one": "Value two-one", "Field two": "Value two-two", "Field three": "Value two-three"},
			},
			expected: `[Section one]
  Field one: Value one-one
  Field three: Value one-three
  Field two: Value one-two

[Section two]
  Field one: Value two-one
  Field three: Value two-three
  Field two: Value two-two`,
		},
		"multi-section case": {
			config: map[string]map[string]string{
				"Section one":   {"Field one": "Value one-one", "Field two": "Value one-two", "Field three": "Value one-three"},
				"Section two":   {"Field one": "Value two-one", "Field two": "Value two-two", "Field three": "Value two-three"},
				"Section three": {},
				"Section four":  {"Field one": "Value four-one", "Field two": "Value four-two", "Field three": "Value four-three"},
			},
			expected: `[Section four]
  Field one: Value four-one
  Field three: Value four-three
  Field two: Value four-two

[Section one]
  Field one: Value one-one
  Field three: Value one-three
  Field two: Value one-two

[Section three]

[Section two]
  Field one: Value two-one
  Field three: Value two-three
  Field two: Value two-two`,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual := MustPrettyPrint(tc.config, "  ")
			if actual != tc.expected {
				t.Errorf("expected output to match %q, got %q", tc.expected, actual)
			}
		})
	}
}

func verifyNil(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}
}

func verifyEqual(t *testing.T, first, second map[string]map[string]string) {
	if !reflect.DeepEqual(first, second) {
		fmt.Println(MustPrettyPrint(first, "  "))
		fmt.Println(" != ")
		fmt.Println(MustPrettyPrint(second, "  "))
		t.Error("expected the two maps to be identical")
	}
}
