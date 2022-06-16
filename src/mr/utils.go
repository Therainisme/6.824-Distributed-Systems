package mr

import (
	"log"
)

const debug = false

func print(format string, v ...interface{}) {
	if debug {
		log.Printf(format+"\n", v...)
	}
}
