package sip

import "testing"

func TestParseStrMessage(t *testing.T) {
	msgs := []string{
		"REGISTER sip:34020000002000000002@192.168.1.100:64824 SIP/2.0\r\nCall-ID: 6988613433\r\nCSeq: 1 REGISTER\r\nFrom: <sip:34020111002000011111@3402011100>;tag=9cd9e5da1f014a8b835693e8672048ae\r\nTo: <sip:34020111002000011111@3402011100>\r\nVia: SIP/2.0/UDP 192.168.1.109:15060;rport;branch=z9hG4bK-3139-346a16d95b4f63fee5539aef302cf965\r\nMax-Forwards: 70\r\nUser-Agent: IP Camera\r\nExpires: 3600\r\nContent-Length: 0\r\n\r\n",
		"REGISTER sip:34020000002000000002@192.168.1.100:64824 SIP/2.0\r\nCall-ID: 6988613433\r\nCSeq: 1 REGISTER\r\nFrom: <sip:34020111002000011111@3402011100>;tag=9cd9e5da1f014a8b835693e8672048ae\r\nTo: <sip:34020111002000011111@3402011100>\r\nVia: SIP/2.0/UDP 192.168.1.109:15060;rport;branch=z9hG4bK-3139-346a16d95b4f63fee5539aef302cf965\r\nMax-Forwards: 70\r\nUser-Agent: IP Camera\r\nExpires: 3600\r\nContent-Length: 0\r\nAuthorization: Digest username=\"34020111002000011111\",realm=\"3402000000\",nonce=\"9bd055\",uri=\"sip:34020000002000000002@192.168.1.100:64824\",responseEvent=\"da8b749e7f6f97e33af9d33e9f2571f1\",algorithm=MD5\r\n\r\n",
		"REGISTER sip:34020000002000000002@192.168.1.100:64824 SIP/2.0\r\nCall-ID: 6988613433\r\nCSeq: 1 REGISTER\r\nFrom: <sip:34020111002000011111@3402011100>;tag=9cd9e5da1f014a8b835693e8672048ae\r\nTo: <sip:34020111002000011111@3402011100>\r\nVia: SIP/2.0/UDP 192.168.1.109:15060;rport;branch=z9hG4bK-3139-346a16d95b4f63fee5539aef302cf965\r\nMax-Forwards: 70\r\nUser-Agent: IP Camera\r\nExpires: 3600\r\nContent-Length: 0\r\nWWW-Authenticate: Digest realm=\"testrealm@IP.com\",qop=\"auth,auth-int\",nonce=\"dcd98b7102dd2f0e8b11d0f600bfb0c093\",opaque=\"5ccc069c403ebaf9f0171e9517f40e41\"\r\n\r\n",
		"SUBSCRIBE sip:340200000013200000009@116.30.230.1:21896 SIP/2.0\r\nVia: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK.AYFPC6gDzGLizaAktlGlJGC7yTqieJQ2\r\nCSeq: 1 SUBSCRIBE\r\nFrom: <sip:34020000002000000001@3402000000>;tag=a033d02237e7\r\nTo: <sip:340200000013200000009@116.30.230.1:21896>\r\nCall-ID: s7BCqyWHgfTVQQ4xbyIgbZVmmX0Qj246\r\nContact: <sip:34020000002000000001@49.235.63.67:5060>\r\nMax-Forwards: 70\r\nExpires: 315360000\r\nContent-Type: Application/MANSCDP+xml\r\nUser-Agent: GoSIP\r\nEvent: Catalog;id=2\r\nContent-Length: 158\r\n\r\n<?xml version=\"1.0\"?>\r\n<Query>\r\n<CmdType>MobilePosition</CmdType>\r\n<SN>1</SN>\r\n<DeviceID>340200000013200000009</DeviceID>\r\n<Interval>10</Interval>\r\n</Query>\r\n",
		"SUBSCRIBE sip:34020000001110000001@3402000000 SIP/2.0\r\nFrom: <sip:34020000002000000001@192.168.1.116:5060>;tag=9876512341241234\r\nTo: <sip:34020000001110000001@3402000000>\r\nContent-Length: 159\r\nCSeq: 5 SUBSCRIBE\r\nCall-ID: 9876512345678911\r\nVia: SIP/2.0/UDP 192.168.1.116:5060;wlsscid=377aa9afcf1b36f;branch=123133532300004\r\nContent-Type: Application/MANSCDP+xml\r\nMax-Forwards: 70\r\nExpires:300\r\nContact: <sip:34020000002000000001@192.168.1.116:5060>\r\nEvent: id=1984\r\n\r\n<?xml version=\"1.0\"?> \r\n<Query>\r\n<CmdType>MobilePosition</CmdType>\r\n<SN>17430</SN>\r\n<DeviceID>34020000001110000001</DeviceID>\r\n<Interval>5</Interval>\r\n</Query>",
		"SUBSCRIBE sip:34020000001110000001@3402000000 SIP/2.0\r\nFrom: <sip:34020000002000000001@192.168.1.116:5060>;tag=9876512341241234\r\nTo: <sip:34020000001110000001@3402000000>\r\nContent-Length: 135\r\nCSeq: 5 SUBSCRIBE\r\nCall-ID: 9876512345678911\r\nVia: SIP/2.0/UDP 192.168.1.116:5060;wlsscid=377aa9afcf1b36f;branch=123133532300006\r\nContent-Type: Application/MANSCDP+xml\r\nMax-Forwards: 70\r\nExpires:0\r\nContact: <sip:34020000002000000001@192.168.1.116:5060>\r\nEvent: id=1984\r\n\r\n<?xml version=\"1.0\"?> \r\n<Query>\r\n<CmdType>MobilePosition</CmdType>\r\n<SN>17430</SN>\r\n<DeviceID>34020000001110000001</DeviceID>\r\n</Query>",
		"REGISTER sip:192.168.1.100 SIP/2.0\r\nVia: SIP/2.0/UDP 192.168.1.120:50793;rport;branch=z9hG4bKPjda7bf2df0fa2427f89214a3aeac1aa22\r\nMax-Forwards: 70\r\nFrom: <sip:34020000001320000001@340200000>;tag=af2cf14cc1f34240aaafdacbeac326c0\r\nTo: <sip:34020000001320000001@340200000>\r\nCall-ID: ba00b14e70c24972b1c648fb4e138c39\r\nCSeq: 21436 REGISTER\r\nUser-Agent: MicroSIP/3.21.2\r\nContact: <sip:34020000001320000001@192.168.1.120:50793;ob>\r\nExpires: 300\r\nAllow: PRACK, INVITE, ACK, BYE, CANCEL, UPDATE, INFO, SUBSCRIBE, NOTIFY, REFER, MESSAGE, OPTIONS\r\nContent-Length:  0\r\n",
		"REGISTER sip:49.235.63.67 SIP/2.0\r\nVia: SIP/2.0/UDP 192.168.1.108:57433;rport;branch=z9hG4bKPj1f18630ac4ab4c3e8b4b3e5652db6c2c\r\nMax-Forwards: 70\r\nFrom: \"phone\" <sip:34020000001110000001@340200000>;tag=f2f0114d1d954af7a660feb1a687184f\r\nTo: \"phone\" <sip:34020000001110000001@340200000>\r\nCall-ID: 6b9bfc9c3da14c41be88ab9a36d852d7\r\nCSeq: 38137 REGISTER\r\nUser-Agent: MicroSIP/3.21.2\r\nContact: \"phone\" <sip:34020000001110000001@192.168.1.108:57433;ob>\r\nExpires: 300\r\nAllow: PRACK, INVITE, ACK, BYE, CANCEL, UPDATE, INFO, SUBSCRIBE, NOTIFY, REFER, MESSAGE, OPTIONS\r\nContent-Length:  0\n\n",
	}
	for _, msg := range msgs[1:2] {
		bytes := []byte(msg)
		message, _, err := parseMessage(bytes, len(bytes))
		println(message)
		println(err)
	}
}
