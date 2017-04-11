// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/Mirantis/k8s-AppController/pkg/client"
	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/report"
)

type Flow struct {
	Base
	flow          *client.Flow
	context       interfaces.GraphContext
	status        interfaces.ResourceStatus
	generatedName string
	originalName  string
}

type flowTemplateFactory struct{}

var counter uint32

// Returns wrapped resource name if it was a flow
func (flowTemplateFactory) ShortName(definition client.ResourceDefinition) string {
	if definition.Flow == nil {
		return ""
	}
	return definition.Flow.Name
}

// k8s resource kind that this fabric supports
func (flowTemplateFactory) Kind() string {
	return "flow"
}

// New returns a new object wrapped as Resource
func (flowTemplateFactory) New(def client.ResourceDefinition, c client.Interface, gc interfaces.GraphContext) interfaces.Resource {
	newFlow := parametrizeResource(def.Flow, gc).(*client.Flow)

	return report.SimpleReporter{
		BaseResource: &Flow{
			Base:          Base{def.Meta},
			flow:          newFlow,
			context:       gc,
			status:        interfaces.ResourceNotReady,
			generatedName: fmt.Sprintf("%s-%v", newFlow.Name, atomic.AddUint32(&counter, 1)),
			originalName:  def.Flow.Name,
		}}
}

// NewExisting returns a new object based on existing one wrapped as Resource
func (flowTemplateFactory) NewExisting(name string, c client.Interface, gc interfaces.GraphContext) interfaces.Resource {
	log.Fatal("Cannot depend on flow that has no resource definition")
	return nil
}

// Identifier of the object
func (f Flow) Key() string {
	return "flow/" + f.generatedName
}

// Triggers the flow deployment like it was the resource creation
func (f *Flow) Create() error {
	args := map[string]string{}
	for arg := range f.flow.Parameters {
		val := f.context.GetArg(arg)
		if val != "" {
			args[arg] = val
		}
	}
	options := interfaces.DependencyGraphOptions{
		FlowName: f.originalName,
		Args:     args,
	}
	graph, err := f.context.Scheduler().BuildDependencyGraph(options)
	if err != nil {
		return err
	}
	go func() {
		stopChan := make(chan struct{})
		graph.Deploy(stopChan)
		f.status = interfaces.ResourceReady
	}()
	return nil
}

// Deletes resources allocated to the flow
func (f Flow) Delete() error {
	return errors.New("Not supported yet")
}

// Current status of the flow deployment
func (f Flow) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	return f.status, nil
}