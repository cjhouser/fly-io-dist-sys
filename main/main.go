package main

import (
	"log"

	goc "github.com/cjhouser/fly-io-dist-sys/goc"
)

func main() {
	//w := echo.Init()
	//w := uidg.Init()
	//w := broadcast.Init()
	w := goc.Init()

	if err := w.Node.Run(); err != nil {
		log.Fatal(err)
	}
}
