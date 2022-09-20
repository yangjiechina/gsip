package sip

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
)

const (
	DefaultAlgorithm = "MD5"
	DefaultSchema    = "Digest"
)

func generateNonce() string {
	k := make([]byte, 12)
	for bytes := 0; bytes < len(k); {
		n, err := rand.Read(k[bytes:])
		if err != nil {
			panic("rand.Read() failed")
		}
		bytes += n
	}
	return base64.StdEncoding.EncodeToString(k)
}

func GenerateChallenge(response *Response, realm string) {
	header := NewWWWAuthenticateHeader()
	header.SetParameter("realm", realm)
	header.SetParameter("nonce", generateNonce())
	header.SetParameter("algorithm", DefaultAlgorithm)
	response.SetHeader(header)
}

func h(data string) string {
	hash := md5.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

func calculateResponse(username, realm, nonce, uri, password string) string {
	//H(data) = MD5(data)
	//KD(secret, data) = H(concat(secret, ":", data))
	//request-digest  = <"> < KD ( H(A1), unq(nonce-value) ":" H(A2) ) > <">
	A1 := fmt.Sprintf("%s:%s:%s", username, realm, password)
	A2 := fmt.Sprintf("%s:%s", REGISTER, uri)

	return h(h(A1) + ":" + nonce + ":" + h(A2))
}

func GenerateCredentials(request *Request, response *Response, password string) bool {
	if header := response.GetHeader(WWWAuthenticateName); header != nil {
		wwwAuthenticateHeader := header[0].(*WWWAuthenticate)
		if "" == wwwAuthenticateHeader.Realm() || "" == wwwAuthenticateHeader.Nonce() {
			return false
		}

		if "" != wwwAuthenticateHeader.Algorithm() && DefaultAlgorithm != wwwAuthenticateHeader.Algorithm() {
			return false
		}
		if "auth" == wwwAuthenticateHeader.Qop() || "auth-int" == wwwAuthenticateHeader.Qop() {
			return false
		}

		fromHeader := request.From()
		userName := fromHeader.Address.Uri.User
		realm := wwwAuthenticateHeader.Realm()
		nonce := wwwAuthenticateHeader.Nonce()
		requestUri := request.GetRequestLine().RequestUri.ToString()

		//Digest username="34020111002000011111",realm="3402000000",nonce="9bd055",uri="sip:34020000002000000002@192.168.1.100:64824",response="da8b749e7f6f97e33af9d33e9f2571f1",algorithm=MD5
		response := calculateResponse(userName, realm, nonce, requestUri, password)
		authorizationHeader := NewAuthorizationHeader()

		authorizationHeader.SetParameter("username", userName)
		authorizationHeader.SetParameter("realm", realm)
		authorizationHeader.SetParameter("nonce", nonce)
		authorizationHeader.SetParameter("uri", requestUri)
		authorizationHeader.SetParameter("response", response)
		authorizationHeader.SetParameter("algorithm", DefaultAlgorithm)

		request.SetHeader(authorizationHeader)

		return true
	}

	return false
}

func DoAuthenticatePlainTextPassword(request *Request, password string) bool {
	if header := request.GetHeader(AuthorizationName); header != nil {
		authorizationHeader := header[0].(*Authorization)
		if authorizationHeader.Username() == "" ||
			authorizationHeader.Realm() == "" ||
			authorizationHeader.Nonce() == "" ||
			authorizationHeader.Uri() == nil ||
			authorizationHeader.Response() == "" {
			return false
		}

		response := calculateResponse(authorizationHeader.Username(), authorizationHeader.Realm(), authorizationHeader.Nonce(), authorizationHeader.Uri().ToString(), password)
		return response == authorizationHeader.Response()
	}

	return false
}
