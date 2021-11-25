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

package normalize

import (
	"fmt"
	"loggie.io/loggie/pkg/core/api"
	"loggie.io/loggie/pkg/core/log"
	"loggie.io/loggie/pkg/core/sink"
	"loggie.io/loggie/pkg/pipeline"
	"loggie.io/loggie/pkg/util"
	"regexp"
)

const Type = "normalize"

func init() {
	pipeline.Register(api.INTERCEPTOR, Type, makeInterceptor)
}

func makeInterceptor(info pipeline.Info) api.Component {
	return &Interceptor{
		done:         make(chan struct{}),
		pipelineName: info.PipelineName,
		config:       &Config{},
	}
}

type Interceptor struct {
	done         chan struct{}
	pipelineName string
	name         string
	config       *Config
	r            *regexp.Regexp
}

func (i *Interceptor) Config() interface{} {
	return i.config
}

func (i *Interceptor) Category() api.Category {
	return api.INTERCEPTOR
}

func (i *Interceptor) Type() api.Type {
	return Type
}

func (i *Interceptor) String() string {
	return fmt.Sprintf("%s/%s", i.Category(), i.Type())
}

func (i *Interceptor) Init(context api.Context) {
	i.name = context.Name()
	log.Info("regex pattern: %s", i.config.RegexpPattern)
	i.r = util.CompilePatternWithJavaStyle(i.config.RegexpPattern)
}

func (i *Interceptor) Start() {
}

func (i *Interceptor) Stop() {
	close(i.done)
}

func (i *Interceptor) Intercept(invoker sink.Invoker, invocation sink.Invocation) api.Result {
	events := invocation.Batch.Events()
	for _, e := range events {
		body := e.Body()
		if len(body) == 0 {
			continue
		}
		paramsMap := util.MatchGroupWithRegex(i.r, string(body))
		pl := len(paramsMap)
		if pl == 0 {
			continue
		}
		header := e.Header()
		header["systemLogBody"] = paramsMap
	}
	return invoker.Invoke(invocation)
}

func (i *Interceptor) Order() int {
	return i.config.Order
}

func (i *Interceptor) BelongTo() (componentTypes []string) {
	return i.config.BelongTo
}

func (i *Interceptor) IgnoreRetry() bool {
	return true
}