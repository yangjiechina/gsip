package main

import (
	"fmt"
	"gsip/sip"
	"strings"
	"time"
)

// statusType ON/OFF
// resultType OK/ERROR

type AlarmStatus struct {
	Num        int    `xml:"num"`
	DeviceID   string `xml:"DeviceID" json:"deviceID"`
	DutyStatus string `xml:"DutyStatus" json:"dutyStatus"` //ONDUTY/OFFDUTY/ALARM
}

// Info A2.6 f
type Info struct {
	DeviceName   string `xml:"DeviceName" json:"deviceName"`
	Manufacturer string `xml:"Manufacturer" json:"manufacturer"`
	Model        string `xml:"Model" json:"model"`
	Firmware     string `xml:"Firmware" json:"firmware"`
	Channel      int    `xml:"Channel" json:"channel"` //<! -- 视频输入通道数(可选)--
}

// Status A2.6 g
type Status struct {
	Online      string        `xml:"Online" json:"online"`         //<! -- 是否在线(必选)-->  ONLINE|OFFLINE
	Status      string        `xml:"Status" json:"status"`         //<! -- 是否正常工作(必选)--> resultType
	Reason      string        `xml:"Reason" json:"reason"`         //<! -- 不正常工作原因(可选)-->
	Encode      string        `xml:"Encode" json:"encode"`         //<! -- 是否编码(可选)-->  statusType
	Record      string        `xml:"Record" json:"record"`         //<! -- 是否录像(可选)-->  statusType
	DeviceTime  string        `xml:"DeviceTime" json:"deviceTime"` //<! -- 设备时间和日期(可选)-->
	AlarmStatus []AlarmStatus `xml:"AlarmStatus"`
}

type Device struct {
	DeviceID string `xml:"DeviceID" json:"deviceID"`
	Info
	Status

	Transport string              `json:"transport"`
	IP        string              `xml:"IP" json:"ip"`
	Port      int                 `xml:"Port" json:"port"`
	Channels  map[string]*Channel `json:"channels"`

	subscribeMobilePositionAutoRefresher sip.AutoRefresher
	subscribeMobilePositionDialog        *sip.Dialog
}

type Channel struct {
	DeviceID            string `xml:"DeviceID" json:"deviceID"`
	Name                string `xml:"Name" json:"name"`
	Manufacturer        string `xml:"Manufacturer" json:"manufacturer"`
	Model               string `xml:"Model" json:"model"`
	Owner               string `xml:"Owner" json:"owner"`
	CivilCode           string `xml:"CivilCode" json:"civilCode"`
	Block               string `xml:"Block" json:"block"`
	Address             string `xml:"Address" json:"address"`
	Parental            int    `xml:"Parental" json:"parental"`
	ParentID            string `xml:"ParentID" json:"parentID"`
	SafetyWay           int    `xml:"SafetyWay" json:"safetyWay"`
	RegisterWay         int    `xml:"RegisterWay" json:"registerWay"`
	CertNum             string `xml:"CertNum" json:"certNum"`
	Certifiable         int    `xml:"Certifiable" json:"certifiable"`
	ErrCode             int    `xml:"ErrCode" json:"errCode"`
	EndTime             string `xml:"EndTime" json:"endTime"`
	Secrecy             int    `xml:"Secrecy" json:"secrecy"`
	IPAddress           string `xml:"IPAddress" json:"ipAddress"`
	Port                int    `xml:"Port" json:"port"`
	Password            string `xml:"Password" json:"password"`
	Status              string `xml:"Status" json:"status"` //statusType
	Longitude           string `xml:"Longitude" json:"longitude"`
	Latitude            string `xml:"Latitude" json:"latitude"`
	PTZType             int    `xml:"PTZType" json:"ptzType"`                         //<! --摄像机类型扩展,标识摄像机类型:1-球机;2-半球;3-固定枪机;4-遥控枪 机。当目录项为摄像机时可选。-->
	PositionType        int    `xml:"PositionType" json:"positionType"`               //<! --摄像机位置类型扩展。1-省际检查站、2-党政机关、3-车站码头、4-中心广 场、5-体育场馆、6-商业中心、7-宗教场所、8-校园周边、9-治安复杂区域、10-交通 干线。当目录项为摄像机时可选。-->
	RoomType            int    `xml:"RoomType" json:"roomType"`                       //<! --摄像机安装位置室外、室内属性。1-室外、2-室内。当目录项为摄像机时可 选,缺省为1。-->
	UseType             int    `xml:"UseType" json:"useType"`                         //<! --摄像机用途属性。1-治安、2-交通、3-重点。当目录项为摄像机时可选。-->
	SupplyLightType     int    `xml:"SupplyLightType" json:"supplyLightType"`         //<! --摄像机补光属性。1-无补光、2-红外补光、3-白光补光。当目录项为摄像机时可选,缺省为1。-->
	DirectionType       int    `xml:"DirectionType" json:"directionType"`             //<! --摄像机监视方位属性。1-东、2-西、3-南、4-北、5-东南、6-东北、7-西南、8-西北。当目录项为摄像机时且为固定摄像机或设置看守位摄像机时可选。-->
	Resolution          string `xml:"Resolution" json:"resolution"`                   //<! --摄像机支持的分辨率,可有多个分辨率值,各个取值间以“/”分隔。分辨率取值参见附录 F中SDPf字段规定。当目录项为摄像机时可选。-->
	BusinessGroupID     string `xml:"BusinessGroupID" json:"businessGroupID"`         //<! --虚拟组织所属的业务分组ID,业务分组根据特定的业务需求制定,一个业务分组包含一组特定的虚拟组织。-->
	DownloadSpeed       string `xml:"DownloadSpeed" json:"downloadSpeed"`             //<! -- 下载倍速范围(可选),各可选参数以“/”分隔,如设备支持1,2,4倍速下载则应写为“1/2/4”-->
	SVCSpaceSupportMode int    `xml:"SVCSpaceSupportMode" json:"svcSpaceSupportMode"` //<! -- 空域编码能力,取值0:不支持;1:1级增强(1个增强层);2:2级增强(2个增强层);3:3级增强(3个增强层)(可选)-->
	SVCTimeSupportMode  int    `xml:"SVCTimeSupportMode" json:"svcTimeSupportMode"`   //<! -- 时域编码能力,取值0:不支持;1:1级增强;2:2级增强;3:3级增强(可选)-->
}

