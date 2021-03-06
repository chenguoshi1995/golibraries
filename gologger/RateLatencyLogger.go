package gologger

import (
	"time"
)

// updatePacket : Struct that holds message updates
type updatePacket struct {
	identifier string
	labels     []string
	value      int64
}

type messageAdder struct {
	name   string
	metric IMetricVec
}

// RateLatencyLogger : Logger that tracks multiple messages & prints to console
type RateLatencyLogger struct {
	messages       map[string]IMetricVec // Map that holds all module's messages
	updateTunnel   chan updatePacket     // Channel which updates latency in message
	countIncTunnel chan updatePacket
	countSubTunnel chan updatePacket
	countSetTunnel chan updatePacket
	addMsgTunnel   chan messageAdder
	logger         *CustomLogger
	isRan          bool
}

var rateLatencyLogger *RateLatencyLogger

// Tic starts the timer
func (mgl *RateLatencyLogger) Tic() time.Time {
	return time.Now()
}

// Toc calculates the time elapsed since Tic() and stores in the Message
func (mgl *RateLatencyLogger) Toc(start time.Time, identifier string, labels ...string) {
	if mgl.isRan {
		elapsed := int64(time.Since(start) / 1000)
		mgl.updateTunnel <- updatePacket{identifier, labels, elapsed}
	}
}

//IncVal is used for counters and gauges
func (mgl *RateLatencyLogger) IncVal(value int64, identifier string, labels ...string) {
	if mgl.isRan {
		mgl.countIncTunnel <- updatePacket{identifier, labels, value}
	}
}

//SubVal is used for counters and gauges
func (mgl *RateLatencyLogger) SubVal(value int64, identifier string, labels ...string) {
	if mgl.isRan {
		mgl.countSubTunnel <- updatePacket{identifier, labels, value}
	}
}

//SetVal is used for counters and gauges
func (mgl *RateLatencyLogger) SetVal(value int64, identifier string, labels ...string) {
	if mgl.isRan {
		mgl.countSetTunnel <- updatePacket{identifier, labels, value}
	}
}

// run : Starts the logger in a go routine.
// Calling this multiple times doesn't have any effect
func (mgl *RateLatencyLogger) run() {

	go func() {
		for {
			select {
			case addMessage := <-mgl.addMsgTunnel:
				_, ok := mgl.messages[addMessage.name]
				if !ok {
					mgl.messages[addMessage.name] = addMessage.metric
				}
			case packet := <-mgl.updateTunnel:
				msg, ok := mgl.messages[packet.identifier]
				if !ok {
					mgl.logger.LogErrorWithoutError("wrong identifier passed. Could not find metric logger with identifier " + packet.identifier)
					continue
				}
				msg.UpdateTime(packet.value, packet.labels...)
			case packet := <-mgl.countIncTunnel:
				msg, ok := mgl.messages[packet.identifier]
				if !ok {
					mgl.logger.LogErrorWithoutError("wrong identifier passed. Could not find metric logger with identifier " + packet.identifier)
					continue
				}
				msg.AddValue(packet.value, packet.labels...)
			case packet := <-mgl.countSubTunnel:
				msg, ok := mgl.messages[packet.identifier]
				if !ok {
					mgl.logger.LogErrorWithoutError("wrong identifier passed. Could not find metric logger with identifier " + packet.identifier)
					continue
				}
				msg.SubValue(packet.value, packet.labels...)
			case packet := <-mgl.countSetTunnel:
				msg, ok := mgl.messages[packet.identifier]
				if !ok {
					mgl.logger.LogErrorWithoutError("wrong identifier passed. Could not find metric logger with identifier " + packet.identifier)
					continue
				}
				msg.SetValue(packet.value, packet.labels...)
			}
		}
	}()
	mgl.isRan = true
}

// AddNewMetric sets New message initialisation function
func (mgl *RateLatencyLogger) AddNewMetric(messageIdentifier string, newMessage IMetricVec) {
	mgl.addMsgTunnel <- messageAdder{name: messageIdentifier, metric: newMessage}
}

// RateLatencyOption sets a parameter for the RateLatencyLogger
type RateLatencyOption func(rl *RateLatencyLogger)

// SetLogger sets the output logger.
// Default is stderr
func SetLogger(logger *CustomLogger) RateLatencyOption {
	return func(rl *RateLatencyLogger) {
		rl.logger = logger
	}
}

// NewRateLatencyLogger : returns a new RateLatencyLogger.
// When no options are given, it returns a RateLatencyLogger with default settings.
// Default logger is default custom logger.
func NewRateLatencyLogger(options ...RateLatencyOption) IMultiLogger {
	if rateLatencyLogger != nil {
		return rateLatencyLogger
	}
	rateLatencyLogger = &RateLatencyLogger{
		messages:       map[string]IMetricVec{},
		updateTunnel:   make(chan updatePacket, 10000),
		countIncTunnel: make(chan updatePacket, 10000),
		countSubTunnel: make(chan updatePacket, 10000),
		countSetTunnel: make(chan updatePacket, 10000),
		addMsgTunnel:   make(chan messageAdder, 10),
		logger:         nil,
	}

	for _, option := range options {
		option(rateLatencyLogger)
	}

	if rateLatencyLogger.logger == nil {
		rateLatencyLogger.logger = NewLogger()
	}
	rateLatencyLogger.run()
	return rateLatencyLogger
}
