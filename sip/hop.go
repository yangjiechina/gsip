package sip

type Hop struct {
	IP        string
	Port      int
	Transport string
}

func (h *Hop) isTCP() bool {
	return h.Transport == TCP
}

func findNextHop(request *Request) (*Hop, error) {
	header := request.GetHeader(RouteName)
	var hostPort HostPort
	if header != nil {
		hostPort = header[0].(*Route).Address[0].HostPort
	} else {
		hostPort = request.GetRequestLine().RequestUri.HostPort
	}

	if hostPort.Port == 0 {
		hostPort.Port = 5060
		//TLS 5061
	}

	return &Hop{hostPort.Host, hostPort.Port, request.Via().transport}, nil
}
