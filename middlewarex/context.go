package midutil

type contextKey string

const (
	// ContextKeyRequestTraceID Trace-Id `uuid` 无`-` 分割格式，模块交互可一直传递使用
	ContextKeyRequestTraceID contextKey = "Trace-Id"

	// ContextKeyRequestXRequestID X-Request-Id
	ContextKeyRequestXRequestID contextKey = "X-Request-Id"

	// ContextKeyRequestAuthUser AuthUser 授权用户(服务端预分配用于签名校验的用户)
	ContextKeyRequestAuthUser contextKey = "Auth-User"

	// ContextKeyRequestMethod Method 	请求方法
	ContextKeyRequestMethod contextKey = "Method"

	// ContextKeyRequestTimestamp Timestamp 请求时间戳(unix timestamp 含毫秒）
	ContextKeyRequestTimestamp contextKey = "Timestamp"

	// ContextKeyRequestSignature Signature 签名
	ContextKeyRequestSignature contextKey = "Signature"

	// ContextKeyCalculateSignature CalculateSignature 计算签名
	ContextKeyCalculateSignature contextKey = "CalculateSignature"

	// ContextKeyInterfaceKey InterfaceKey 授权的Key
	ContextKeyInterfaceKey contextKey = "InterfaceKey"

	// ContextKeyInterfaceKey PartnerId 授权的partnerId
	ContextKeyPartnerId contextKey = "PartnerId"
	ContextKeyUserName  contextKey = "UserName"
	ContextKeyUserKey   contextKey = "UserKey"
	ContextKeyTraceId   contextKey = "TraceId"
)
