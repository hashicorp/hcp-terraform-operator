// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

// rawRunStatusKey is the run attribute the transport adds to preserve a run status
// that the go-tfe v2 status enum cannot represent.
const rawRunStatusKey = "x-operator-raw-status"

// unrepresentableRunStatuses are the run statuses missing from the go-tfe v2 enum.
// The enum parser turns an unknown value into nil, which loses the original status.
var unrepresentableRunStatuses = []string{
	appv1alpha2.RunPostPlanAwaitingDecision,
}

// runStatusTransport wraps a http.RoundTripper and tags JSON responses carrying a run status
// that go-tfe v2 cannot parse with an extra attribute holding the original value.
// Kiota keeps unknown attributes in the model's additional data, so the status can be
// read back with runAttributesStatus.
type runStatusTransport struct {
	next http.RoundTripper
}

func newRunStatusTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &runStatusTransport{next: next}
}

func (t *runStatusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return resp, nil
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	for _, s := range unrepresentableRunStatuses {
		old := []byte(`"status":"` + string(s) + `"`)
		if bytes.Contains(body, old) {
			tagged := append(append([]byte{}, old...), []byte(`,"`+rawRunStatusKey+`":"`+string(s)+`"`)...)
			body = bytes.ReplaceAll(body, old, tagged)
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	if resp.Header.Get("Content-Length") != "" {
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}
	return resp, nil
}
