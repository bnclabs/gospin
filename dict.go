package failsafe

import (
	"encoding/json"
	"fmt"
	"github.com/prataprc/go-jsonpointer"
	"sync"
)

// error codes

// ErrorInvalidPath
var ErrorInvalidPath = fmt.Errorf("errorInvalidPath")

// ErrorInvalidType
var ErrorInvalidType = fmt.Errorf("errorInvalidType")

// ErrorInvalidCAS
var ErrorInvalidCAS = fmt.Errorf("errorInvalidCAS")

const nullCAS = float64(0)

// SafeDict is a failsafe data-structure similar to JSON property.
type SafeDict struct {
	mu  sync.Mutex             `json:"-"`
	m   map[string]interface{} `json:"m"`   // JSON decoded data-structure
	CAS float64                `json:"CAS"` // monotonically increasing CAS
}

// NewSafeDict returns a reference to new failsafe dictionary. Can be
// initialized to support CAS and/or initialized with initial JSON encoded
// string or JSON decode map[string]interface{} data.
func NewSafeDict(data interface{}, cas bool) (*SafeDict, error) {
	sd := &SafeDict{}
	sd.m = make(map[string]interface{})

	switch arg := data.(type) {
	case map[string]interface{}:
		sd.m = arg

	case []byte:
		if arg != nil {
			if err := json.Unmarshal(arg, &sd.m); err != nil {
				return nil, err
			}
		}
	}
	if cas {
		sd.CAS = float64(1)
	}
	return sd, nil
}

// MarshalJSON implements encoding/json.Marshaler interface.
func (sd *SafeDict) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		M   map[string]interface{} `json:"m"`
		CAS float64                `json:"CAS"`
	}{sd.m, sd.CAS})
}

// UnmarshalJSON implements encoding/json.Unmarshaler interface.
func (sd *SafeDict) UnmarshalJSON(data []byte) error {
	t := struct {
		M   map[string]interface{} `json:"m"`
		CAS float64                `json:"CAS"`
	}{}

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	sd.m = t.M
	sd.CAS = t.CAS
	return nil
}

// GetCAS returns the current CAS value.
func (sd *SafeDict) GetCAS() float64 {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.CAS
}

// Get field value located by `path` jsonpointer, full json-pointer spec is
// allowed.
func (sd *SafeDict) Get(path string) (rv interface{}, CAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	rv = jsonpointer.Get(sd.m, path)
	if rv == nil {
		return nil, nullCAS, ErrorInvalidPath
	}
	return rv, sd.CAS, nil
}

// Set value at the specified path, full json-pointer spec. is allowed. If CAS
// is specified as nullCAS, CAS is ignored.
func (sd *SafeDict) Set(path string, value interface{}, CAS float64) (nextCAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.checkCAS(CAS) == false {
		return nullCAS, ErrorInvalidCAS
	}

	switch path {
	case "":
		if m, ok := value.(map[string]interface{}); ok {
			sd.m = m
			return sd.incrementCAS(), nil
		}
		err = ErrorInvalidType

	default:
		if err = jsonpointer.Set(sd.m, path, value); err == nil {
			return sd.incrementCAS(), nil
		}
	}
	return nullCAS, err
}

// Delete value at the specified path, last segment shall always index
// into json property. If CAS is specied as nullCAS, CAS is ignored.
func (sd *SafeDict) Delete(path string, CAS float64) (nextCAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.checkCAS(CAS) == false {
		return nullCAS, ErrorInvalidCAS
	}

	switch path {
	case "":
		sd.m = nil
		return sd.incrementCAS(), nil

	default:
		if err = jsonpointer.Delete(sd.m, path); err == nil {
			return sd.incrementCAS(), nil
		}
	}
	return nullCAS, err
}

// Save implements raft.StateMachine interface.
func (sd *SafeDict) Save() (data []byte, err error) {
	return json.Marshal(sd)
}

// Restore implements raft.StateMachine interface.
func (sd *SafeDict) Recovery(data []byte) (err error) {
	return json.Unmarshal(data, &sd)
}

// monotonically increasing CAS.
func (sd *SafeDict) incrementCAS() float64 {
	if sd.CAS != nullCAS {
		sd.CAS += 1.0
	}
	return sd.CAS
}

// compare local CAS with API supplied CAS, provided API supplied CAS is not
// nullCAS.
func (sd *SafeDict) checkCAS(CAS float64) bool {
	switch CAS {
	case nullCAS:
		return true
	case sd.CAS:
		return true
	}
	return false
}
