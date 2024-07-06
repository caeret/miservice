package miservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func GetSpecTypes(ctx context.Context) (types map[string]string, err error) {
	specPath := filepath.Join(os.TempDir(), "miservice_miot_specs.json")
	var b []byte
	b, err = os.ReadFile(specPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://miot-spec.org/miot-spec-v2/instances?status=all", nil)
		if err != nil {
			return
		}
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		b, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = os.WriteFile(specPath, b, 0o644)
		if err != nil {
			return
		}
	}

	err = json.Unmarshal(b, &types)

	return
}

func GetSpec(ctx context.Context, typ string) (b []byte, err error) {
	if !strings.HasPrefix(typ, "urn") {
		var types map[string]string
		types, err = GetSpecTypes(ctx)
		if err != nil {
			return
		}
		v, ok := types[typ]
		if !ok {
			for k := range types {
				if strings.Contains(k, typ) {
					v = types[k]
				}
			}
		}
		if v == "" {
			err = fmt.Errorf("unknown type: %s", typ)
			return
		}
		typ = v
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://miot-spec.org/miot-spec-v2/instance?type="+typ, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}
