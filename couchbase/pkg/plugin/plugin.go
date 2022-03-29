package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/live"
)

// Make sure CouchbaseDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*CouchbaseDatasource)(nil)
	_ backend.CheckHealthHandler    = (*CouchbaseDatasource)(nil)
	_ backend.StreamHandler         = (*CouchbaseDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*CouchbaseDatasource)(nil)
)

// NewCouchbaseDatasource creates a new datasource instance.
func NewCouchbaseDatasource(instance backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
  var settings map[string]string
  if e := json.Unmarshal(instance.JSONData, &settings); e != nil {
    return &backend.CheckHealthResult{
      Status: backend.HealthStatusError,
      Message: "failed to parse settings",
    }, e
  }
  password := instance.DecryptedSecureJSONData["password"]
  if cluster, err := gocb.Connect(
    settings["host"],
    gocb.ClusterOptions{
      Authenticator: gocb.PasswordAuthenticator{
        Username: settings["username"],
        Password: password,
      },
    },
  ); err != nil {
    log.DefaultLogger.Error("Failed to connect to cluster", err)
    return nil, err
  } else {
    log.DefaultLogger.Info("Connected to the cluster, executing a test query...")
    bucket := cluster.Bucket(settings["bucket"])
    if be := bucket.WaitUntilReady(5*time.Second, nil); be != nil {
      log.DefaultLogger.Error("Bucket is not ready", "bucket", settings["bucket"], be.Error())
      return nil, be
    }

    return &CouchbaseDatasource{
      *cluster,
      *bucket,
    }, nil
  }
}

// CouchbaseDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type CouchbaseDatasource struct{
  Cluster gocb.Cluster
  Bucket gocb.Bucket
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewCouchbaseDatasource factory function.
func (d *CouchbaseDatasource) Dispose() {
	// Clean up datasource instance resources.
  // nothing to do
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *CouchbaseDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	WithStreaming bool `json:"withStreaming"`
}

func (d *CouchbaseDatasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	response := backend.DataResponse{}

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	// create data frame response.
	frame := data.NewFrame("response")

	// add fields.
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
		data.NewField("values", nil, []int64{10, 20}),
	)

	// If query called with streaming on then return a channel
	// to subscribe on a client-side and consume updates from a plugin.
	// Feel free to remove this if you don't need streaming for your datasource.
	if qm.WithStreaming {
		channel := live.Channel{
			Scope:     live.ScopeDatasource,
			Namespace: pCtx.DataSourceInstanceSettings.UID,
			Path:      "stream",
		}
		frame.SetMeta(&data.FrameMeta{Channel: channel.String()})
	}

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *CouchbaseDatasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
  if pings, be := d.Bucket.Ping(&gocb.PingOptions{
      ReportID: "grafana-test",
      ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue},
    }); be != nil {
      log.DefaultLogger.Error("Failed to ping the cluster", "error", be.Error())
      return &backend.CheckHealthResult{
        Status: backend.HealthStatusError,
        Message: be.Error(),
      }, be
    } else {
     for _, responses := range pings.Services{
       for _, response := range responses {
         if response.State != gocb.PingStateOk {
           return &backend.CheckHealthResult{
             Status: backend.HealthStatusError,
             Message: fmt.Sprintf("Node %s at remote %s ping response was not ok: %s", response.ID, response.Remote, response.Error),
           }, nil
         }
       }
     }
    }

    log.DefaultLogger.Info("Verified cluster connectivity")
    return &backend.CheckHealthResult{
      Status: backend.HealthStatusOk,
      Message: "Ping OK",
    }, nil
  }


// SubscribeStream is called when a client wants to connect to a stream. This callback
// allows sending the first message.
func (d *CouchbaseDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Info("SubscribeStream called", "request", req)

	status := backend.SubscribeStreamStatusPermissionDenied
	if req.Path == "stream" {
		// Allow subscribing only on expected path.
		status = backend.SubscribeStreamStatusOK
	}
	return &backend.SubscribeStreamResponse{
		Status: status,
	}, nil
}

// RunStream is called once for any open channel.  Results are shared with everyone
// subscribed to the same channel.
func (d *CouchbaseDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Info("RunStream called", "request", req)

	// Create the same data frame as for query data.
	frame := data.NewFrame("response")

	// Add fields (matching the same schema used in QueryData).
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, make([]time.Time, 1)),
		data.NewField("values", nil, make([]int64, 1)),
	)

	counter := 0

	// Stream data frames periodically till stream closed by Grafana.
	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("Context done, finish streaming", "path", req.Path)
			return nil
		case <-time.After(time.Second):
			// Send new data periodically.
			frame.Fields[0].Set(0, time.Now())
			frame.Fields[1].Set(0, int64(10*(counter%2+1)))

			counter++

			err := sender.SendFrame(frame, data.IncludeAll)
			if err != nil {
				log.DefaultLogger.Error("Error sending frame", "error", err)
				continue
			}
		}
	}
}

// PublishStream is called when a client sends a message to the stream.
func (d *CouchbaseDatasource) PublishStream(_ context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	log.DefaultLogger.Info("PublishStream called", "request", req)

	// Do not allow publishing at all.
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
