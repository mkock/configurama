package configurama

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	integralRegExp = regexp.MustCompile(`^[0-9]*$`)
}

var (
	// Default sets a default value that will be returned for empty parameters.
	Default = func(val string) Option { return func(o *option) { o.defaultValue = val } }

	// Require sets a parameter as required. Empty parameters will cause an error to be returned when fetched.
	Require = func() Option { return func(o *option) { o.require = true } }

	// Validate validates a parameter against a regular expression. Mismatches will cause an error
	// to be returned when fetched.
	Validate = func(regex *regexp.Regexp) Option { return func(o *option) { o.validate = regex } }

	// ValidateIntegral validates a parameter as an integral (integer).
	ValidateIntegral = func() Option { return func(o *option) { o.validate = integralRegExp } }

	// integralRegExp is the regular expression used to validate integrals.
	integralRegExp *regexp.Regexp
)

// Option represents options for retrieving values, i.e. setting defaults, required values, adding validation and more.
type Option func(*option)

// option is the internal representation of the set of options for a parameter.
type option struct {
	defaultValue string
	validate     *regexp.Regexp
	require      bool
}

// Pool represents a pool of configuration data, divided into named sections.
type Pool struct {
	params map[string]map[string]string
}

// Section represents a single section of configuration data.
type Section struct {
	name   string
	params map[string]string
}

// Strategy represents a merge strategy, identified by the constants below.
type Strategy uint8

const (
	// Report is the default strategy for merges, it will report an error
	// for parameters that already exist.
	Report Strategy = iota

	// Overwrite strategy for merges overwrites parameters for which new values exist.
	Overwrite

	// Keep strategy for merges will keep existing values for parameters that are
	// given new values.
	Keep
)

// New returns a new configuration pool containing the given sectioned data.
func New(params map[string]map[string]string) *Pool {
	p := Pool{}
	_ = p.Merge(params, Overwrite) // There's no error for Overwrite strategy.
	return &p
}

// Raw returns the entire configuration pool as-is.
func (p *Pool) Raw() map[string]map[string]string {
	return p.params
}

// Section returns the section identified by the given name.
// The parameter ok is false if the section does not exist.
func (p *Pool) Section(name string) (section Section, ok bool) {
	params, ok := p.params[name]
	section = Section{
		name:   name,
		params: params,
	}
	return section, ok
}

// Merge stores the given map of configuration parameters, overriding (by default)
// values that already exist in the pool. The available merge strategies are:
// - Overwrite: an existing key is always overwritten with the new value
// - Keep: an existing key is kept and the new one discarded
// - Report: the merge is aborted with an error on the first conflicting key name
// The default strategy is Report.
func (p *Pool) Merge(params map[string]map[string]string, strategy Strategy) error {
	res, err := merge(p.params, params, strategy)
	if err != nil {
		return err
	}
	p.params = res
	return nil
}

// Unset attempts to remove the given key from the given section.
// You can provide an empty string for the key to remove the entire section.
// It returns true if the key/section was removed, otherwise it returns false.
func (p *Pool) Unset(section, key string) bool {
	sec, ok := p.params[section]
	if !ok {
		return false
	}

	// Removing the section.
	if key == "" {
		delete(p.params, section)
		return true
	}

	// Removing the key.
	_, ok = sec[key]
	if ok {
		delete(p.params[section], key)
	}
	return ok
}

// Compare returns the sections and parameters from the given pool that doesn't
// already exist in the pool that Compare is called from.
// Two pools p1 and p2 are identical if, and only if
// len(p1.Compare(p2)) == 0 && len(p2.Compare(p1)) == 0
func (p *Pool) Compare(pool *Pool) map[string]map[string]string {
	return diff(pool.params, p.params)
}

// String returns the string value for the given key in the current section.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
func (s Section) String(key string, options ...Option) (string, error) {
	val, ok := s.params[key]
	return checkApplyOptions(key, val, ok, options...)
}

// Strings returns the string values for the given key in the given section.
// Separator will be used to split the string into a slice.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
func (s Section) Strings(key, separator string, options ...Option) ([]string, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return nil, err
	}
	ss := strings.Split(val, separator)
	return ss, nil
}

// Int attempts to convert the value for the requested key into an int.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
// A ConversionError is returned if type conversion fails.
func (s Section) Int(key string, options ...Option) (int, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return 0, err
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, ConversionError{key, val, "int"}
	}
	return i, nil
}

// Float attempts to convert the value for the requested key into a float64.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
// A ConversionError is returned if type conversion fails.
func (s Section) Float(key string, options ...Option) (float64, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return 0, err
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, ConversionError{key, val, "float64"}
	}
	return f, nil
}

// Bool attempts to convert the value for the requested key into a bool.
// Acceptable values for truth are: t, true, y, yes and 1.
// Acceptable values for falsehood are: f, false, n, no and 0.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
// A ConversionError is returned if type conversion fails.
func (s Section) Bool(key string, options ...Option) (bool, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return false, err
	}
	switch val {
	case "t", "true", "y", "yes", "on", "1":
		return true, nil
	case "f", "false", "n", "no", "off", "0":
		return false, nil
	}
	return false, ConversionError{key, val, "bool"}
}

