package main

import (
	"bytes"
	_ "embed"
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
	sockChannel      chan []byte

	defaultImageLength = 50

	//go:embed images/bmo-hello.jpg
	bmoHello []byte
	//go:embed images/bmo-joke.jpg
	bmoJoke []byte
	//go:embed images/bmo-shocked.jpg
	bmoShocker []byte
	//go:embed images/bmo-default.jpg
	bmoDefault []byte
	//go:embed images/bmo-greet.jpg
	bmoGreet []byte
	//go:embed images/bmo-heart.jpg
	bmoHeart []byte
	//go:embed images/bmo-bububaba.jpg
	bmoBubuBaba []byte
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
			sockChannel <- buf
		}(conn)
	}
}

func translateSocketMessageToImage(msg string) []byte {
	switch {
	case strings.HasPrefix(msg, "joke"):
		return bmoJoke
	case strings.HasPrefix(msg, "greet"):
		return bmoHello
	case strings.HasPrefix(msg, "sob-reaction"):
		return bmoShocker
	case strings.HasPrefix(msg, "pretty-well-reaction"):
		return bmoHeart
	case strings.HasPrefix(msg, "message"):
		return bmoBubuBaba
	}

	return []byte{}
}

type Game struct {
	showState  [][]byte
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
	default:
		sockMsg = "unknown"
	}

	imageBytes := translateSocketMessageToImage(sockMsg)
	if len(imageBytes) != 0 {
		if len(g.showState) == 0 {
			g.imagesLeft = defaultImageLength
		}
		g.showState = append(g.showState, imageBytes)
	}

	if g.imagesLeft > 0 {
		g.imagesLeft--
		ebImage, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(g.showState[0]))
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
			ebImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(bmoDefault))
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
	ebiten.SetWindowSize(480, 640)
	ebiten.SetWindowTitle("I'm Johny!")
	if err := ebiten.RunGame(&Game{
		showState: make([][]byte, 0),
	}); err != nil {
		log.Fatal(err)
	}
}
