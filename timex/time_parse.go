package timeutil

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

var cst *time.Location

// CSTLayout China Standard Time Layout
const (
	CSTLayout  = "2006-01-02 15:04:05"
	DATELayout = "2006-01-02"
	TIMELayout = "2006-01-02 15:04"
)

func init() {
	var err error
	if cst, err = time.LoadLocation("Asia/Shanghai"); err != nil {
		panic(err)
	}
}

// GetTimeLocal return *time.Location
func GetTimeLocal() *time.Location {
	return cst
}

// RFC3339ToCSTLayout convert rfc3339 value to china standard timex layout
// 2020-11-08T08:18:46+08:00 => 2020-11-08 08:18:46
func RFC3339ToCSTLayout(value string) (string, error) {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", err
	}

	return ts.In(cst).Format(CSTLayout), nil
}

// CSTLayoutString 格式化时间
// 返回 "2006-01-02 15:04:05" 格式的时间
func CSTLayoutString() string {
	ts := time.Now()
	return ts.In(cst).Format(CSTLayout)
}

// CSTLayoutTimeString 格式化传入时间
// 返回 "2006-01-02 15:04:05" 格式的时间
func CSTLayoutTimeString(ts time.Time) string {
	return ts.In(cst).Format(CSTLayout)
}

// LayoutString 格式化时间
// 返回 自定义 格式的时间
func LayoutString(layout string) string {
	ts := time.Now()
	return ts.In(cst).Format(layout)
}

// LayoutTime 格式化传入时间
// 返回 自定义 格式的时间
func LayoutTime(ts time.Time, layout string) string {
	return ts.In(cst).Format(layout)
}

// ParseCSTInLocation 格式化时间
func ParseCSTInLocation(date string) (time.Time, error) {
	return time.ParseInLocation(CSTLayout, date, cst)
}

// CSTLayoutStringToUnix 返回 unix 时间戳
// 2020-01-24 21:11:11 => 1579871471
func CSTLayoutStringToUnix(cstLayoutString string) (int64, error) {
	stamp, err := time.ParseInLocation(CSTLayout, cstLayoutString, cst)
	if err != nil {
		return 0, err
	}
	return stamp.Unix(), nil
}

// GMTLayoutString 格式化时间
// 返回 "Mon, 02 Jan 2006 15:04:05 GMT" 格式的时间
func GMTLayoutString() string {
	return time.Now().In(cst).Format(http.TimeFormat)
}

// ParseGMTInLocation 格式化时间
func ParseGMTInLocation(date string) (time.Time, error) {
	return time.ParseInLocation(http.TimeFormat, date, cst)
}

// ParseInLocationLayout 格式化时间 自定义格式
func ParseInLocationLayout(date, layout string) (time.Time, error) {
	return time.ParseInLocation(layout, date, cst)
}

// TransferLayout 解析并格式化时间 自定义格式
func TransferLayout(date, layoutIn, layoutOut string) (string, error) {
	ts, err := time.ParseInLocation(layoutIn, date, cst)
	if err != nil {
		return date, err
	}
	str := ts.In(cst).Format(layoutOut)
	return str, nil
}

// SubInLocation 计算时间差
func SubInLocation(ts time.Time) float64 {
	return math.Abs(time.Now().In(cst).Sub(ts).Seconds())
}

// GetTimeNow 获取当前时间
func GetTimeNow() time.Time {
	return time.Now().In(cst)
}

// GetDateNow 获取当前年月日
func GetDateNow() (int, time.Month, int) {
	return time.Now().In(cst).Date()
}

// GetTimeStamp 获取当前时间戳 毫秒
func GetTimeStamp() string {
	return strconv.FormatInt(time.Now().In(cst).UnixNano()/1e6, 10)
}

// GetDiffNowDate 获取和当前日期相隔的天数
func GetDiffNowDate(date string) (int, error) {
	tm, err := time.ParseInLocation(DATELayout, date, cst)
	if err != nil {
		return -1, err
	}

	year, month, day := GetDateNow()
	tmNow := time.Date(year, month, day, 0, 0, 0, 0, cst)

	return int(tm.Sub(tmNow).Hours() / 24), nil
}

// IsAfterPoint 是否在时间点之后
func IsAfterPoint(str string) bool {
	var hour, min int
	count, err := fmt.Sscanf(str, "%02d:%02d", &hour, &min)
	if count != 2 || err != nil {
		return false
	}
	timeNow := time.Now().In(cst)

	year, month, day := GetDateNow()
	tmPoint := time.Date(year, month, day, hour, min, 0, 0, cst)
	return !timeNow.Before(tmPoint)
}
