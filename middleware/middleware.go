package midutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
)

// DecodeHttpResponseJson DecodeHttpResponseJson
func DecodeHttpResponseJson(resp interface{}) func(_ context.Context, r *http.Response) (interface{}, error) {
	return func(_ context.Context, r *http.Response) (interface{}, error) {
		if r.StatusCode != http.StatusOK {
			return nil, errors.New(r.Status)
		}
		tempResp := reflect.New(reflect.TypeOf(resp).Elem()).Interface()
		err := json.NewDecoder(r.Body).Decode(&tempResp)
		return tempResp, err
	}
}

// EncodeHttpRequestGeneric EncodeHttpRequestGeneric
func EncodeHttpRequestGeneric(ctx context.Context, r *http.Request, request interface{}) error {
	if r.Method == "GET" {
		req := request.(map[string]interface{})
		vars := r.URL.Query()
		for k, v := range req {
			vars.Add(k, fmt.Sprintf("%v", v))
		}
		r.URL.RawQuery = vars.Encode()
	} else {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(request); err != nil {
			return err
		}
		r.Body = ioutil.NopCloser(&buf)
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	return nil
}
