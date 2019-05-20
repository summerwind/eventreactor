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
	"net/http"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PipelineValidator struct {
	decoder *admission.Decoder
}

func (v *PipelineValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pipeline := &v1alpha1.Pipeline{}

	err := v.decoder.Decode(req, pipeline)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err = pipeline.Validate()
	if err != nil {
		return admission.Denied(err.Error())
	}

	return admission.Allowed("")
}

func (v *PipelineValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
