package test

import (
	"fmt"
	"github.com/pixelbender/go-sdp/sdp"
	"testing"
)

func TestSdp(t *testing.T) {
	sess, err := sdp.ParseString(`v=0
o=alice 2890844526 2890844526 IN IP4 alice.example.org
s=Example
c=IN IP4 127.0.0.1
t=0 0
a=sendrecv
m=audio 10000 RTP/AVP 0 8
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000`)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("sess", sess)
		fmt.Println("aaa", sess.Media[0].Format[0].Name) // prints PCMU
	}
}
