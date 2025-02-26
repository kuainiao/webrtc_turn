package client

import (
	"errors"
	"github.com/ricochet2200/gun/msg"
	"log"
	"net"
	"time"
)

type Client struct {
	server   string
	user     *msg.UserAttr
	realm    *msg.RealmAttr
	nonce    *msg.NonceAttr
	password string
}

func NewClient(server, user, passwd string) (*Client, error) {

	userAttr, err := msg.NewUser(user)
	if err != nil {
		return nil, err
	}

	//TODO: SASLPrep the password
	return &Client{server, userAttr, nil, nil, passwd}, nil
}

// Sends a request where you expect to get a response back
func (this *Client) SendReqRes(req *msg.Message) (*Connection, error) {

	conn, err := net.DialTimeout("tcp", this.server, 15*time.Second)
	if err != nil {
		log.Println("Failed to create connection: ", err)
		return nil, err
	}

	laddr := conn.LocalAddr()
	ip := laddr.(*net.TCPAddr).IP
	port := laddr.(*net.TCPAddr).Port

	xor := msg.NewXORAddress(ip, port, req.Header())
	req.AddAttribute(xor)

	if this.nonce != nil && this.realm != nil {
		req.AddAttribute(this.user)
		req.AddAttribute(this.realm)
		req.AddAttribute(this.nonce)

		username := this.user.String()
		realm := this.realm.String()

		integrity := msg.NewIntegrityAttr(username, this.password, realm, req)
		req.AddAttribute(integrity)
	}

	conn.Write(req.EncodeMessage())

	res, err := msg.DecodeMessage(conn)

	if err != nil {
		conn.Close()
		return nil, err
	}

	if eattr, err := res.Attribute(msg.ErrorCode); err == nil {
		if code, err := eattr.(*msg.StunError).Code(); err == nil {
			log.Println("error code", code)
			switch code {

			case msg.StaleNonce:
				log.Println("Stale Nonce, calling authenticate...")
				return this.Authenticate(res, req)

			case msg.Unauthorized:
				log.Println("unauthorized")

				if _, err := req.Attribute(msg.MessageIntegrity); err == nil {
					conn.Close()
					return nil, errors.New("Invalid credentials")
				} else {
					return this.Authenticate(res, req)
				}
			}
		}
	}

	return &Connection{res, conn}, nil
}

func (this *Client) Bind() (*Connection, error) {

	log.Println("Binding...")
	req := msg.NewRequest(msg.Request | msg.Binding)
	return this.SendReqRes(req)
}

func ToIPPort(conn *Connection) (net.IP, int, error) {

	xattr, err := conn.Res.Attribute(msg.XORMappedAddress)
	if err != nil {
		return nil, -1, err
	}

	xor := xattr.(*msg.XORAddress)

	return xor.IP(conn.Res.Header()), xor.Port(), nil
}

func (this *Client) Authenticate(res, oldReq *msg.Message) (*Connection, error) {

	req := msg.NewRequest(msg.Request | msg.Binding)
	req.CopyAttributes(oldReq)

	r, _ := res.Attribute(msg.Realm)
	this.realm = r.(*msg.RealmAttr)

	nonce, _ := res.Attribute(msg.Nonce)
	this.nonce = nonce.(*msg.NonceAttr)

	return this.SendReqRes(req)
}
