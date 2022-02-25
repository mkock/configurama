package configurama

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	// empty is a convenience map that refers to the empty configuration pool.
	empty = map[string]map[string]string{}

	// notABCError is used by validateABC.
	notABCError = errors.New("not a,b,c")

	// validateABC is a validation function that checks if the given string equals a, b or c.
	validateABC = func(key, val string) error {
		if val == "a" || val == "b" || val == "c" {
			return nil
		}
		return notABCError
	}
)

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

func TestCheckApplyOptions(t *testing.T) {
	tt := map[string]struct {
		key, value    string
		ok            bool
		options       []Option
		expectedError error
		expectedValue string
	}{
		"no value, options: none": {
			"x", "", false, []Option{}, nil, "",
		},
		"got value, options: none": {
			"x", "y", true, []Option{}, nil, "y",
		},
		"got value, options: required": {
			"x", "y", true, []Option{Require()}, nil, "y",
		},
		"no value, options: required": {
			"x", "", false, []Option{Require()}, NoKeyError("x"), "",
		},
		"got value, options: required, default": {
			"x", "y", true, []Option{Require(), Default("z")}, nil, "y",
		},
		"got value, options: default": {
			"x", "y", true, []Option{Default("z")}, nil, "y",
		},
		"no value, options: required, default": {
			"x", "", false, []Option{Require(), Default("z")}, NoKeyError("x"), "",
		},
		"no value, options: default": {
			"x", "", false, []Option{Default("z")}, nil, "z",
		},
		"got value, options: validate (fails)": {
			"x", "y", true, []Option{ValidateRegExp(regexp.MustCompile(`^[0-9]$`))}, RegExpValidationError("x"), "",
		},
		"got value, options: validate (succeeds)": {
			"x", "y", true, []Option{ValidateRegExp(regexp.MustCompile(`^[a-z]$`))}, nil, "y",
		},
		"no value, options: default, validate (succeeds)": {
			"x", "", false, []Option{Default("z"), ValidateRegExp(regexp.MustCompile(`^[a-z]$`))}, nil, "z",
		},
		"no value, options: validate (succeeds), default": {
			"x", "", false, []Option{ValidateRegExp(regexp.MustCompile(`^[a-z]$`)), Default("z")}, nil, "z",
		},
		"no value, options: validate (fails), default": {
			"x", "", false, []Option{ValidateRegExp(regexp.MustCompile(`^hello$`)), Default("z")}, RegExpValidationError("x"), "",
		},
		"got value, options: require, validate (succeeds), default": {
			"x", "y", true, []Option{Require(), ValidateRegExp(regexp.MustCompile(`^y$`)), Default("z")}, nil, "y",
		},
		"no value, options: require, validate (succeeds), default": {
			"x", "", false, []Option{Require(), ValidateRegExp(regexp.MustCompile(`^y$`)), Default("z")}, NoKeyError("x"), "",
		},
		"got value, options: require, validate integral (succeeds)": {
			"x", "123", true, []Option{Require(), ValidateIntegral()}, nil, "123",
		},
		"got value, options: require, validate integral (fails)": {
			"x", "123.5", true, []Option{Require(), ValidateIntegral()}, RegExpValidationError("x"), "",
		},
		"got value, options: require, validate func (fails)": {
			"x", "d", true, []Option{Require(), ValidateFunc(validateABC)}, notABCError, "",
		},
		"got value, options: require, validate func (succeeds)": {
			"x", "c", true, []Option{Require(), ValidateFunc(validateABC)}, nil, "c",
		},
		"got value, options: require, validate enum (fails)": {
			"x", "d", true, []Option{Require(), ValidateEnum([]string{"a", "b", "c"})}, EnumValidationError("x"), "",
		},
		"got value, options: require, validate enum (succeeds)": {
			"x", "a", true, []Option{Require(), ValidateEnum([]string{"a", "b", "c"})}, nil, "a",
		},
		"got value, options: require, validate empty enum (succeeds)": {
			"x", "a", true, []Option{Require(), ValidateEnum(nil)}, nil, "a",
		},
		"no value, options: validate empty enum (succeeds), default": {
			"x", "", false, []Option{ValidateEnum(nil), Default("a")}, nil, "a",
		},
	}

	var actual string
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			actual, err = checkApplyOptions(tc.key, tc.value, tc.ok, tc.options...)
			if err != tc.expectedError {
				t.Errorf("expected error %q, got %q", tc.expectedError, err)
			}
			if actual != tc.expectedValue {
				t.Errorf("expected value %q, got %q", tc.expectedValue, actual)
			}
		})
	}
}

