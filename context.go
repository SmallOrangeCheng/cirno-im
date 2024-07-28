package cim

import (
	"cirno-im/logger"
	"cirno-im/wire"
	"cirno-im/wire/pkt"
	"sync"

	"google.golang.org/protobuf/proto"
)

type Session interface {
	GetChannelID() string
	GetGateID() string
	GetAccount() string
	GetZone() string
	GetIsp() string
	GetRemoteIP() string
	GetDevice() string
	GetApp() string
	GetTags() []string
}

type Context interface {
	Dispatcher
	SessionStorage
	Header() *pkt.Header
	ReadBody(val proto.Message) error
	Session() Session
	RespWithError(status pkt.Status, err error) error
	Resp(status pkt.Status, body proto.Message) error
	Dispatch(body proto.Message, recvs ...*Location) error
}

type HandlerFunc func(ctx Context)

type HandlerChain []HandlerFunc

type ContextImpl struct {
	sync.Mutex
	Dispatcher
	SessionStorage

	handlers HandlerChain
	index    int
	request  *pkt.LogicPkt
	session  Session
}

func BuildContext() Context {
	return &ContextImpl{}
}

func (c *ContextImpl) Next() {
	if c.index >= len(c.handlers) {
		return
	}
	f := c.handlers[c.index]
	f(c)
	c.index++
}

func (c *ContextImpl) Header() *pkt.Header {
	return &c.request.Header
}

func (c *ContextImpl) ReadBody(val proto.Message) error {
	return c.request.ReadBody(val)
}

func (c *ContextImpl) Session() Session {
	if c.session == nil {
		server, _ := c.request.GetMeta(wire.MetaDestServer)
		c.session = &pkt.Session{
			ChannelID: c.request.ChannelID,
			GateID:    server.(string),
			Tags:      []string{"AutoGenerated"},
		}
	}
	return c.session
}

func (c *ContextImpl) RespWithError(status pkt.Status, err error) error {
	return c.Resp(status, &pkt.ErrorResponse{Message: err.Error()})
}

func (c *ContextImpl) Resp(status pkt.Status, body proto.Message) error {
	packet := pkt.NewFrom(&c.request.Header)
	packet.Status = status
	packet.WriteBody(body)
	packet.Flag = pkt.Flag_Response
	logger.Debugf("<-- Resp to %s command:%s  status: %v body: %s", c.Session().GetAccount(), &c.request.Header, status, body)
	err := c.Push(c.session.GetGateID(), []string{c.session.GetChannelID()}, packet)
	if err != nil {
		return err
	}
	return nil
}

// Dispatch the packet to the Destination of request,
// the header flag of this packet will be set with FlagDelivery
// exceptMe:  exclude self if self is false
func (c *ContextImpl) Dispatch(body proto.Message, recvs ...*Location) error {
	if len(recvs) == 0 {
		return nil
	}
	packet := pkt.NewFrom(&c.request.Header)
	packet.Flag = pkt.Flag_Push
	packet.WriteBody(body)
	logger.Debugf("<-- Dispatch to %d users command:%s", len(recvs), &c.request.Header)

	group := make(map[string][]string)
	for _, recv := range recvs {
		if recv.ChannelID == c.Session().GetChannelID() {
			continue
		}
		if _, ok := group[recv.GateID]; !ok {
			group[recv.GateID] = make([]string, 0)
		}
		group[recv.GateID] = append(group[recv.GateID], recv.ChannelID)
	}
	for gateway, ids := range group {
		err := c.Push(gateway, ids, packet)
		if err != nil {
			logger.Error(err.Error())
			return err 
			///todo i think there maybe has a bug
		}
	}
	return nil
}

func (c *ContextImpl) reset() {
	c.request = nil
	c.index = 0
	c.handlers = nil
	c.session = nil
}