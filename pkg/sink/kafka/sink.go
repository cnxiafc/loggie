/*
Copyright 2021 Loggie Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kafka

import (
	"context"
	"fmt"
	"loggie.io/loggie/pkg/sink/codec"

	"github.com/segmentio/kafka-go"
	"loggie.io/loggie/pkg/core/api"
	"loggie.io/loggie/pkg/core/log"
	"loggie.io/loggie/pkg/core/result"
	"loggie.io/loggie/pkg/pipeline"
)

const Type = "kafka"

func init() {
	pipeline.Register(api.SINK, Type, makeSink)
}

func makeSink(info pipeline.Info) api.Component {
	return NewSink()
}

type Sink struct {
	config *Config
	writer *kafka.Writer
	cod    codec.Codec

	topicMatcher [][]string
}

func NewSink() *Sink {
	return &Sink{
		config: &Config{},
	}
}

func (s *Sink) Config() interface{} {
	return s.config
}

func (s *Sink) SetCodec(c codec.Codec) {
	s.cod = c
}

func (s *Sink) Category() api.Category {
	return api.SINK
}

func (s *Sink) Type() api.Type {
	return Type
}

func (s *Sink) String() string {
	return fmt.Sprintf("%s/%s", api.SINK, Type)
}

func (s *Sink) Init(context api.Context) {
	s.topicMatcher = codec.InitMatcher(s.config.Topic)
}

func (s *Sink) Start() {
	c := s.config
	w := &kafka.Writer{
		Addr:         kafka.TCP(c.Brokers...),
		MaxAttempts:  c.MaxAttempts,
		Balancer:     balanceInstance(c.Balance),
		BatchSize:    c.BatchSize,
		BatchBytes:   c.BatchBytes,
		BatchTimeout: c.BatchTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		RequiredAcks: kafka.RequiredAcks(c.RequiredAcks),
		Compression:  compression(c.Compression),
	}

	s.writer = w

	log.Info("kafka-sink start,topic: %s,broker: %v", s.config.Topic, s.config.Brokers)
}

func (s *Sink) Stop() {
	if s.writer != nil {
		_ = s.writer.Close()
	}
}

func (s *Sink) Consume(batch api.Batch) api.Result {
	events := batch.Events()
	l := len(events)
	if l == 0 {
		return nil
	}
	km := make([]kafka.Message, 0, l)
	for _, e := range events {
		msg, err := s.cod.Encode(e)
		if err != nil {
			log.Warn("encode event error: %+v", err)
			return result.Fail(err)
		}
		topic, err := s.selectTopic(msg)
		if err != nil {
			log.Error("select kafka topic error: %+v", err)
			return result.Fail(err)
		}
		km = append(km, kafka.Message{
			Value: msg.Raw,
			Topic: topic,
		})
	}
	err := s.writer.WriteMessages(context.Background(), km...)
	if err != nil {
		log.Error("write to kafka error: %v", err)
		return result.Fail(err)
	}
	return result.Success()
}

func (s *Sink) selectTopic(res *codec.Result) (string, error) {
	return codec.PatternSelect(res, s.config.Topic, s.topicMatcher)
}