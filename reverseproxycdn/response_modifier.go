package reverseproxycdn

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/martian/v3/parse"
	reqresprewriter "github.com/zckevin/reverse-proxy-cdn/reqresp-rewriter"
)

func init() {
	parse.Register("reverseproxycdn.response.Modifier", responseModifierFromJSON)
}

type ResponseModifier struct {
	rewriter reqresprewriter.ReqrespRewriter
}

type ResponseModifierJSON struct {
	BaseDomain string               `json:"base_domain"`
	Scope      []parse.ModifierType `json:"scope"`
}

func (m *ResponseModifier) ModifyResponse(resp *http.Response) error {
	// bypass for surfly.io's bootstrap static js scripts
	if strings.HasPrefix(resp.Request.Host, "local.host") {
		return nil
	}
	if err := m.rewriter.RewriteHTTPResponse(resp); err != nil {
		return err
	}
	return nil
}

func responseModifierFromJSON(b []byte) (*parse.Result, error) {
	msg := &ResponseModifierJSON{}
	if err := json.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	modifier := &ResponseModifier{
		rewriter: reqresprewriter.NewReqrespRewriterFromBaseDomain(msg.BaseDomain),
	}
	return parse.NewResult(modifier, msg.Scope)
}