// Duration attempts to convert the value for the requested key into a time.Duration.
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
// A ConversionError is returned if type conversion fails.
func (s Section) Duration(key string, options ...Option) (time.Duration, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return 0, err
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, ConversionError{key, val, "Duration"}
	}
	return d, nil
}

// Time attempts to convert the value for the requested key into a time.Time.
// If the time format is omitted, timestamps are parsed as RFC3339 (2006-01-02T15:04:05Z07:00).
// A NoKeyError is returned if the key is required but does not exist.
// A ValidationError is returned if the value didn't validate.
// A ConversionError is returned if type conversion fails.
func (s Section) Time(key, format string, options ...Option) (time.Time, error) {
	val, err := s.String(key, options...)
	if err != nil || val == "" {
		return time.Time{}, err
	}
	if format == "" {
		format = time.RFC3339
	}
	t, err := time.Parse(format, val)
	if err != nil {
		return time.Time{}, ConversionError{key, val, "Time"}
	}
	return t, nil
}

// checkApplyOptions unpacks the given options and checks the given key and value against them.
// checkApplyOptions returns the original value unaltered if validation succeeds, a default value if one was given
// and the key does not exist (ok == false), or an empty string and an error if the key was required but does not
// exist or if value validation failed. Finally, an empty string and a nil error is returned for keys that don't exist
// but are not required and have no default values.
func checkApplyOptions(key, value string, ok bool, options ...Option) (string, error) {
	var opt option
	for _, o := range options {
		o(&opt)
	}

	if value == "" {
		ok = false
	}

check:
	switch {
	case !ok && opt.require:
		return "", NoKeyError(key)
	case !ok && opt.defaultValue != "":
		ok, value = true, opt.defaultValue
		goto check
	case !ok:
		return "", nil
	case opt.validate != nil:
		ok := opt.validate.MatchString(value)
		if !ok {
			return "", ValidationError(key)
		}
	}

	return value, nil
}

// merge two set of parameters, with respect to the provided strategy.
func merge(first, second map[string]map[string]string, strategy Strategy) (map[string]map[string]string, error) {
	fLen, sLen := uint32(len(first)), uint32(len(second))

	// Quickly resolving edge cases.
	if fLen == 0 {
		return second, nil
	}
	if sLen == 0 {
		return first, nil
	}

	longest := fLen
	if sLen > fLen {
		longest = sLen
	}
	res := make(map[string]map[string]string, longest)

	// Copy "first" into the new pool.
	for sec, params := range first {
		res[sec] = make(map[string]string, len(params))
		for field, val := range params {
			res[sec][field] = val
		}
	}

	// Merge "second" into the new pool.
	for sec, params := range second {
		if _, secOK := res[sec]; secOK {
			// Section exists. Mind the merge strategy.
			for key, val := range params {
				if _, fieldOK := res[sec][key]; fieldOK {
					switch strategy {
					case Report:
						return res, fmt.Errorf("section %q, key %q already exists", sec, key)
					case Keep:
						continue
					case Overwrite:
						res[sec][key] = val
					default:
						continue
					}
				} else {
					res[sec][key] = val
				}
			}
		} else {
			// Section does not exist, or the Overwrite strategy is being used.
			// So we just copy values one by one.
			res[sec] = make(map[string]string)
			for field, val := range params {
				res[sec][field] = val
			}
		}
	}

	return res, nil
}

// diff returns all the sections, fields and values that are present in "first",
// but not in "second".
func diff(first, second map[string]map[string]string) map[string]map[string]string {
	res := make(map[string]map[string]string)

	for sec, params := range first {
		var gotSection bool
		if _, ok := second[sec]; !ok {
			// The section is missing, so all parameters are missing as well.
			res[sec] = make(map[string]string)
			for key, val := range first[sec] {
				res[sec][key] = val
			}
		}
		// Section is present, so check key/value pairs one by one.
		for key, val := range params {
			if secVal, ok := second[sec][key]; !ok || (ok && secVal != first[sec][key]) {
				if !gotSection {
					res[sec] = make(map[string]string)
					gotSection = true
				}
				res[sec][key] = val
			}
		}
	}

	return res
}

// MustPrettyPrint returns a string representation of the given configuration
// pool, with section names nested in brackets, and key/value pairs listed
// line-by-line using the given indentation. It panics if it couldn't generate
// a valid string.
// Section names, as well as key names within each section, are sorted
// alphabetically in order to create deterministic and more comparable output.
func MustPrettyPrint(pool map[string]map[string]string, indent string) string {
	var out strings.Builder

	writeString := func(s string) {
		_, err := out.WriteString(s)
		if err != nil {
			panic(err)
		}
	}

	sections := make([]string, 0, len(pool))
	for sec := range pool {
		sections = append(sections, sec)
	}
	sort.Strings(sections)

	for _, sec := range sections {
		params := make([]string, 0, len(pool[sec]))
		for param := range pool[sec] {
			params = append(params, param)
		}
		sort.Strings(params)
		writeString("\n[" + sec + "]\n")
		for _, key := range params {
			writeString(indent + key + ": " + pool[sec][key] + "\n")
		}
	}

	return strings.Trim(out.String(), "\n")
}
