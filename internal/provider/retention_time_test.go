package provider

import (
	"context"
	"encoding/json"
	"testing"

	webclient "axual-webclient"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fiveYears is the largest retention_time, in milliseconds. It does not fit in a 32-bit int,
// so CI also runs these tests as 386.
const fiveYears int64 = 160704000000

func TestTopicConfigRetentionTimeRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &topicConfigResource{provider: AxualProvider{client: &webclient.Client{}}}
	request, err := createTopicConfigRequestFromData(ctx, &topicConfigResourceData{Topic: types.StringValue("t"), Environment: types.StringValue("e"), RetentionTime: types.Int64Value(fiveYears)}, r)
	if err != nil {
		t.Fatal(err)
	}
	var response webclient.TopicConfigResponse
	jsonRoundTrip(t, request, &response)
	var data topicConfigResourceData
	mapTopicConfigResponseToData(ctx, &data, &response)
	if got := data.RetentionTime.ValueInt64(); got != fiveYears {
		t.Errorf("retention_time = %d after a round trip, expected %d", got, fiveYears)
	}
}

func TestEnvironmentRetentionTimeRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &environmentResource{provider: AxualProvider{client: &webclient.Client{}}}
	request, err := createEnvironmentRequestFromData(ctx, &environmentResourceData{Owners: types.StringValue("g"), RetentionTime: types.Int64Value(fiveYears)}, r)
	if err != nil {
		t.Fatal(err)
	}
	var response webclient.EnvironmentResponse
	jsonRoundTrip(t, request, &response)
	var data environmentResourceData
	mapEnvironmentResponseToData(ctx, &data, &response)
	var dataSourceData environmentDataSourceData
	mapEnvironmentDataSourceResponseToData(ctx, &dataSourceData, &response)
	for name, got := range map[string]types.Int64{"axual_environment": data.RetentionTime, "axual_environment data source": dataSourceData.RetentionTime} {
		if got.ValueInt64() != fiveYears {
			t.Errorf("%s retention_time = %d after a round trip, expected %d", name, got.ValueInt64(), fiveYears)
		}
	}
}

// jsonRoundTrip encodes request and decodes the JSON into response, which reads retentionTime under the same key.
func jsonRoundTrip(t *testing.T, request, response any) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, response); err != nil {
		t.Fatal(err)
	}
}
