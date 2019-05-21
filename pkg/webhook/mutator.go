/*
Copyright 2019 The Event Reactor Authors.

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

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Mutator struct {
	decoder *admission.Decoder
}

func (m *Mutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	var res admission.Response

	switch req.Kind.Kind {
	case "Pipeline":
		res = m.mutatePipeline(req)
	default:
		res = admission.Errored(http.StatusBadRequest, errors.New("unexpected resource"))
	}

	return res
}

func (m *Mutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

func (m *Mutator) mutatePipeline(req admission.Request) admission.Response {
	var trigger string

	pipeline := &v1alpha1.Pipeline{}

	err := m.decoder.Decode(req, pipeline)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pipeline.ObjectMeta.Labels == nil {
		pipeline.ObjectMeta.Labels = map[string]string{}
	}

	if pipeline.Spec.Trigger.Pipeline != nil {
		trigger = v1alpha1.TriggerTypePipeline
	}

	if pipeline.Spec.Trigger.Event != nil {
		trigger = v1alpha1.TriggerTypeEvent
		pipeline.ObjectMeta.Labels[v1alpha1.KeyEventType] = pipeline.Spec.Trigger.Event.Type
	}

	if trigger == "" {
		return admission.Errored(http.StatusInternalServerError, errors.New("invalid trigger"))
	}

	pipeline.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = trigger

	p, err := json.Marshal(pipeline)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, p)
}
