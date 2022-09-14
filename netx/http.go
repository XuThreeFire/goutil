package netutil

import (
	"crypto/tls"
	midutil "github.com/XuThreeFire/goutil/middlewarex"
	httptransport "github.com/go-kit/kit/transport/http"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

// CopyURL get url + path
func CopyURL(base *url.URL, path string) (next *url.URL) {
	n := *base
	n.Path = path
	next = &n
	return
}

var stdHttpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          1500,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	},
}

// DefaultHTTPOptions 默认请求中间件
func DefaultHTTPOptions(logger *zap.Logger, signUser, signKey string, methods []string) map[string][]httptransport.ClientOption {
	options := map[string][]httptransport.ClientOption{}
	for _, method := range methods {
		options[method] = []httptransport.ClientOption{
			httptransport.ClientBefore(midutil.GenerateSignatureToRequest(signUser, signKey, method)),
		}
	}
	//  全部method添加中间件
	// add ContextToHTTPRequest
	addHTTPOptionsToAllMethods(methods, options, httptransport.ClientBefore(midutil.ContextToHTTPRequest()))
	// add stdHttpClient
	addHTTPOptionsToAllMethods(methods, options, httptransport.SetClient(stdHttpClient))
	return options
}

func addHTTPOptionsToAllMethods(methods []string, options map[string][]httptransport.ClientOption, opt httptransport.ClientOption) {
	for _, v := range methods {
		options[v] = append(options[v], opt)
	}
}
