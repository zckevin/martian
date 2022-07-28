package reverseproxycdn

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/martian/v3/parse"
	reqresprewriter "github.com/zckevin/reverse-proxy-cdn/reqresp-rewriter"
)

func init() {
	parse.Register("reverseproxycdn.request.Modifier", requestModifierFromJSON)
}

type RequestModifier struct {
	rewriter reqresprewriter.ReqrespRewriter
}

type requestModifierJSON struct {
	BaseDomain string               `json:"base_domain"`
	Scope      []parse.ModifierType `json:"scope"`
}

func (m *RequestModifier) ModifyRequest(req *http.Request) error {
	// bypass for surfly.io's bootstrap static js scripts
	if strings.HasPrefix(req.Host, "local.host") {
		return nil
	}
	if err := m.rewriter.RewriteHTTPRequest(req); err != nil {
		return err
	}
	return nil
}

func requestModifierFromJSON(b []byte) (*parse.Result, error) {
	msg := &requestModifierJSON{}
	if err := json.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	mod := &RequestModifier{
		rewriter: reqresprewriter.NewReqrespRewriterFromBaseDomain(msg.BaseDomain),
	}
	return parse.NewResult(mod, msg.Scope)
}
