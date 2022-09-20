package main

import (
	"encoding/xml"
	"fmt"
	"gsip/examples"
	"gsip/sip"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var xmlContentType sip.ContentType = "Application/MANSCDP+xml"
var registerSuccessCount int
var heartbeatCount uint64
var activeAlarmCount uint64
var mobilePositionCount uint64

type GBClient struct {
	deviceId            string
	channelId           string
	password            string
	expires             int
	keepaliveInterval   int
	activeAlarmInterval int

	sipStack    *sip.Stack
	listenPoint *sip.ListeningPoint
	requestUri  *sip.SipUri

	from                   *sip.From
	registerTo             *sip.To
	parentTo               *sip.To
	refreshRegisterRequest *sip.Request
	contactHeader          *sip.Contact

	keepaliveTimer      *time.Timer
	mobilePositionTimer *time.Timer
	activeAlarmTimer    *time.Timer

	keepaliveFailedCount    int
	mobilePositionInterval  int
	mobilePositionExpires   int
	mobilePositionEvent     string
	mobilePositionId        string
	mobilePositionDialog    *sip.Dialog
	mobilePositionStartTime int64

	autoRefresher sip.AutoRefresher
}

func (c *GBClient) sendRequest(request *sip.Request) *sip.ResponseEvent {
	//fmt.Printf("send request:%s", request.ToString())
	if transaction, err := c.listenPoint.NewClientTransaction(request); err == nil {
		if response, err := transaction.Execute(); err != nil {
			fmt.Printf("发送消息失败:%s\r\n%s", err.Error(), request.ToString())
		} else {
			return response
		}
	}

	return nil
}

func (c *GBClient) OnRefreshRegisterHandle(status bool, err error) {
	if status {
		println("刷新注册成功")
	} else {
		fmt.Printf("刷新注册失败:%s", err.Error())
		c.offline()
	}
}

func (c *GBClient) doRegister(message *sip.Request) bool {
	var registerOK bool
	if event := c.sendRequest(message); event != nil {
		if event.Response.GetStatusCode() == sip.OK {
			c.refreshRegisterRequest = message
			registerOK = true
		} else if event.Response.GetStatusCode() == sip.Unauthorized && sip.GenerateCredentials(message, event.Response, c.password) {
			c.refreshRegisterRequest = message
			message.RemoveTransactionTag()
			if responseEvent := c.sendRequest(message); responseEvent != nil && responseEvent.Response.GetStatusCode() == sip.OK {
				c.autoRefresher = c.sipStack.StartAutoRefreshWithRegister(c.refreshRegisterRequest, c.OnRefreshRegisterHandle)
				via := responseEvent.Response.Via()
				if via.Received() != "" && via.RPort() != 0 {
					c.contactHeader.Address.Uri.HostPort.Host = via.Received()
					c.contactHeader.Address.Uri.HostPort.Port = via.RPort()
				}
				registerOK = true
			}
		}
	}

	return registerOK
}

func (c *GBClient) online() {
	registerSuccessCount++

	if c.keepaliveTimer == nil {
		c.keepaliveTimer = time.AfterFunc(time.Duration(c.keepaliveInterval)*time.Second, c.startKeepalive)
	} else {
		c.keepaliveTimer.Reset(time.Duration(c.keepaliveInterval) * time.Second)
	}

	//先执行一次
	go c.startKeepalive()

	if c.activeAlarmInterval > 0 {

		if c.activeAlarmTimer == nil {
			c.activeAlarmTimer = time.AfterFunc(time.Duration(c.activeAlarmInterval)*time.Second, c.startActiveAlarm)
		} else {
			c.activeAlarmTimer.Reset(time.Duration(c.activeAlarmInterval) * time.Second)
		}

		c.startActiveAlarm()
	}
}

func (c *GBClient) offline() {
	if c.autoRefresher != nil {
		c.autoRefresher.Stop()
	}
	registerSuccessCount--
	c.keepaliveTimer.Stop()
	if c.mobilePositionTimer != nil {
		c.mobilePositionTimer.Stop()
	}
	if c.activeAlarmTimer != nil {
		c.activeAlarmTimer.Stop()
	}

	c.startRegister()
}

func (c *GBClient) startRegister() {
	message := c.listenPoint.NewEmptyRequestMessage(sip.REGISTER, c.requestUri, c.from.Clone().(*sip.From), c.registerTo)
	expires := sip.Expires(c.expires)
	message.SetHeader(&expires)

	if !c.doRegister(message) {
		time.Sleep(10 * time.Second)
		c.startRegister()
	} else {
		c.online()
	}
}

func (c *GBClient) startActiveAlarm() {
	c.activeAlarmTimer.Reset(time.Duration(c.activeAlarmInterval) * time.Second)

	alarmFormat := "<?xml version=\"1.0\" ?>\r\n" +
		"<Notify>\r\n" +
		"<CmdType>Alarm</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<AlarmPriority>%d</AlarmPriority>\r\n" +
		"<AlarmMethod>%d</AlarmMethod>\r\n" +
		"<AlarmTime>%s</AlarmTime>\r\n" +
		"%s" +
		"</Notify>\r\n"
	alarmTime := time.Now().Format("2006-01-02T15:04:05")
	priority := rand.Intn(4)
	method := rand.Intn(7)
	var alarmType int
	if method == 2 {
		alarmType = rand.Intn(5)
	} else if method == 5 {
		alarmType = rand.Intn(12)
	} else if method == 6 {
		alarmType = rand.Intn(2)
	}

	var body string
	if alarmType == 0 {
		body = fmt.Sprintf(alarmFormat, "1", c.deviceId, priority, method, alarmTime, "")
	} else {
		body = fmt.Sprintf(alarmFormat, "1", c.deviceId, priority, method, alarmTime, fmt.Sprintf("<Info>\r\n<AlarmType>%d</AlarmType>\r\n</Info>\r\n", alarmType))
	}

	message := c.listenPoint.NewRequestMessage(sip.MESSAGE, c.requestUri, c.from, c.registerTo, &xmlContentType, []byte(body))
	if responseEvent := c.sendRequest(message); responseEvent != nil && responseEvent.Response.GetStatusCode() == sip.OK {
		activeAlarmCount++
	}
}

func (c *GBClient) startKeepalive() {
	c.keepaliveTimer.Reset(time.Duration(c.keepaliveInterval) * time.Second)
	if c.doKeepalive() {
		heartbeatCount++
		c.keepaliveFailedCount = 0
	} else {
		c.keepaliveFailedCount++
	}
	if c.keepaliveFailedCount >= 3 {
		c.offline()
	}
}

func (c *GBClient) doKeepalive() bool {
	keepaliveFormat := "<?xml version=\"1.0\" ?>\r\n" +
		"<Notify>\r\n" +
		"<CmdType>Keepalive</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<Status>OK</Status>\r\n" +
		"</Notify>\r\n"
	content := fmt.Sprintf(keepaliveFormat, "1", c.deviceId)

	message := c.listenPoint.NewRequestMessage(sip.MESSAGE, c.requestUri, c.from, c.registerTo, &xmlContentType, []byte(content))
	responseEvent := c.sendRequest(message)
	return responseEvent != nil && responseEvent.Response.GetStatusCode() == sip.OK
}

func (c *GBClient) init() {
	c.from = sip.NewFrom(c.deviceId, c.listenPoint.IP, c.listenPoint.Port)
	c.registerTo = sip.NewTo(c.deviceId, c.listenPoint.IP, c.listenPoint.Port)
	c.parentTo = sip.NewTo(c.requestUri.User, c.requestUri.HostPort.Host, c.requestUri.HostPort.Port)
	c.contactHeader = &sip.Contact{Address: sip.NewAddress(sip.NewSipUri(c.deviceId, c.listenPoint.IP, c.listenPoint.Port))}
}

func (c *GBClient) start() {
	c.startRegister()
}

func (c *GBClient) OnDeviceInfo(body *MessageBody) {
	deviceInfoFormat := "<?xml version=\"1.0\"?>\r\n" +
		"<Response>\r\n" +
		"<CmdType>Info</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<DeviceName>gsip_examples</DeviceName>\r\n" +
		"<Result>OK</Result>\r\n" +
		"<DeviceType>IPC</DeviceType>\r\n" +
		"<Manufacturer>gsip</Manufacturer>\r\n" +
		"<Model>gsip</Model>\r\n" +
		"<Firmware>gsipV0.0.1</Firmware>\r\n" +
		"<Channel>1</Channel>\r\n" +
		"</Response>\r\n"

	deviceInfo := fmt.Sprintf(deviceInfoFormat, c.deviceId, body.SN)
	message := c.listenPoint.NewRequestMessage(sip.MESSAGE, c.requestUri, c.from, c.parentTo, &xmlContentType, []byte(deviceInfo))
	c.sendRequest(message)
}

func (c *GBClient) OnDeviceStatus(body *MessageBody) {
	deviceStatusFormat := "<?xml version=\"1.0\" ?>\r\n" +
		"<Response>\r\n" +
		"<CmdType>Status</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<Result>OK</Result>\r\n" +
		"<Online>ONLINE</Online>\r\n" +
		"<Status>OK</Status>\r\n" +
		"<Encode>ON</Encode>\r\n" +
		"<Record>ON</Record>\r\n" +
		"<Alarmstatus Num=\"0\" />\r\n" +
		"</Response>\r\n"

	deviceStatus := fmt.Sprintf(deviceStatusFormat, c.deviceId, body.SN)
	message := c.listenPoint.NewRequestMessage(sip.MESSAGE, c.requestUri, c.from, c.parentTo, &xmlContentType, []byte(deviceStatus))
	c.sendRequest(message)
}

func (c *GBClient) OnCatalog(body *MessageBody) {
	catalogFormat := "<?xml version=\"1.0\" ?>\r\n" +
		"<Response>\r\n" +
		"<CmdType>Catalog</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<SumNum>1</SumNum>\r\n" +
		"<DeviceList Num=\"1\">\r\n" +
		"<Item>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<Name>video_channel_1</Name>\r\n" +
		"<Manufacturer>gsip</Manufacturer>\r\n" +
		"<Model>gsip</Model>\r\n" +
		"<Owner>Owner</Owner>\r\n" +
		"<Address>Address</Address>\r\n" +
		"<Parental>0</Parental>\r\n" +
		"<ParentID>%s</ParentID>\r\n" +
		"<SafetyWay>0</SafetyWay>\r\n" +
		"<RegisterWay>1</RegisterWay>\r\n" +
		"<Secrecy>0</Secrecy>\r\n" +
		"<Status>ON</Status>\r\n" +
		"</Item>\r\n" +
		"</DeviceList>\r\n" +
		"</Response>\r\n"

	catalog := fmt.Sprintf(catalogFormat, body.SN, c.deviceId, c.channelId, c.deviceId)
	message := c.listenPoint.NewRequestMessage(sip.MESSAGE, c.requestUri, c.from, c.parentTo, &xmlContentType, []byte(catalog))
	c.sendRequest(message)
}

func (c *GBClient) OnMessage(body *MessageBody) {
	switch body.CmdType {
	case "Info":
		c.OnDeviceInfo(body)
		break
	case "Status":
		c.OnDeviceStatus(body)
		break
	case "Catalog":
		c.OnCatalog(body)
		break
	}
}

func (c *GBClient) startMobilePosition() {
	c.mobilePositionTimer.Reset(time.Duration(c.mobilePositionInterval) * time.Second)

	current := time.Now()
	expires := current.Unix() - c.mobilePositionStartTime
	request, _ := c.mobilePositionDialog.CreateRequest(sip.NOTIFY)
	if request == nil {
		c.mobilePositionTimer.Stop()
		return
	}
	event := &sip.Event{Type: c.mobilePositionEvent, ID: c.mobilePositionId}
	state := &sip.SubscriptionState{}
	state.State = "active"
	state.Expires = strconv.FormatInt(int64(c.mobilePositionExpires)-expires, 10)
	request.SetHeader(event)
	request.SetHeader(state)

	mobilePositionNotifyFormat := "<?xml version=\"1.0\" ?>\r\n" +
		"<Notify>\r\n" +
		"<DeviceID>%s</DeviceID>\r\n" +
		"<CmdType>MobilePosition</CmdType>\r\n" +
		"<SN>%s</SN>\r\n" +
		"<Time>%s</Time>\r\n" +
		"<Longitude>%s</Longitude>\r\n" +
		"<Latitude>%s</Latitude>\r\n" +
		"<Speed>0.0</Speed>\r\n" +
		"<Direction>0.0</Direction>\r\n" +
		"<Altitude>0.0</Altitude>\r\n" +
		"</Notify>\r\n"

	content := fmt.Sprintf(mobilePositionNotifyFormat, c.deviceId, "1", current.Format("2006-01-02T15:04:05"), "0.0", "0.0")
	request.SetContent(&xmlContentType, []byte(content))
	responseEvent := c.sendRequest(request)
	if responseEvent == nil || responseEvent.Response == nil || responseEvent.Response.GetStatusCode() != sip.OK {
		//通知失败
		fmt.Printf("发送位置失败:\r\n %s", request.ToString())
		c.stopMobilePosition()
	} else {
		mobilePositionCount++
	}
}

func (c *GBClient) stopMobilePosition() {
	c.mobilePositionTimer.Stop()
	c.mobilePositionDialog.Delete()
}

func (c *GBClient) OnSubMobilePosition(body *MessageBody, dialog *sip.Dialog, event *sip.Event, expires int, refresh bool) {
	c.mobilePositionInterval = body.Interval
	c.mobilePositionExpires = expires
	c.mobilePositionEvent = event.Type
	c.mobilePositionId = event.ID
	c.mobilePositionDialog = dialog
	c.mobilePositionStartTime = time.Now().Unix()

	if c.mobilePositionTimer == nil {
		c.mobilePositionTimer = time.AfterFunc(time.Duration(body.Interval)*time.Second, c.startMobilePosition)
	} else {
		c.mobilePositionTimer.Stop()
		c.mobilePositionTimer.Reset(time.Duration(body.Interval) * time.Second)
	}

	if !refresh {
		//订阅后先发一次
		c.startMobilePosition()
	}
}

func (c GBClient) OnSubAlarm(body *MessageBody, dialog *sip.Dialog, event *sip.Event, expires int) {

}

func (c GBClient) OnSubCatalog(body *MessageBody, dialog *sip.Dialog, event *sip.Event, expires int) {

}

func (c *GBClient) OnSubscribe(body *MessageBody, dialog *sip.Dialog, event *sip.Event, expires int) {
	switch body.CmdType {
	case "MobilePosition":
		//refresh subscribe
		if c.mobilePositionDialog != nil && c.mobilePositionDialog == dialog && c.mobilePositionDialog.GetDialogId() == dialog.GetDialogId() && expires > 0 {
			c.OnSubMobilePosition(body, dialog, event, expires, true)
		} else if expires <= 0 {
			c.stopMobilePosition()
		} else {
			if c.mobilePositionDialog != nil {
				c.stopMobilePosition()
			}
			c.OnSubMobilePosition(body, dialog, event, expires, false)
		}

		break
	case "Catalog":
		c.OnSubCatalog(body, dialog, event, expires)
		break
	case "Alarm":
		c.OnSubAlarm(body, dialog, event, expires)
		break
	}
}

type GBUA struct {
	devices map[string]*GBClient
	mutex   sync.RWMutex
}

func (G *GBUA) OnRequest(event *sip.RequestEvent) {
	body := event.Request.Content()
	contentType := event.Request.ContentType()
	var xmlBody *MessageBody
	if body != nil && contentType != nil && strings.ToLower(contentType.Value()) == strings.ToLower(string(xmlContentType)) {
		xmlBody = &MessageBody{}
		all := strings.ReplaceAll(string(body), "encoding=\"GB2312\"", "")
		if err := xml.Unmarshal([]byte(all), xmlBody); err != nil {
			println(err.Error())
		}
		//if err := decodeXML(body, xmlBody); err != nil {
		//	println(err.Error())
		//}
	}

	if xmlBody == nil {
		response := event.Request.CreateResponse(sip.BadRequest)
		event.ServerTransaction.SendResponse(response)
		return
	}
	G.mutex.RLock()
	device := G.devices[xmlBody.DeviceID]
	G.mutex.RUnlock()

	if device == nil {
		response := event.Request.CreateResponse(sip.NotFound)
		event.ServerTransaction.SendResponse(response)
		return
	}

	switch event.Request.GetRequestMethod() {
	case sip.MESSAGE:
		response := event.Request.CreateResponse(sip.OK)
		event.ServerTransaction.SendResponse(response)
		device.OnMessage(xmlBody)
		break
	case sip.SUBSCRIBE:
		evt := event.Request.Event()
		expires := event.Request.Expires()

		response := event.Request.CreateResponse(sip.OK)
		response.SetToTag(sip.GenerateTag())
		response.SetHeader(device.contactHeader)
		response.SetExpires(int(*expires))
		event.ServerTransaction.SendResponse(response)

		device.OnSubscribe(xmlBody, event.ServerTransaction.GetDialog(), evt, int(*expires))
		break
	}
}

type MessageBody struct {
	CmdType   string `xml:"CmdType"`
	DeviceID  string `xml:"DeviceID"`
	SN        string `xml:"SN"`
	StartTime string `xml:"StartTime"`
	EndTime   string `xml:"EndTime"`
	Interval  int    `xml:"Interval"`
}

func StartClient(config examples.ClientConfig) {
	gbua := &GBUA{}

	gbua.devices = make(map[string]*GBClient, 1000)

	requestUri := &sip.SipUri{
		User:     config.ParentId,
		HostPort: sip.HostPort{Host: config.ParentIP, Port: config.ParentPort},
		//HostPort: sip.HostPort{Host: "49.235.63.67", Port: 5060},
	}

	if strings.ToUpper(config.Transport) == sip.UDP && !config.PortSharedMode && config.Count > (65535-config.ListenPort) {
		config.Count = 65535 - config.ListenPort
	}

	if config.Count > 10000000 {
		config.Count = 9999999
	}

	deviceIdPrefix := "3402000000111"
	channelIdPrefix := "3402000000131"
	var stack *sip.Stack
	var listeningPoint *sip.ListeningPoint
	var stacks []*sip.Stack
	go func() {
		for i := 1; i <= config.Count; i++ {
			suffix := strconv.Itoa(i)
			for j := len(suffix); j <= 7; j++ {
				suffix = "0" + suffix
			}

			client := &GBClient{
				deviceId:            deviceIdPrefix + suffix,
				channelId:           channelIdPrefix + suffix,
				password:            config.Password,
				expires:             config.RegisterExpires,
				keepaliveInterval:   config.Heartbeat,
				activeAlarmInterval: config.ActiveAlarmInterval,
				sipStack:            stack,
				listenPoint:         listeningPoint,
				requestUri:          requestUri,
			}

			if config.PortSharedMode {
				if listeningPoint == nil {
					listeningPoint = &sip.ListeningPoint{IP: config.ListenIP, Port: config.ListenPort, Transport: strings.ToUpper(config.Transport)}
					stack = &sip.Stack{
						Listens:       []*sip.ListeningPoint{listeningPoint},
						EventListener: gbua,
					}

					if err := stack.Start(); err != nil {
						fmt.Errorf("sip stack 启动失败 %s", err.Error())
						return
					}
					stacks = append(stacks, stack)
				}
				client.sipStack = stack
				client.listenPoint = listeningPoint
			} else {
				listeningPoint = &sip.ListeningPoint{IP: config.ListenIP, Port: config.ListenPort + i - 1, Transport: strings.ToUpper(config.Transport)}
				stack = &sip.Stack{
					Listens:       []*sip.ListeningPoint{listeningPoint},
					EventListener: gbua,
				}

				if err := stack.Start(); err != nil {
					fmt.Errorf("sip stack 启动失败 %s", err.Error())
					continue
				}

				stacks = append(stacks, stack)
				client.sipStack = stack
				client.listenPoint = listeningPoint
			}

			client.init()
			gbua.mutex.Lock()
			gbua.devices[client.deviceId] = client
			gbua.mutex.Unlock()

			if config.Sleep > 0 {
				time.Sleep(time.Duration(config.Sleep) * time.Millisecond)
			}
			go client.start()

		}
	}()
	for {
		var clientTransactions int
		var serverTransactions int
		var dialogs int
		for _, s := range stacks {
			i, i2, i3 := s.Debug()
			clientTransactions += i
			serverTransactions += i2
			dialogs += i3
		}
		fmt.Printf("%s total:%d online:%d offline:%d heartbeat:%d active alarm:%d mobile position:%d clientTransactions:%d serverTransactions:%d dialogs:%d \r\n", time.Now().Format("2006-01-02T15:04:05"), config.Count, registerSuccessCount, config.Count-registerSuccessCount, heartbeatCount, activeAlarmCount, mobilePositionCount,
			clientTransactions,
			serverTransactions,
			dialogs)
		//runtime.GC()
		time.Sleep(5 * time.Second)
	}
}
