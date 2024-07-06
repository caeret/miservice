package miservice

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

type Mina struct {
	account *Account
}

func NewMina(account *Account) *Mina {
	return &Mina{account: account}
}

func (m *Mina) request(ctx context.Context, path string, data map[string]any) (ret []byte, err error) {
	requestID := "app_ios_" + generateRandomStr(30)
	if data != nil {
		data["requestId"] = requestID
	} else {
		path += "&requestId=" + requestID
	}

	headers := map[string]string{"User-Agent": "MiHome/6.0.103 (com.xiaomi.mihome; build:6.0.103.1; iOS 14.4.0) Alamofire/6.0.103 MICO/iOSApp/appStore/6.0.103"}

	return m.account.Request(ctx, "micoapi", "https://api2.mina.mi.com"+path, DataMap(data), headers, true)
}

func (m *Mina) ListDevices(ctx context.Context, master int) (ret []byte, err error) {
	ret, err = m.request(ctx, "/admin/v2/device_list?master="+cast.ToString(master), nil)
	if err != nil {
		return
	}

	ret = []byte(jsoniter.Get(ret, "data").ToString())
	return
}
