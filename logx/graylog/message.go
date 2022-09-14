package graylog

type logData struct {
	Source string `json:"host"`  //日志源主机/进程
	Level  string `json:"level"` //日志级别 debug,info,error等
	//ShortMessage string  `json:"short_message"` //a short descriptive message;
	//Timestamp    float64 `json:"timestamp"`
	//Version      string  `json:"version,omitempty"`
	Tag      string `json:"tag"`                //日志的tag，如订单中的订单号，登录/保持的用户名
	Message  string `json:"message"`            //日志内容
	Facility string `json:"facility,omitempty"` //筛选条件，一般为程序模块名,如loginSvr,sessionSvr

	ElapsedTime    int64  `json:"_elapsedtime"`              //该日志业务耗时时间(毫秒)
	SecondFacility string `json:"_secondfacility,omitempty"` //二级筛选条件：如robot中的createOrder
	LogTime        string `json:"_logtime"`                  //本条日志记录的生成时间
}