func TestString(t *testing.T) {
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
		sectionExists          bool
		err                    error
	}{
		"empty case": {
			"", "", "", false, nil,
		},
		"missing key": {
			"dev", "unknown", "", true, nil,
		},
		"matching key": {
			"dev", "db.username", "root", true, nil,
		},
		"matching key, empty value": {
			"dev", "db.connstring", "", true, nil,
		},
	}

	var actual string
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.String(tc.key)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if actual != tc.expected {
				t.Errorf("expected value %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestStrings(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":    "0",
			"simple":  "simple",
			"long":    "one,two,three,four,five,six,seven,eight,nine,ten",
			"complex": "14,hello:56,\"quo,ted\",+,",
			"empty":   "",
		},
	})

	tt := map[string]struct {
		section, key, separator string
		options                 []Option
		expected                []string
		sectionExists           bool
		err                     error
	}{
		"empty case": {
			"", "", "", []Option{}, nil, false, nil,
		},
		"missing key": {
			"dev", "unknown", "", []Option{}, nil, true, nil,
		},
		"matching key": {
			"dev", "simple", ",", []Option{}, []string{"simple"}, true, nil,
		},
		"matching key, empty value": {
			"dev", "empty", ",", []Option{}, []string{}, true, nil,
		},
		"matching key, empty value with default": {
			"dev", "empty", ",", []Option{Default("one,two,three")}, []string{"one", "two", "three"}, true, nil,
		},
		"matching key, long": {
			"dev", "long", ",", []Option{}, []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}, true, nil,
		},
		"matching key, long, different separator": {
			"dev", "long", ":", []Option{}, []string{"one,two,three,four,five,six,seven,eight,nine,ten"}, true, nil,
		},
		"matching key, complex": {
			"dev", "complex", ",", []Option{}, []string{"14", "hello:56", "\"quo", "ted\"", "+", ""}, true, nil,
		},
		"matching key, complex with validation (success)": {
			"dev", "complex", ",", []Option{ValidateRegExp(regexp.MustCompile(`^[0-9]{2}`))}, []string{"14", "hello:56", "\"quo", "ted\"", "+", ""}, true, nil,
		},
		"matching key, complex with validation (failed)": {
			"dev", "complex", ",", []Option{ValidateRegExp(regexp.MustCompile(`^[a-z]{2}`))}, []string{}, true, RegExpValidationError("complex"),
		},
	}

	var actual []string
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Strings(tc.key, tc.separator, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if strings.Join(actual, ",") != strings.Join(tc.expected, ",") {
				t.Errorf("expected value %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestInt(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":    "0",
			"invalid": "invalid",
			"int":     "14",
			"float":   "12.5",
			"empty":   "",
		},
	})

	tt := map[string]struct {
		section, key  string
		options       []Option
		expected      int
		sectionExists bool
		err           error
	}{
		"empty case": {
			"", "", []Option{}, 0, false, nil,
		},
		"missing key with default": {
			"dev", "unknown", []Option{Default("90")}, 90, true, nil,
		},
		"missing key, required": {
			"dev", "unknown", []Option{Require()}, 0, true, NoKeyError("unknown"),
		},
		"matching key": {
			"dev", "int", []Option{}, 14, true, nil,
		},
		"matching key, empty value": {
			"dev", "zero", []Option{}, 0, true, nil,
		},
		"matching key, invalid value": {
			"dev", "invalid", []Option{}, 0, true, ConversionError{"invalid", "invalid", "int"},
		},
		"matching key, float value": {
			"dev", "float", []Option{}, 0, true, ConversionError{"float", "12.5", "int"},
		},
		"empty parameter with default": {
			"dev", "empty", []Option{Default("42")}, 42, true, nil,
		},
	}

	var actual int
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Int(tc.key, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if actual != tc.expected {
				t.Errorf("expected value %d, got %d", tc.expected, actual)
			}
		})
	}
}

