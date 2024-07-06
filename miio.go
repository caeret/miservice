package miservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type MiIO struct {
	account *Account
	server  string
}

func NewMiIO(account *Account, region string) *MiIO {
	s := &MiIO{
		account: account,
	}
	if region == "" || region == "cn" {
		s.server = "https://api.io.mi.com/app"
	} else {
		s.server = "https://" + region + ".api.io.mi.com/app"
	}
	return s
}

func (m *MiIO) SendAction(ctx context.Context, did string, iid Tuple2[int, int], args ...any) (err error) {
	ret, err := m.miotRequest(ctx, "action", map[string]any{"did": did, "siid": iid.A, "aiid": iid.B, "in": args})
	if err != nil {
		return
	}
	if jsoniter.Get(ret, "code").ToString() != "0" {
		err = fmt.Errorf("action failed: %s", string(ret))
		return
	}
	return
}

func (m *MiIO) ListDevices(ctx context.Context, getVirtualModel bool, getHuamiDevices int) (list []Device, err error) {
	ret, err := m.miioRequest(ctx, "/home/device_list", map[string]any{"getVirtualModel": getVirtualModel, "getHuamiDevices": getHuamiDevices})
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(jsoniter.Get(ret, "list").ToString()), &list)
	return
}

func (m *MiIO) GetMiotProps(ctx context.Context, did string, props []Tuple2[int, int]) (values []any, err error) {
	params := make([]map[string]any, 0)
	for _, prop := range props {
		params = append(params, map[string]any{"did": did, "siid": prop.A, "piid": prop.B})
	}
	ret, err := m.miotRequest(ctx, "prop/get", params)
	if err != nil {
		return
	}

	for i := 0; i < jsoniter.Get(ret).Size(); i++ {
		if jsoniter.Get(ret, i, "code").ToString() == "0" {
			values = append(values, jsoniter.Get(ret, i, "value").GetInterface())
		} else {
			values = append(values, nil)
		}
	}

	return
}

func (m *MiIO) SetMiotProps(ctx context.Context, did string, props []Tuple3[int, int, any]) (values []any, err error) {
	params := make([]map[string]any, 0)
	for _, prop := range props {
		params = append(params, map[string]any{"did": did, "siid": prop.A, "piid": prop.B, "value": prop.C})
	}
	ret, err := m.miotRequest(ctx, "prop/set", params)
	if err != nil {
		return
	}

	for i := 0; i < jsoniter.Get(ret).Size(); i++ {
		values = append(values, jsoniter.Get(ret, i, "code").ToString())
	}

	return
}

func (m *MiIO) GetHomeProps(ctx context.Context, did string, props []string) ([]byte, error) {
	return m.homeRequest(ctx, did, "get_prop", props)
}

func (m *MiIO) SetHomeProp(ctx context.Context, did string, prop string, value any) ([]byte, error) {
	T := reflect.TypeOf(value)
	if T.Kind() != reflect.Slice {
		s := reflect.MakeSlice(reflect.SliceOf(T), 0, 0)
		reflect.AppendSlice(s, reflect.ValueOf(value))
		value = s.Interface()
	}
	return m.homeRequest(ctx, did, "set_"+prop, value)
}

func (m *MiIO) homeRequest(ctx context.Context, did, method string, params any) ([]byte, error) {
	return m.miioRequest(ctx, "/home/rpc/"+did, map[string]any{"id": 1, "method": method, "accessKey": "IOS00026747c5acafc2", "params": params})
}

func (m *MiIO) miotRequest(ctx context.Context, cmd string, params any) ([]byte, error) {
	return m.miioRequest(ctx, "/miotspec/"+cmd, map[string]any{"params": params})
}

func (m *MiIO) miioRequest(ctx context.Context, path string, data map[string]any) (ret []byte, err error) {
	headers := map[string]string{
		"User-Agent":                 "iOS-14.4-6.0.103-iPhone12,3--D7744744F7AF32F0544445285880DD63E47D9BE9-8816080-84A3F44E137B71AE-iPhone",
		"x-xiaomi-protocal-flag-cli": "PROTOCAL-HTTP2",
	}
	fn := DataFunc(func(token *Token, cookies Cookies) (map[string]any, error) {
		cookies["PassportDeviceId"] = token.DeviceID
		return m.signData(path, data, token.Tokens["xiaomiio"][0])
	})
	ret, err = m.account.Request(ctx, "xiaomiio", m.server+path, fn, headers, true)
	if err != nil {
		return
	}
	j := jsoniter.Get(ret, "result")
	if j.ValueType() == jsoniter.InvalidValue {
		err = fmt.Errorf("result not found => %s: %s", path, string(ret))
		return
	}
	ret = []byte(j.ToString())
	return
}

func (m *MiIO) signData(path string, data map[string]any, ssecurity string) (map[string]any, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	nonce, err := generateNonce()
	if err != nil {
		return nil, err
	}

	snonce, err := signNonce(ssecurity, nonce)
	if err != nil {
		return nil, err
	}

	msg := strings.Join([]string{path, snonce, nonce, "data=" + string(b)}, "&")

	key, err := base64.StdEncoding.DecodeString(snonce)
	if err != nil {
		return nil, err
	}
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	sign := h.Sum(nil)

	return map[string]any{
		"_nonce":    nonce,
		"data":      string(b),
		"signature": base64.StdEncoding.EncodeToString(sign),
	}, nil
}
