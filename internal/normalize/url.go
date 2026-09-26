package normalize

import (
	"net/url"
	"strings"
)

var trackingParams = map[string]bool{
	"utm_source": true, "utm_medium": true, "utm_campaign": true,
	"utm_term": true, "utm_content": true, "utm_id": true,
	"fbclid": true, "gclid": true, "dclid": true, "msclkid": true,
	"mc_cid": true, "mc_eid": true, "_ga": true, "ref": true,
	"referrer": true, "source": true, "campaign": true,
	"igshid": true, "mkt_tok": true, "oly_enc_id": true,
}

func CanonicalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	if host, port, ok := splitHostPort(u.Host); ok {
		if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
			u.Host = host
		}
	}
	if u.RawQuery != "" {
		vals := u.Query()
		for k := range vals {
			if trackingParams[strings.ToLower(k)] {
				vals.Del(k)
			}
		}
		u.RawQuery = vals.Encode()
	}
	if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

func splitHostPort(host string) (string, string, bool) {
	idx := strings.LastIndex(host, ":")
	if idx < 0 || idx == len(host)-1 {
		return host, "", false
	}
	return host[:idx], host[idx+1:], true
}