func TestFloat(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":    "0",
			"invalid": "invalid",
			"int":     "14",
			"float":   "12.5",
			"empty":   "",
		},
	})

	tt := map[string]struct {
		section, key  string
		options       []Option
		expected      float64
		sectionExists bool
		err           error
	}{
		"empty case": {
			"", "", []Option{}, 0, false, nil,
		},
		"missing key": {
			"dev", "unknown", []Option{}, 0, true, nil,
		},
		"matching key": {
			"dev", "int", []Option{}, 14, true, nil,
		},
		"matching key, empty value": {
			"dev", "zero", []Option{}, 0, true, nil,
		},
		"matching key, empty value with default": {
			"dev", "zero", []Option{Default("42")}, 0, true, nil,
		},
		"matching key, invalid value": {
			"dev", "invalid", []Option{}, 0, true, ConversionError{"invalid", "invalid", "float64"},
		},
		"matching key, float value": {
			"dev", "float", []Option{}, 12.5, true, nil,
		},
		"empty parameter with default": {
			"dev", "empty", []Option{Default("42")}, 42, true, nil,
		},
	}

	var actual float64
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Float(tc.key, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if actual != tc.expected {
				t.Errorf("expected value %.2f, got %.2f", tc.expected, actual)
			}
		})
	}
}

func TestBool(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":    "0",
			"invalid": "invalid",
			"int":     "1",
			"true":    "true",
			"yes":     "y",
			"no":      "no",
			"empty":   "",
		},
	})

	tt := map[string]struct {
		section, key  string
		options       []Option
		expected      bool
		sectionExists bool
		err           error
	}{
		"empty case": {
			"", "", []Option{}, false, false, nil,
		},
		"missing key": {
			"dev", "unknown", []Option{}, false, true, nil,
		},
		"matching key": {
			"dev", "int", []Option{}, true, true, nil,
		},
		"matching key, empty value": {
			"dev", "zero", []Option{}, false, true, nil,
		},
		"matching key, empty value with default": {
			"dev", "empty", []Option{Default("t")}, true, true, nil,
		},
		"matching key, invalid value": {
			"dev", "invalid", []Option{}, false, true, ConversionError{"invalid", "invalid", "bool"},
		},
		"matching key, verbal": {
			"dev", "true", []Option{}, true, true, nil,
		},
		"matching key with default": {
			"dev", "no", []Option{Default("true")}, false, true, nil,
		},
	}

	var actual bool
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Bool(tc.key, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if actual != tc.expected {
				t.Errorf("expected value %t, got %t", tc.expected, actual)
			}
		})
	}
}

func TestDuration(t *testing.T) {
	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":     "0",
			"invalid":  "invalid",
			"duration": "10m20s",
			"negative": "-1.5h",
			"empty":    "",
		},
	})

	tt := map[string]struct {
		section, key  string
		options       []Option
		expected      string
		sectionExists bool
		err           error
	}{
		"empty case": {
			"", "", []Option{}, "0s", false, nil,
		},
		"missing key": {
			"dev", "unknown", []Option{}, "0s", true, nil,
		},
		"matching key": {
			"dev", "duration", []Option{}, "10m20s", true, nil,
		},
		"matching key, empty value": {
			"dev", "zero", []Option{}, "0s", true, nil,
		},
		"matching key, invalid value": {
			"dev", "invalid", []Option{}, "0s", true, ConversionError{"invalid", "invalid", "Duration"},
		},
		"matching key, negative duration": {
			"dev", "negative", []Option{}, "-1h30m0s", true, nil,
		},
		"empty parameter with default": {
			"dev", "empty", []Option{Default("30m")}, "30m0s", true, nil,
		},
		"empty parameter with invalid default": {
			"dev", "empty", []Option{Default("hello")}, "0s", true, ConversionError{"empty", "hello", "Duration"},
		},
	}

	var actual time.Duration
	var err error
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Duration(tc.key, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if actual.String() != tc.expected {
				t.Errorf("expected value %q, got %q", tc.expected, actual.String())
			}
		})
	}
}

