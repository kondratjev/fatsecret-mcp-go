package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/httpx"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// OAuth1 credentials for the 3-legged profile flow.
type OAuth1Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

const (
	oauth1RequestTokenURL = "https://authentication.fatsecret.com/oauth/request_token"
	oauth1AuthorizeURL    = "https://authentication.fatsecret.com/oauth/authorize"
	oauth1AccessTokenURL  = "https://authentication.fatsecret.com/oauth/access_token"
)

// percentEncode is RFC 3986 encoding (encodeURIComponent + !'()*).
func percentEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func nonce() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf[:])
}

func buildBaseString(method, rawURL string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.ToUpper(method) + "&" + percentEncode(rawURL) + "&" + percentEncode(strings.Join(pairs, "&"))
}

func signBaseString(baseString, consumerSecret, tokenSecret string) string {
	key := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(baseString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildOAuth1Params signs method+url with the credentials; extra params join the signature.
func BuildOAuth1Params(method, rawURL string, creds OAuth1Credentials, extra map[string]string) map[string]string {
	oauth := map[string]string{
		"oauth_consumer_key":     creds.ConsumerKey,
		"oauth_nonce":            nonce(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        fmt.Sprint(time.Now().Unix()),
		"oauth_version":          "1.0",
	}
	if creds.AccessToken != "" {
		oauth["oauth_token"] = creds.AccessToken
	}
	all := map[string]string{}
	for k, v := range oauth {
		all[k] = v
	}
	for k, v := range extra {
		all[k] = v
	}
	oauth["oauth_signature"] = signBaseString(buildBaseString(method, rawURL, all), creds.ConsumerSecret, creds.AccessTokenSecret)
	for k, v := range extra {
		oauth[k] = v
	}
	return oauth
}

func signedURL(method, rawURL string, creds OAuth1Credentials, extra map[string]string) string {
	params := BuildOAuth1Params(method, rawURL, creds, extra)
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	return rawURL + "?" + vals.Encode()
}

func readOAuth1Result(resp *http.Response) (token, secret string, err error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("OAuth1 request failed: %d %s", resp.StatusCode, string(body))
	}
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return "", "", fmt.Errorf("invalid OAuth1 response: %s", string(body))
	}
	token, secret = vals.Get("oauth_token"), vals.Get("oauth_token_secret")
	if token == "" || secret == "" {
		return "", "", fmt.Errorf("invalid OAuth1 response: %s", string(body))
	}
	return token, secret, nil
}

// requestToken starts the 3-legged flow. Docs say POST; GET works as fallback,
// so POST is tried first with a GET fallback.
func RequestToken(creds OAuth1Credentials) (token, secret, authURL string, err error) {
	basic := OAuth1Credentials{ConsumerKey: creds.ConsumerKey, ConsumerSecret: creds.ConsumerSecret}
	postReq, _ := http.NewRequest(http.MethodPost, signedURL(http.MethodPost, oauth1RequestTokenURL, basic, map[string]string{"oauth_callback": "oob"}), nil)
	if resp, rerr := httpx.ResilientFetch(postReq); rerr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		token, secret, err = readOAuth1Result(resp)
		if err == nil {
			return token, secret, oauth1AuthorizeURL + "?oauth_token=" + url.QueryEscape(token), nil
		}
	} else if resp != nil {
		resp.Body.Close()
	}
	getReq, _ := http.NewRequest(http.MethodGet, signedURL(http.MethodGet, oauth1RequestTokenURL, basic, map[string]string{"oauth_callback": "oob"}), nil)
	resp, rerr := httpx.ResilientFetch(getReq)
	if rerr != nil {
		return "", "", "", rerr
	}
	token, secret, err = readOAuth1Result(resp)
	if err != nil {
		return "", "", "", err
	}
	return token, secret, oauth1AuthorizeURL + "?oauth_token=" + url.QueryEscape(token), nil
}

func AccessToken(creds OAuth1Credentials, oauthToken, oauthTokenSecret, verifier string) (string, string, error) {
	tokenCreds := creds
	tokenCreds.AccessToken = oauthToken
	tokenCreds.AccessTokenSecret = oauthTokenSecret
	req, _ := http.NewRequest(http.MethodGet, signedURL(http.MethodGet, oauth1AccessTokenURL, tokenCreds, map[string]string{"oauth_verifier": verifier}), nil)
	resp, err := httpx.ResilientFetch(req)
	if err != nil {
		return "", "", err
	}
	return readOAuth1Result(resp)
}
