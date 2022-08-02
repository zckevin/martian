package reverseproxycdn

import (
	"encoding/json"
	"net/http"

	"github.com/google/martian/v3"
	"github.com/google/martian/v3/parse"
)

func init() {
	parse.Register("reverseproxycdn.request.Modifier", requestModifierFromJSON)
}

type RequestModifier struct{}

type requestModifierJSON struct {
	Scope []parse.ModifierType `json:"scope"`
}

func (m *RequestModifier) ModifyRequest(req *http.Request) error {
	// just leave CONNECT request alone
	if req.Method == "CONNECT" {
		return nil
	}

	resp, err := rewriter.HijackHTTPRequest(req)
	if err != nil {
		return err
	}
	if resp != nil {
		ctx := martian.NewContext(req)
		_, brw, err := ctx.Session().Hijack()
		if err != nil {
			return err
		}
		err = resp.Write(brw)
		if err != nil {
			return err
		}
		return brw.Flush()
	}

	if err := rewriter.RewriteHTTPRequest(req); err != nil {
		return err
	}
	return nil
}

func requestModifierFromJSON(b []byte) (*parse.Result, error) {
	msg := &requestModifierJSON{}
	if err := json.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	mod := &RequestModifier{}
	return parse.NewResult(mod, msg.Scope)
}
