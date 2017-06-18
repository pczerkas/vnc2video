package main

import (
	"context"
	"log"
	"net"

	vnc "github.com/vtolstov/go-vnc"
)

func main() {
	ln, err := net.Listen("tcp", ":5900")
	if err != nil {
		log.Fatalf("Error listen. %v", err)
	}

	schClient := make(chan vnc.ClientMessage)
	schServer := make(chan vnc.ServerMessage)

	scfg := &vnc.ServerConfig{
		Width:           800,
		Height:          600,
		VersionHandler:  vnc.ServerVersionHandler,
		SecurityHandler: vnc.ServerSecurityHandler,
		SecurityHandlers: []vnc.SecurityHandler{
			&vnc.ClientAuthVeNCrypt02Plain{Username: []byte("test"), Password: []byte("test")},
			&vnc.ClientAuthNone{},
		},
		ClientInitHandler: vnc.ServerClientInitHandler,
		ServerInitHandler: vnc.ServerServerInitHandler,
		Encodings: []vnc.Encoding{
			&vnc.TightPngEncoding{},
			&vnc.CopyRectEncoding{},
			&vnc.RawEncoding{},
		},
		PixelFormat:     vnc.PixelFormat32bit,
		ClientMessageCh: schClient,
		ServerMessageCh: schServer,
		ClientMessages:  vnc.DefaultClientMessages,
		DesktopName:     []byte("vnc proxy"),
	}
	c, err := net.Dial("tcp", "127.0.0.1:5995")
	if err != nil {
		log.Fatalf("Error dial. %v", err)
	}
	cchServer := make(chan vnc.ServerMessage)
	cchClient := make(chan vnc.ClientMessage)

	ccfg := &vnc.ClientConfig{
		VersionHandler:    vnc.ClientVersionHandler,
		SecurityHandler:   vnc.ClientSecurityHandler,
		SecurityHandlers:  []vnc.SecurityHandler{&vnc.ClientAuthNone{}},
		ClientInitHandler: vnc.ClientClientInitHandler,
		ServerInitHandler: vnc.ClientServerInitHandler,
		PixelFormat:       vnc.PixelFormat32bit,
		ClientMessageCh:   cchClient,
		ServerMessageCh:   cchServer,
		ServerMessages:    vnc.DefaultServerMessages,
		Encodings:         []vnc.Encoding{&vnc.RawEncoding{}},
	}

	cc, err := vnc.Connect(context.Background(), c, ccfg)
	if err != nil {
		log.Fatalf("Error dial. %v", err)
	}

	//	scfg.Width = cc.Width()
	//	scfg.Height = cc.Height()
	//	scfg.PixelFormat = cc.PixelFormat()
	go vnc.Serve(context.Background(), ln, scfg)

	defer cc.Close()
	go cc.Handle()

	for {
		select {
		case msg := <-cchServer:
			switch msg.Type() {
			default:
				schServer <- msg
			}
		case msg := <-schClient:
			switch msg.Type() {
			case vnc.SetEncodingsMsgType:
				var encTypes []vnc.EncodingType
				for _, enc := range scfg.Encodings {
					encTypes = append(encTypes, enc.Type())
				}
				msg0 := &vnc.SetEncodings{
					Encodings: encTypes,
				}
				cchClient <- msg0
				/*
					if scfg.Width != cc.Width() || scfg.Height != cc.Height() {
						msg1 := &vnc.FramebufferUpdate{
							Rects: []*vnc.Rectangle{&vnc.Rectangle{
								Width:   cc.Width(),
								Height:  cc.Height(),
								EncType: vnc.EncDesktopSizePseudo,
								Enc:     &vnc.DesktopSizePseudoEncoding{},
							},
							},
						}
						schServer <- msg1
					}
				*/
			default:
				cchClient <- msg
			}
		}
	}
}