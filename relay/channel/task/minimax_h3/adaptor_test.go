/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package minimax_h3

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTaskContext(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx
}

func TestValidateAndBuildRequestBodySupportsNumericSeconds(t *testing.T) {
	ctx := newTaskContext(`{"model":"minimax-h3-original-768p","prompt":"a paper boat","seconds":8,"aspect_ratio":"9:16"}`)
	defer common.CleanupBodyStorage(ctx)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://video.ctmoai.com", UpstreamModelName: "minimax-h3-original-768p"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"minimax-h3-original-768p","prompt":"a paper boat","seconds":8,"aspect_ratio":"9:16","size":"768x1376"}`, string(data))
}

func TestValidateRejectsInvalidReferenceAndDuration(t *testing.T) {
	ctx := newTaskContext(`{"model":"minimax-h3-original-cf-2k","prompt":"a portrait","seconds":3}`)
	defer common.CleanupBodyStorage(ctx)
	adaptor := &TaskAdaptor{}
	err := adaptor.ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
	require.NotNil(t, err)
	assert.Equal(t, "invalid_seconds", err.Code)
}

func TestValidateSizeMatchesMiniMaxH3ModelCapability(t *testing.T) {
	testCases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "accepts 768p dimensions",
			body: `{"model":"minimax-h3-original-768p","prompt":"a paper boat","seconds":8,"aspect_ratio":"16:9","size":"1376x768"}`,
		},
		{
			name:    "rejects 1080p dimensions for 768p model",
			body:    `{"model":"minimax-h3-original-768p","prompt":"a paper boat","seconds":8,"aspect_ratio":"16:9","size":"1920x1088"}`,
			wantErr: true,
		},
		{
			name: "accepts 1080p dimensions",
			body: `{"model":"minimax-h3-original-1080p","prompt":"a paper boat","seconds":8,"aspect_ratio":"16:9","size":"1920x1088"}`,
		},
		{
			name:    "rejects 768p dimensions for 1080p model",
			body:    `{"model":"minimax-h3-original-1080p","prompt":"a paper boat","seconds":8,"aspect_ratio":"16:9","size":"1376x768"}`,
			wantErr: true,
		},
		{
			name: "accepts 2K for cf 2K model",
			body: `{"model":"minimax-h3-original-cf-2k","prompt":"a paper boat","seconds":8,"images":["https://example.com/reference.png"],"size":"2K"}`,
		},
		{
			name:    "rejects 4K for cf 2K model",
			body:    `{"model":"minimax-h3-original-cf-2k","prompt":"a paper boat","seconds":8,"images":["https://example.com/reference.png"],"size":"4K"}`,
			wantErr: true,
		},
		{
			name: "accepts 4K for cf 4K model",
			body: `{"model":"minimax-h3-original-cf-4k","prompt":"a paper boat","seconds":8,"images":["https://example.com/reference.png"],"size":"4K"}`,
		},
		{
			name:    "rejects 2K for cf 4K model",
			body:    `{"model":"minimax-h3-original-cf-4k","prompt":"a paper boat","seconds":8,"images":["https://example.com/reference.png"],"size":"2K"}`,
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := newTaskContext(testCase.body)
			defer common.CleanupBodyStorage(ctx)

			err := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
			if testCase.wantErr {
				require.NotNil(t, err)
				assert.Equal(t, "invalid_size", err.Code)
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestValidateAndBuildRequestBodyIncludesMultipartReferenceImage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "minimax-h3-original-cf-2k"))
	require.NoError(t, writer.WriteField("prompt", "make this image move"))
	require.NoError(t, writer.WriteField("seconds", "6"))
	require.NoError(t, writer.WriteField("aspect_ratio", "1:1"))
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="input_reference[]"; filename="reference.png"`)
	partHeader.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = part.Write([]byte("reference-image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	defer common.CleanupBodyStorage(ctx)

	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://video.ctmoai.com", UpstreamModelName: "minimax-h3-original-cf-2k"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	requestBody, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	var payload requestPayload
	require.NoError(t, common.Unmarshal(data, &payload))
	assert.Equal(t, []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("reference-image"))}, payload.Images)
}

func TestParseTaskResultTreatsUnknownAsInProgress(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"task_1","status":"unknown","progress":40}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, result.Status)
	assert.Equal(t, "40%", result.Progress)
}
