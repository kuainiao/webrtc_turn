package msg

import (
	"errors"
	"io"
	"log"
)

type Message struct {
	header *Header
	attr   []TLV
}

func NewRequest(msgType MessageType) *Message {
	return &Message{NewHeader(msgType, 0), []TLV{}}
}

// msgType should only include a class.  The method will be taken from
// the req.
func NewResponse(msgType MessageType, req *Message) *Message {
	t := req.Header().Type()&MethodMask | msgType&ClassMask
	header := &Header{t, 0, req.header.id}
	return &Message{header, []TLV{}}
}

func DecodeMessage(conn io.Reader) (*Message, error) {

	header, err := DecodeHeader(conn)
	if err != nil {
		return nil, err
	}

	tvl := []TLV{}
	for i := uint16(0); i < header.length; {
		if t, padding, err := Decode(conn); err != nil {
			log.Println(err)
			return nil, err
		} else {
			tvl = append(tvl, t)
			i += t.Length() + uint16(padding)
			if (t.Length()+uint16(padding))%4 != 0 {
				log.Println(t.TypeString(), "is not 4 byte aligned")
				return nil, errors.New(t.TypeString() + " not 4 byte aligned")
			}
		}
	}

	return &Message{header, tvl}, err
}

func (this *Message) EncodeMessage() []byte {
	ret := this.header.Data()
	for _, a := range this.attr {
		ret = append(ret, a.Encode()...)
	}
	return ret
}

func (this *Message) Type() MessageType {
	return this.header.msgType
}

func (this *Message) Header() *Header {
	return this.header
}

func (this *Message) AddAttribute(tlv TLV) {

	inserted := false
	for i, a := range this.attr {
		if tlv.Type() == a.Type() {
			this.header.length -= ((a.Length() + 3) / 4) * 4
			this.attr[i] = tlv
			inserted = true
			break
		}
	}

	if !inserted {
		this.attr = append(this.attr, tlv)
	}

	// make sure it is on a 4 byte block
	this.header.length += ((tlv.Length() + 3) / 4) * 4
}

func (this *Message) AddDupAttribute(tlv TLV) {

	this.attr = append(this.attr, tlv)

	// make sure it is on a 4 byte block
	this.header.length += ((tlv.Length() + 3) / 4) * 4
}

func (this *Message) CopyAttributes(other *Message) {

	if other == nil {
		return
	}

	for _, a := range other.attr {
		this.AddAttribute(a)
	}
}

func (this *Message) Attribute(t TLVType) (TLV, error) {
	for _, a := range this.attr {
		if a.Type() == t {
			return a, nil
		}
	}
	return nil, errors.New("Message not found")
}

func (this *Message) Attributes(t TLVType) []TLV {
	ret := []TLV{}
	for _, a := range this.attr {
		if a.Type() == t {
			ret = append(ret, a)
		}
	}
	return ret
}

func (this *Message) String() string {
	ret := this.header.String()
	for _, a := range this.attr {
		ret += "\n" + a.String()
	}
	return ret
}
