package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesHandlerAggregatesUnexpectedSSEForNonStream(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	body := `event: response.created
data: {"type":"response.created","response":{"id":"resp_test","object":"response","created_at":1,"status":"in_progress","model":"gpt-5.4","output":[]}}

event: response.completed
data: {"type":"response.completed","response":{"id":"resp_test","object":"response","created_at":1,"status":"completed","model":"gpt-5.4","output":[],"usage":{"input_tokens":7,"output_tokens":5,"total_tokens":12}}}

`

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, err := OaiResponsesHandler(c, &relaycommon.RelayInfo{}, resp)

	require.Nil(t, err)
	require.Equal(t, 7, usage.PromptTokens)
	require.Equal(t, 5, usage.CompletionTokens)
	require.Equal(t, 12, usage.TotalTokens)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), `"id":"resp_test"`)
	require.NotContains(t, recorder.Body.String(), "event:")
}
