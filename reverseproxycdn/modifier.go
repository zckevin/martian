package reverseproxycdn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/google/martian/v3"
	"github.com/google/martian/v3/parse"
	"github.com/google/martian/v3/verify"
)

var noop = martian.Noop("martianhttp.Modifier")

// Modifier is a locking modifier that is configured via http.Handler.
type Modifier struct {
	mu     sync.RWMutex
	config []byte
	reqmod martian.RequestModifier
	resmod martian.ResponseModifier
}

// NewModifier returns a new martianhttp.Modifier.
func NewModifier(configFilePath string) *Modifier {
	m := &Modifier{}
	m.init(configFilePath)
	return m
}

// SetRequestModifier sets the request modifier.
func (m *Modifier) SetRequestModifier(reqmod martian.RequestModifier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setRequestModifier(reqmod)
}

func (m *Modifier) setRequestModifier(reqmod martian.RequestModifier) {
	if reqmod == nil {
		reqmod = noop
	}

	m.reqmod = reqmod
}

// SetResponseModifier sets the response modifier.
func (m *Modifier) SetResponseModifier(resmod martian.ResponseModifier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setResponseModifier(resmod)
}

func (m *Modifier) setResponseModifier(resmod martian.ResponseModifier) {
	if resmod == nil {
		resmod = noop
	}

	m.resmod = resmod
}

// ModifyRequest runs reqmod.
func (m *Modifier) ModifyRequest(req *http.Request) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.reqmod.ModifyRequest(req)
}

// ModifyResponse runs resmod.
func (m *Modifier) ModifyResponse(res *http.Response) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.resmod.ModifyResponse(res)
}

// VerifyRequests verifies reqmod, iff reqmod is a RequestVerifier.
func (m *Modifier) VerifyRequests() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if reqv, ok := m.reqmod.(verify.RequestVerifier); ok {
		return reqv.VerifyRequests()
	}

	return nil
}

// VerifyResponses verifies resmod, iff resmod is a ResponseVerifier.
func (m *Modifier) VerifyResponses() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if resv, ok := m.resmod.(verify.ResponseVerifier); ok {
		return resv.VerifyResponses()
	}

	return nil
}

// ResetRequestVerifications resets verifications on reqmod, iff reqmod is a
// RequestVerifier.
func (m *Modifier) ResetRequestVerifications() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if reqv, ok := m.reqmod.(verify.RequestVerifier); ok {
		reqv.ResetRequestVerifications()
	}
}

// ResetResponseVerifications resets verifications on resmod, iff resmod is a
// ResponseVerifier.
func (m *Modifier) ResetResponseVerifications() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if resv, ok := m.resmod.(verify.ResponseVerifier); ok {
		resv.ResetResponseVerifications()
	}
}

func (m *Modifier) init(configFilePath string) {
	body, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}
	r, err := parse.FromJSON(body)
	if err != nil {
		panic(fmt.Sprintf("martianhttp: error parsing JSON: %v", err))
	}

	buf := new(bytes.Buffer)
	if err := json.Indent(buf, body, "", "  "); err != nil {
		panic(fmt.Sprintf("martianhttp: error formatting JSON: %v", err))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = buf.Bytes()
	m.setRequestModifier(r.RequestModifier())
	m.setResponseModifier(r.ResponseModifier())
}