func TestTime(t *testing.T) {
	reference, err := time.Parse(time.RFC3339, "2021-11-06T22:30:00+01:00")
	verifyNil(t, err)

	cnf := New(map[string]map[string]string{
		"dev": {
			"zero":    "0",
			"invalid": "invalid",
			"time":    "2021-11-06T22:30:00+01:00",
			"date":    "2021-11-06Z01:00",
			"empty":   "",
		},
	})

	tt := map[string]struct {
		section, key, format string
		options              []Option
		expected             time.Time
		sectionExists        bool
		err                  error
	}{
		"empty case": {
			"", "", time.RFC3339, []Option{}, time.Time{}, false, nil,
		},
		"missing key": {
			"dev", "unknown", time.RFC3339, []Option{}, time.Time{}, true, nil,
		},
		"matching key": {
			"dev", "time", time.RFC3339, []Option{}, reference, true, nil,
		},
		"matching key, empty value": {
			"dev", "zero", time.RFC3339, []Option{}, time.Time{}, true, ConversionError{"zero", "0", "Time"},
		},
		"matching key, invalid value": {
			"dev", "invalid", time.RFC3339, []Option{}, time.Time{}, true, ConversionError{"invalid", "invalid", "Time"},
		},
		"empty parameter with default": {
			"dev", "empty", time.RFC3339, []Option{Default("2021-11-06T22:30:00+01:00")}, reference, true, nil,
		},
		"empty parameter with invalid default": {
			"dev", "empty", time.RFC3339, []Option{Default("hello")}, time.Time{}, true, ConversionError{"empty", "hello", "Time"},
		},
	}

	var actual time.Time
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			sec, ok := cnf.Section(tc.section)
			if ok != tc.sectionExists {
				t.Errorf("expected to find section %q", tc.section)
			}
			actual, err = sec.Time(tc.key, tc.format, tc.options...)
			if err != tc.err {
				t.Errorf("expected error %s, got %s", tc.err, err)
			}
			if !actual.Equal(tc.expected) {
				t.Errorf("expected value %q, got %q", tc.expected, actual.String())
			}
		})
	}
}

func TestUnset(t *testing.T) {
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

	hero, ok := c.Section("Hero")
	if !ok {
		t.Errorf("expected to find section %q", "Hero")
	}
	name, _ := hero.String("name")
	alias, _ := hero.String("alias")
	worldSavedLast, _ := hero.String("worldSavedLast")
	score, _ := hero.String("score")

	if name != "Peter Parker" {
		t.Fatalf("expected Name to equal %q, got %q", "Peter Parker", name)
	}
	if alias != "Spiderman" {
		t.Fatalf("expected Alias to equal %q, got %q", "Spiderman", alias)
	}
	if worldSavedLast != "" {
		t.Fatalf("expected WorldSavedLast to equal %q, got %q", "", worldSavedLast)
	}
	if score != "0.8" {
		t.Fatalf("expected Score to equal %q, got %q", "0.8", score)
	}
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
			actual := c1.Compare(c2)
			verifyEqual(t, tc.expected, actual)
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

func TestErrors(t *testing.T) {
	var actual error
	var ok bool

	errConv := ConversionError{"x", "y", "Int"}
	if ok = errors.As(errConv, &actual); !ok {
		t.Errorf("expected to be able to match ConversionError")
	}

	errNoKey := NoKeyError("x")
	if ok = errors.As(errNoKey, &actual); !ok {
		t.Errorf("expected to be able to match NoKeyError")
	}

	errValid := RegExpValidationError("x")
	if ok = errors.As(errValid, &actual); !ok {
		t.Errorf("expected to be able to match RegExpValidationError")
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
