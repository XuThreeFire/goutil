package errorutil

var (
	ErrParseError       = New(101, "数据解析失败", false)
	ErrSignatureError   = New(102, "签名验证错误", false)
	ErrIllegaUser       = New(103, "未授权用户", false)
	ErrDataError        = New(104, "数据型数据解析失败", false)
	ErrEmptyParam       = New(105, "请求数据为空", false)
	ErrExpiredSignature = New(106, "签名过期", false)
	ErrInternalError    = New(107, "内部错误", false)
	ErrIllegalRequest   = New(108, "非法请求", false)
	ErrNotFound         = New(109, "未找到对应记录", false)
	ErrIllegalData      = New(110, "信息有误或不完整", false)
)
