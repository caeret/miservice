package miservice

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type Token struct {
	DeviceID  string
	UserID    string
	PassToken string
	Tokens    map[string][]string
}

func NewToken() *Token {
	return &Token{Tokens: make(map[string][]string)}
}

func (t *Token) reset() {
	t.DeviceID = ""
	t.UserID = ""
	t.PassToken = ""
	t.Tokens = make(map[string][]string)
}

type Account struct {
	username string
	password string
	client   *http.Client
	token    *Token
}

func NewAccount(username, password string, token *Token) (*Account, error) {
	if token == nil {
		token = NewToken()
	}
	return &Account{username: username, password: password, client: http.DefaultClient, token: token}, nil
}

func (a *Account) Login(ctx context.Context, sid string) (err error) {
	if a.token.DeviceID == "" {
		a.token.DeviceID = strings.ToUpper(generateRandomStr(16))
	}
	ret, err := a.serviceLogin(ctx, fmt.Sprintf("serviceLogin?sid=%s&_json=true", sid), nil)
	if err != nil {
		return
	}

	j := jsoniter.Get(ret)
	if j.Get("code").ToString() != "0" {
		data := make(url.Values)
		for k, v := range map[string]string{
			"_json":    "true",
			"qs":       j.Get("qs").ToString(),
			"sid":      j.Get("sid").ToString(),
			"_sign":    j.Get("_sign").ToString(),
			"callback": j.Get("callback").ToString(),
			"user":     a.username,
			"hash":     strings.ToUpper(md5Hash(a.password)),
		} {
			data.Set(k, v)
		}
		ret, err = a.serviceLogin(ctx, "serviceLoginAuth2", data)
		if err != nil {
			return
		}
		j = jsoniter.Get(ret)
		if j.Get("code").ToString() != "0" {
			err = fmt.Errorf("login failed: %s", string(ret))
			return
		}
	}

	a.token.PassToken = j.Get("passToken").ToString()
	a.token.UserID = j.Get("userId").ToString()

	var serviceToken string
	serviceToken, err = a.securityTokenService(ctx, j.Get("location").ToString(), j.Get("nonce").ToString(), j.Get("ssecurity").ToString())
	if err != nil {
		return
	}
	a.token.Tokens[sid] = []string{j.Get("ssecurity").ToString(), serviceToken}

	return nil
}

func (a *Account) Token() Token {
	return *a.token
}

func (a *Account) serviceLogin(ctx context.Context, path string, data url.Values) (b []byte, err error) {
	var (
		method = http.MethodGet
		body   io.Reader
	)
	if data != nil {
		method = http.MethodPost
		body = strings.NewReader(data.Encode())
	}

	URL, err := url.Parse("https://account.xiaomi.com/pass/" + path)
	if err != nil {
		return
	}

	cookies := Cookies{
		"sdkVersion": "3.9",
		"deviceId":   a.token.DeviceID,
	}
	if passToken := a.token.PassToken; passToken != "" {
		cookies["passToken"] = passToken
		cookies["userId"] = a.token.UserID
	}

	req, err := http.NewRequestWithContext(ctx, method, URL.String(), body)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "APP/com.xiaomi.mihome APPV/6.0.103 iosPassportSDK/3.9.0 iOS/14.4 miHSTS")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	cookies.Write(req)
	resp, err := a.client.Do(req)
	if err != nil {
		return
	}

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if len(b) > 11 {
		b = b[11:]
	}

	return
}

func (a *Account) securityTokenService(ctx context.Context, location, nonce, ssecurity string) (string, error) {
	// Construct nsec string
	nsec := "nonce=" + nonce + "&" + ssecurity

	// Generate clientSign
	h := sha1.New()
	h.Write([]byte(nsec))
	clientSign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Make HTTP GET request
	reqURL := location + "&clientSign=" + url.QueryEscape(clientSign)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "APP/com.xiaomi.mihome APPV/6.0.103 iosPassportSDK/3.9.0 iOS/14.4 miHSTS")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract serviceToken from cookies
	var serviceToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "serviceToken" {
			serviceToken = cookie.Value
			break
		}
	}

	if serviceToken == "" {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("serviceToken not found: %s", string(body))
	}

	return serviceToken, nil
}

func (a *Account) Request(ctx context.Context, sid, URI string, data Data, headers map[string]string, tryLogin bool) (ret []byte, err error) {
	if len(a.token.Tokens[sid]) == 0 || a.token.Tokens[sid][0] == "" {
		err = a.Login(ctx, sid)
		if err != nil {
			return
		}
	}

	var (
		method = http.MethodGet
		body   io.Reader
	)

	cookieKV := Cookies{
		"serviceToken": a.token.Tokens[sid][1],
		"userId":       a.token.UserID,
	}

	if data != nil {
		var params map[string]any
		params, err = data.Parse(a.token, cookieKV)
		if err != nil {
			return
		}
		if params != nil {
			method = http.MethodPost
			values := url.Values{}
			for k, v := range params {
				values.Set(k, fmt.Sprint(v))
			}
			body = strings.NewReader(values.Encode())
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, URI, body)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "APP/com.xiaomi.mihome APPV/6.0.103 iosPassportSDK/3.9.0 iOS/14.4 miHSTS")
	cookieKV.Write(req)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	ret, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	status := resp.StatusCode

	switch status {
	case http.StatusOK:
		if jsoniter.Get(ret, "code").ToString() == "0" {
			return
		}
		if strings.Contains(strings.ToLower(jsoniter.Get(ret, "message").ToString()), "auth") {
			status = http.StatusUnauthorized
		}
	}

	if status == http.StatusUnauthorized && tryLogin {
		a.token.reset()
		return a.Request(ctx, sid, URI, data, headers, false)
	}

	err = fmt.Errorf("request failed: %s", string(ret))
	return
}
