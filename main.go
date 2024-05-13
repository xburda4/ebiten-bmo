package main

import (
	"fmt"
	_ "image/jpeg"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	socketConnection net.Listener
	//testVar          int
	sockChannel chan []byte

	defaultImageLength = 50
)

func listener() {
	for {
		conn, err := socketConnection.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(conn net.Conn) {
			defer conn.Close()
			// Create a buffer for incoming data.
			buf := make([]byte, 4096)

			// Read data from the connection.
			_, err := conn.Read(buf)
			if err != nil {
				log.Fatal(err)
			}
			//fmt.Println(strings.TrimSpace(string(buf[:])))
			sockChannel <- buf
		}(conn)
	}
}

func translateSocketMessageToImage(msg string) string {
	switch {
	case strings.HasPrefix(msg, "joke"):
		return "bmo-joke.jpg"
	case strings.HasPrefix(msg, "greet"):
		return "bmo-hello.jpg"
	case strings.HasPrefix(msg, "sob-reaction"):
		return "bmo-shocked.jpg"
	}

	return "unknown"
}

type Game struct {
	showState  []string
	imagesLeft int
	curImage   string
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	var sockMsg string
	var msg []byte
	select {
	case msg = <-sockChannel:
		sockMsg = strings.TrimSuffix(strings.TrimSpace(string(msg[:])), "\n")
		g.showState = append(g.showState, sockMsg)
	default:
		sockMsg = "unknown"
	}

	imageName := translateSocketMessageToImage(sockMsg)
	if imageName != "unknown" {
		if len(g.showState) == 0 {
			g.imagesLeft = defaultImageLength
		}
		g.showState = append(g.showState, imageName)
	}

	if g.imagesLeft > 0 {
		g.imagesLeft--
		ebImage, _, err := ebitenutil.NewImageFromFile(fmt.Sprintf("images/%s", g.showState[0]))
		if err != nil {
			log.Fatal(err)
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, -256)
		screen.DrawImage(ebImage, op)
	} else {
		var ebImage *ebiten.Image
		var err error

		if len(g.showState) == 0 {
			ebImage, _, err = ebitenutil.NewImageFromFile("images/bmo-default.jpg")
			if err != nil {
				log.Fatal(err)
			}

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, -256)
			screen.DrawImage(ebImage, op)
		} else {
			g.showState = g.showState[1:]

			if len(g.showState) != 0 {
				g.imagesLeft = defaultImageLength
			}
		}
	}

	//testVar++
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("Hello, World! x %d", testVar))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 720
}

func main() {
	conn, err := net.Listen("unix", "/tmp/bmo.sock")
	if err != nil {
		log.Fatal(err)
	}
	socketConnection = conn
	sockChannel = make(chan []byte)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Remove("/tmp/bmo.sock")
		os.Exit(1)
	}()

	go listener()

	ebiten.SetFullscreen(false)
	ebiten.SetWindowTitle("Hello, World!")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
