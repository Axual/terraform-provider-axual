package webclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindTopicConfigsByTopicReadsTheListAndEscapesTheTopicUrl(t *testing.T) {
	var gotStream string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotStream = r.URL.Query().Get("stream")
		_, _ = w.Write([]byte(`{"_embedded":{"stream_configs":[
			{"uid":"c1","_embedded":{"environment":{"shortName":"dev"}}},
			{"uid":"c2","_embedded":{"environment":{"shortName":"prd"}}}]}}`))
	}))
	t.Cleanup(server.Close)
	client := &Client{HTTPClient: server.Client(), ApiURL: server.URL}

	found, err := client.FindTopicConfigsByTopic("https://pm/api/streams/t1?x=1&y=2")
	if err != nil {
		t.Fatalf("find error = %v", err)
	}
	if gotStream != "https://pm/api/streams/t1?x=1&y=2" {
		t.Errorf("stream parameter = %q, expected the full topic URL", gotStream)
	}
	configs := found.Embedded.TopicConfigs
	if len(configs) != 2 || configs[0].Uid != "c1" || configs[1].Embedded.Environment.ShortName != "prd" {
		t.Errorf("configs = %+v, expected c1 on dev and c2 on prd", configs)
	}
}
