package dialer

import (
	"bytes"
	cim "cirno-im"
	"context"
	"fmt"
	"net"
	"time"

	"cirno-im/logger"
	"cirno-im/wire"
	"cirno-im/wire/pkt"
	"cirno-im/wire/token"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type ClientDialer struct {
	AppSecret string
}

func (d *ClientDialer) DialAndHandshake(ctx cim.DialerContext) (net.Conn, error) {
	// 1. 拨号
	conn, _, _, err := ws.Dial(context.TODO(), ctx.Address)
	if err != nil {
		return nil, err
	}
	if d.AppSecret == "" {
		d.AppSecret = token.DefaultSecret
	}
	// 2. 直接使用封装的JWT包生成一个token
	tk, err := token.Generate(d.AppSecret, &token.Token{
		Account: ctx.Id,
		App:     "cim",
		Exp:     time.Now().AddDate(0, 0, 1).Unix(),
	})
	if err != nil {
		return nil, err
	}
	// 3. 发送一条CommandLoginSignIn消息
	loginreq := pkt.New(wire.CommandLoginSignIn).WriteBody(&pkt.LoginRequest{
		Token: tk,
	})
	err = wsutil.WriteClientBinary(conn, pkt.Marshal(loginreq))
	if err != nil {
		return nil, err
	}

	// wait resp
	_ = conn.SetReadDeadline(time.Now().Add(ctx.Timeout))
	frame, err := ws.ReadFrame(conn)
	if err != nil {
		return nil, err
	}
	ack, err := pkt.MustReadLogicPkt(bytes.NewBuffer(frame.Payload))
	if err != nil {
		return nil, err
	}
	// 4. 判断是否登录成功
	if ack.Status != pkt.Status_Success {
		return nil, fmt.Errorf("login failed: %v", &ack.Header)
	}
	var resp = new(pkt.LoginResponse)
	_ = ack.ReadBody(resp)

	logger.Debug("logined ", resp.GetChannelID())
	return conn, nil
}
