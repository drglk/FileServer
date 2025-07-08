package utils

import (
	"encoding/json"
	"fileserver/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteJSONError(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteJSONError(rr, http.StatusInternalServerError, models.ErrInternal.Error())

	var body map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	assert.NoError(t, err)

	errorObj, ok := body["error"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, float64(http.StatusInternalServerError), errorObj["code"])
	assert.Equal(t, models.ErrInternal.Error(), errorObj["text"])
}

func TestWriteStatusError(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	WriteStatusError(rr, http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, 0, rr.Body.Len())

}
