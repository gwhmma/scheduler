package alert

import (
	"github.com/BurntSushi/toml"
	"github.com/go-gomail/gomail"
)

type Alert struct {
	Host      string `json:"host"`       // 邮箱host
	Port      int    `json:"port"`       // 邮箱端口
	FromEmail string `json:"from_email"` // 发送邮件邮箱
	Password  string `json:"password"`   // 密码 or 授权码
	ToEmail   string `json:"to_email"`   // 接受邮件邮箱
}

type AlertsInfo struct {
	Worker    string // 产生告警信息的worker节点
	AlertType int64  // 告警的类型 1 超时  2 执行出错
	ErrorInfo string // 错误信息
	Time      string // 告警发生时间
}

// 如果该节点不再是leader 那么就取消他的告警功能
var LoseLeader chan struct{}
var AlertsEvent chan *AlertsInfo
var GM *gomail.Message

func loadEmailCfg(path string) (*Alert, error) {
	cfg := &Alert{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func InitAlert(path string) error {
	cfg, err := loadEmailCfg(path)
	if err != nil {
		return err
	}

	GM := gomail.NewMessage()
	GM.SetHeader("To", cfg.ToEmail)
	GM.SetAddressHeader("From", cfg.FromEmail, "")

	go cfg.alertLoop()

	return nil
}

func (a *Alert) alertLoop() {
	for {
		select {
		case <-LoseLeader: // 如果该节点不再是leader 那么就取消他的告警功能
			goto END
		case alerts := <-AlertsEvent:
			a.sendMail(alerts)
		}
	}
END:
}

// 发送告警邮件
func (a *Alert) sendMail(alert *AlertsInfo) error {
	var tpe string
	switch alert.AlertType {
	case 1:
		tpe = "超时"
	case 2:
		tpe = "执行出错"
	}

	body := "<h2> 告警类型 : " + tpe + "</h2>\n"
	body += "<p>告警节点 : " + alert.Worker + "</p>\n"
	body += "<p>告警信息 : " + alert.ErrorInfo + "</p>\n"

	gm := GM
	gm.SetBody("text/html", body)
	dialer := gomail.NewDialer(a.Host, a.Port, a.FromEmail, a.Password)

	err := dialer.DialAndSend(gm)
	if err != nil {
		return err
	}

	return nil
}

func SendAlertInfo(alerts *AlertsInfo) {
	AlertsEvent <- alerts
}

func StopAlerts() {
	LoseLeader <- struct{}{}
}
