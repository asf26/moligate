package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUploadStableVideoMediaUsesSelectedNewAPIChannel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sd-media/upload", r.URL.Path)
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		assert.Equal(t, "images", r.FormValue("type"))
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://media.example/reference.png"}`))
	}))
	defer upstream.Close()

	previousDB := model.DB
	previousMemoryCache := common.MemoryCacheEnabled
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
	})
	common.MemoryCacheEnabled = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	baseURL := upstream.URL
	channel := model.Channel{Id: 91, Name: "stable-video", Type: constant.ChannelTypeNewAPI, Key: "upstream-key", Status: common.ChannelStatusEnabled, BaseURL: &baseURL}
	require.NoError(t, db.Create(&channel).Error)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "seedance2.0-stable-full-720p"))
	require.NoError(t, writer.WriteField("type", "images"))
	part, err := writer.CreateFormFile("file", "reference.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("reference"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/media", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	common.SetContextKey(context, constant.ContextKeyOriginalModel, "seedance2.0-stable-full-720p")
	common.SetContextKey(context, constant.ContextKeyChannelType, constant.ChannelTypeNewAPI)
	common.SetContextKey(context, constant.ContextKeyChannelId, channel.Id)
	common.SetContextKey(context, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(context, constant.ContextKeyChannelKey, "upstream-key")
	defer common.CleanupBodyStorage(context)

	UploadStableVideoMedia(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"url":"https://media.example/reference.png"}`, recorder.Body.String())
}
