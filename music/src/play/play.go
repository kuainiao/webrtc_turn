// play
package mp

import (
	"fmt"
)

type Player interface {
	Playm(source string)
}

func Play(source, mtype string) {
	var p Player
	switch mtype {
	case "MP3":
		p = &MP3Player{}
	default:
		fmt.Println("Unsupported music type", mtype)
		return
	}
	p.Playm(source)
}