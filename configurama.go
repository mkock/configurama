package configurama

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// Pool represents a pool of configuration data, divided into named sections.
type Pool struct {
	params map[string]map[string]string
}

// Strategy represents a merge strategy, identified by the consts below.
type Strategy uint8

const (
	// Report is the default strategy, it will report an error
	//for parameters that already exist.
	Report Strategy = iota

	// The Overwrite strategy overwrites parameters for which new values exist.
	Overwrite

	// The Keep strategy will keep existing values for parameters that are
	// given new values.
	Keep
)

// New returns a new configuration pool containing the given sectioned data.
func New(params map[string]map[string]string) *Pool {
	p := Pool{}
	_ = p.Merge(params, Overwrite) // There's no error for the Overwrite strategy.
	return &p
}

// ExtractWithHooks attempts to extract the values from the section with the
// given name into the given struct (which must be passed by reference) using
// the given name prefix (or an empty string in case of no prefix) to match
// keys to the struct's field names.
// If given, the functions "pre" and "post" are called just before and after
// extraction, respectively. Use "pre" to validate data and set defaults by
// changing the contents of the map. The "post" function is suitable for running
// any code that should run after extraction. Inside the "post" function, you can
// safely assume that the struct has already been filled with parameter data.
// If you return an error in the "pre" function, ExtractWithHooks will return
// this error without extracting any parameters.
func (p *Pool) ExtractWithHooks(section, prefix string, out interface{}, pre func(map[string]string) error, post func()) error {
	params, err := p.extractParams(section, prefix)
	if err != nil {
		return err
	}

	// Run the "pre" hook.
	if pre != nil {
		err = pre(params)
		if err != nil {
			return err
		}
	}

	if err = decodeParams(params, out); err != nil {
		return err
	}

	// Run the "post" hook.
	if post != nil {
		post()
	}

	return nil
}

// Raw returns the entire configuration pool as-is.
func (p *Pool) Raw() map[string]map[string]string {
	return p.params
}

// Extract attempts to populate the given struct with configuration data from
// the section with the given name. Names are matched in a fuzzy manner, so for
// example, all of these names will be matched to the field MySQL:
// "mysql", "MySQL", "mySQL" and "my_sql".
func (p *Pool) Extract(section, prefix string, out interface{}) error {
	params, err := p.extractParams(section, prefix)
	if err == nil {
		err = decodeParams(params, out)
	}
	return err
}

// extractParams extracts all the parameters from the section with the given
// name, if it exists, using the given prefix to match keys with struct fields.
func (p *Pool) extractParams(section, prefix string) (map[string]string, error) {
	params, ok := p.params[section]
	if !ok {
		return params, errors.New("unknown section name: " + section)
	}
	if prefix != "" {
		tmp := make(map[string]string, len(params))
		for key, val := range params {
			if strings.HasPrefix(key, prefix) {
				tmp[strings.TrimPrefix(key, prefix)] = val
			}
		}
		params = tmp
	}
	return params, nil
}

// decodeParams attempts to fill the given struct "out" with values from the
// given map. "out" must be passed by reference.
func decodeParams(params map[string]string, out interface{}) error {
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &out,
	}
	dec, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return dec.Decode(params)
}

// Get returns the value for the given key in the given section.
func (p *Pool) Get(section, key string) (val string, ok bool) {
	params, ok := p.params[section]
	if !ok {
		return
	}
	val, ok = params[key]
	return
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
func (p *Pool) Compare(pool Pool) map[string]map[string]string {
	return diff(pool.params, p.params)
}

// merge merges two set of parameters, with respect to the provided strategy.
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
