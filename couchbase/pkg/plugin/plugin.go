package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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
    log.DefaultLogger.Info("Processing query", "refId", q.RefID)
    response.Responses[q.RefID] = d.query(ctx, req.PluginContext, q)
	}

	return response, nil
}

type QueryRequest struct {
  Query string `json:"query"`
  Fts bool `json:"fts"`
}

func (d *CouchbaseDatasource) query(_ context.Context, pCtx backend.PluginContext, q backend.DataQuery) backend.DataResponse {
	response := backend.DataResponse{}


  range_filter := fmt.Sprintf("d.time > %d AND d.time < %d", q.TimeRange.From.UnixMilli(), q.TimeRange.To.UnixMilli())
  var query_data QueryRequest
  query_response := &backend.DataResponse{}
  if err := json.Unmarshal(q.JSON, &query_data); err != nil {
    log.DefaultLogger.Error("Failed to unmarshal request json", "error", err)
    query_response.Error = err;
    query_response.Frames = make(data.Frames, 0)
  } else {
    log.DefaultLogger.Info(fmt.Sprintf("Query data: %+v", query_data))
    query_string := query_data.Query
    if err := validateQuery(strings.ToLower(query_string)); err != nil {
      response.Error = err
      return response
    }
    log.DefaultLogger.Info("Unmarshalled json", "query_string", query_string)
    query_string = "SELECT * FROM (" + query_string + ") AS d WHERE " + range_filter

    log.DefaultLogger.Info("Querying couchbase", "query_string", query_string)
    if res, qerr := d.Cluster.Query(query_string, nil); qerr != nil {
      log.DefaultLogger.Error(fmt.Sprintf("Query failed: %+v", qerr))
      query_response.Error = qerr
      return *query_response
    } else {
      frame := data.NewFrame("response")

      var row map[string]interface{}
      for res.Next() {
        res.Row(row)
        for key, val := range row {
          frame.Fields = append(frame.Fields, data.NewField(key, nil, val))
        }
      }
      response.Frames = append(response.Frames, frame)
    }
  }

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

func locateClause(query string, clause string) int {
  inString := false
  escaped := false
  clen := len(clause)

  for i := 0; i < len(query); i++ {
    if c := query[i]; c == '"' {
      if !escaped {
        inString = !inString
      }
    } else if inString {
      if c == '\\' {
        escaped = !escaped
      } else {
        escaped = false
      }
    } else if i > clen && strings.EqualFold(query[i-clen:i], "where") {
      return i-5
    }
  }

  return -1;
}

func validateQuery(query string) error {
  if strings.Contains(query, "limit") {
    return errors.New("Limit clause is not supported")
  } else if !strings.Contains(query, "time") && !strings.Contains(query, "*") {
    return errors.New("Please map a timestamp to `time` field")
  }
  return nil
}