func (d *Device) DoDeviceInfo() {
	deviceInfoFormat := "<?xml version=\"1.0\"?>\r\n" +
		"<Query>\r\n" +
		"<CmdType>DeviceInfo</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"</Query>\r\n"

	query := fmt.Sprintf(deviceInfoFormat, "1", d.DeviceID)

	message := SipAgent.createRequestMessage(d.DeviceID, d.IP, d.Port, SipAgent.sipId, d.DeviceID, d.Transport, xmlContentType, []byte(query))
	transaction, err := SipAgent.newClientTransaction(d.Transport, message)
	if err != nil {
		fmt.Printf("server error msg:%s", err.Error())
		return
	}

	if _, err := transaction.Execute(); err != nil {
		return
	}
}

func (d *Device) DoDeviceStatus() {
	deviceInfoFormat := "<?xml version=\"1.0\"?>\r\n" +
		"<Query>\r\n" +
		"<CmdType>DeviceStatus</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"</Query>\r\n"

	query := fmt.Sprintf(deviceInfoFormat, "1", d.DeviceID)

	message := SipAgent.createRequestMessage(d.DeviceID, d.IP, d.Port, SipAgent.sipId, d.DeviceID, d.Transport, xmlContentType, []byte(query))
	transaction, err := SipAgent.newClientTransaction(d.Transport, message)
	if err != nil {
		fmt.Printf("server error msg:%s", err.Error())
		return
	}

	if _, err := transaction.Execute(); err != nil {
		return
	}
}

func (d *Device) DoCatalog() {
	catalogFormat := "<?xml version=\"1.0\"?>\r\n" +
		"<Query>\r\n" +
		"<CmdType>Catalog</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"</Query>\r\n"

	msgBody := fmt.Sprintf(catalogFormat, "1", d.DeviceID)

	catalogRequest := SipAgent.createRequestMessage(d.DeviceID, d.IP, d.Port, SipAgent.sipId, d.DeviceID, d.Transport, xmlContentType, []byte(msgBody))

	transaction, err := SipAgent.newClientTransaction(d.Transport, catalogRequest)
	if err != nil {
		return
	}

	if _, err = transaction.Execute(); err != nil {
		return
	}
}

func (d *Device) OnKeepAlive(request *sip.Request) {

}

func (d *Device) OnCatalog(request *sip.Request) {
	content := request.Content()
	if content == nil {
		return
	}

}

func (d *Device) OnLogout(request *sip.Request) {

}

func (d *Device) OnACK(event *sip.RequestEvent) {
	go func() {
		time.Sleep(10 * time.Second)
		request, _ := event.Dialog.CreateRequest(sip.BYE)
		event.Dialog.Delete()
		if transaction, err := SipAgent.newClientTransaction(d.Transport, request); err == nil {
			transaction.Execute()
		}
	}()
}

func (d *Device) OnNotify(event *sip.RequestEvent) {
	if header := event.Request.GetHeader(sip.SubscriptionStateName); header != nil {
		state := header[0].(*sip.SubscriptionState)
		if strings.ToLower(state.State) == "terminated" {
			event.Dialog.Delete()
			if d.subscribeMobilePositionAutoRefresher != nil {
				d.subscribeMobilePositionAutoRefresher.Stop()
			}
		}
	}
}

func (d *Device) OnBye(event *sip.RequestEvent) {
	event.Dialog.Delete()
}

func (d *Device) DoLive() {

	sdpFormat := "v=0\r\n" +
		"o=%s 0 0 IN IP4 %s\r\n" +
		"s=Play\r\n" +
		"c=IN IP4 %s\r\n" +
		"t=0 0\r\n" +
		"m=video 20000 RTP/AVP 96\r\n" +
		"a=recvonly\r\n" +
		"a=rtpmap:96 PS/90000\r\n" +
		"y=011232323\r\n"

	sdp := fmt.Sprintf(sdpFormat, SipAgent.sipId, SipAgent.listIP, SipAgent.listIP)

	channelId := d.DeviceID[0:10] + "131" + d.DeviceID[13:]

	inviteRequest := SipAgent.createEmptyRequestMessage(sip.INVITE, channelId, d.IP, d.Port, SipAgent.sipId, channelId, d.Transport)
	contentType := sip.ContentType("Application/SDP")
	inviteRequest.SetContent(&contentType, []byte(sdp))
	uri := sip.NewSipUri(SipAgent.sipId, SipAgent.listIP, SipAgent.listPort)
	contact := sip.Contact{Address: sip.NewAddress(uri)}
	inviteRequest.SetHeader(&contact)

	if transaction, err := SipAgent.newClientTransaction(d.Transport, inviteRequest); err == nil {
		transaction.SendRequest(func(event *sip.ResponseEvent) {
			code := event.Response.GetStatusCode()
			if code < 200 {
				println("收到临时应答")
			} else if code == 200 {
				println("收到终结应答")
				//发送ACK
				ack := event.Dialog.CreateAck(event.Response.CSeq().Number)
				ack.SetHeader(&contact)
				if err2 := event.Dialog.SendAck(ack); err2 != nil {
					println("发送ACK失败:%s", err2.Error())
				} else {
					go func() {
						time.Sleep(10 * time.Second)
						request, _ := event.Dialog.CreateRequest(sip.BYE)
						event.Dialog.Delete()
						if transaction, err := SipAgent.newClientTransaction(d.Transport, request); err == nil {
							transaction.Execute()
						}
					}()
				}
			} else {
				println("请求视频失败")
			}
		}, func(err *sip.UACError) {
			println("请求视频失败,没有收到任何应答")
		})
	}
}

func (d *Device) OnSubscribeMobilePositionHandle(status bool, terminated bool, err error) {
	//restart subscribe
	if terminated {
		d.subscribeMobilePositionDialog.Delete()
		println("刷新订阅失败 订阅已经终结")
	}

	if status {
		println("刷新订阅成功")
	} else {
		println("刷新订阅失败 当前订阅任然有效")
	}
}

func (d *Device) DoSubscribeMobilePosition() {
	contentFormat := "<?xml version=\"1.0\"?>" +
		"<Query>" +
		"<CmdType>MobilePosition</CmdType>" +
		"<SN>%s</SN>" +
		"<DeviceID>%s</DeviceID>" +
		"<Interval>%d</Interval>" +
		"</Query>"

	content := fmt.Sprintf(contentFormat, "1", d.DeviceID, 10)
	subscribeRequest := SipAgent.createEmptyRequestMessage(sip.SUBSCRIBE, d.DeviceID, d.IP, d.Port, SipAgent.sipId, d.DeviceID, d.Transport)
	contentType := sip.ContentType(xmlContentType)
	subscribeRequest.SetContent(&contentType, []byte(content))
	event := sip.Event{Type: "presence"}
	subscribeRequest.SetHeader(&event)
	uri := sip.NewSipUri(SipAgent.sipId, SipAgent.listIP, SipAgent.listPort)
	contact := sip.Contact{Address: sip.NewAddress(uri)}
	subscribeRequest.SetHeader(&contact)
	expires := sip.Expires(3600)
	subscribeRequest.SetHeader(&expires)

	transaction, err := SipAgent.newClientTransaction(d.Transport, subscribeRequest)
	if err != nil {
		return
	}

	if responseEvent, err := transaction.Execute(); err != nil {
		return
	} else {
		response := responseEvent.Response
		if response.GetStatusCode() == 200 {
			d.subscribeMobilePositionDialog = responseEvent.Dialog
			d.subscribeMobilePositionAutoRefresher = SipAgent.stack.StartAutoRefreshWithSubscribe(subscribeRequest, responseEvent.Dialog, 10*time.Second, d.OnSubscribeMobilePositionHandle)
			println("订阅位置成功")
		} else {
			println("订阅位置失败")
		}
	}
}
