package plugin

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"runtime/debug"
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

var channels = make(map[string]*QueryRequest)

// NewCouchbaseDatasource creates a new datasource instance.
func NewCouchbaseDatasource(context context.Context, instance backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	var settings map[string]string
	if e := json.Unmarshal(instance.JSONData, &settings); e != nil {
		log.DefaultLogger.Error("Failed to parse couchbase datasource settings: %w", e)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "failed to parse settings",
		}, e
	}
	password := instance.DecryptedSecureJSONData["password"]
	log.DefaultLogger.Info("Connecting to cluster: '%s'", settings["host"])
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
		log.DefaultLogger.Info("Connected to couchbase cluster")
		return &CouchbaseDatasource{
			*cluster,
			instance,
		}, nil
	}
}

// CouchbaseDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type CouchbaseDatasource struct {
	Cluster  gocb.Cluster
	Instance backend.DataSourceInstanceSettings
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
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Panic occured: %v", err))
			debug.PrintStack()
			panic(err)
		}
	}()
	log.DefaultLogger.Info("QueryData called", "request", fmt.Sprintf("%+v", req))
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		log.DefaultLogger.Info("Processing query", "refId", q.RefID, "TYPE: "+q.QueryType)
		h := sha1.New()
		h.Write([]byte(q.JSON))
		query := parseQuery(q.JSON)
		query.Range = q.TimeRange
		//channel := "ds/" + d.Instance.UID + "/" + query.Key
		//channels[channel] = &query
		// channels are unstable
		if len(strings.TrimSpace(query.Query)) > 0 {
			response.Responses[q.RefID] = d.query(nil, &query)
		} else {
			log.DefaultLogger.Warn("Empty query with id '%s'", q.RefID)
		}
	}

	return response, nil
}

type QueryRequest struct {
	Query     string `json:"query"`
	Analytics bool   `json:"analytics"`
	Key       string `json:"key"`
	Range     backend.TimeRange
}

type cbResult interface {
	Next() bool
	Row(valuePtr interface{}) error
}

func parseQuery(raw []byte) QueryRequest {
	var query_data QueryRequest
	if err := json.Unmarshal(raw, &query_data); err != nil {
		log.DefaultLogger.Error("Failed to unmarshal request json", string(raw), "error", err)
		panic(err)
	} else {
		return query_data
	}
}

func parseJson(raw []byte) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		log.DefaultLogger.Error("Failed to unmarshal json", string(raw), "error", err)
		panic(err)
	} else {
		return result
	}
}

func (d *CouchbaseDatasource) query(channel *string, query_data *QueryRequest) backend.DataResponse {

	response := backend.DataResponse{
		Frames: make(data.Frames, 0),
	}

	query_string := query_data.Query
	tr := query_data.Range
	var timeField *string

	log.DefaultLogger.Info("Transforming query")
	if strTimeRg, e := regexp.Compile("(?i)str_time_range\\s*\\((?P<field>[^\\)]+)\\)"); e != nil {
		panic(e)
	} else {
		for _, match := range strTimeRg.FindAllStringSubmatch(query_string, -1) {
			if timeField != nil {
				response.Error = errors.New("Only one call to STR_TIME_RANGE per query is supported")
				return response
			}
			timeField = &match[strTimeRg.SubexpIndex("field")]
			query_string = strTimeRg.ReplaceAllString(query_string, fmt.Sprintf("STR_TO_MILLIS($1) > STR_TO_MILLIS('%s') AND STR_TO_MILLIS($1) <= STR_TO_MILLIS('%s')", tr.From.Format(time.RFC3339), tr.To.Format(time.RFC3339)))
			query_string = "SELECT * FROM (" + query_string + ") AS data ORDER by str_to_millis(data." + *timeField + ") ASC"
		}
	}

	if timeRg, e := regexp.Compile("(?i)time_range\\s*\\((?P<field>[^\\)]+)\\)"); e != nil {
		panic(e)
	} else {
		for _, match := range timeRg.FindAllStringSubmatch(query_string, -1) {
			if timeField != nil {
				response.Error = errors.New("Only one call to TIME_RANGE per query is supported")
				return response
			}
			timeField = &match[timeRg.SubexpIndex("field")]
			query_string = timeRg.ReplaceAllString(query_string, fmt.Sprintf("TO_NUMBER($1) > STR_TO_MILLIS('%s') AND TO_NUMBER($1) <= STR_TO_MILLIS('%s')", tr.From.Format(time.RFC3339), tr.To.Format(time.RFC3339)))
			query_string = "SELECT * FROM (" + query_string + ") AS data ORDER by TO_NUMBER(data." + *timeField + ") ASC"
		}
	}

	if timeField == nil {
		response.Error = errors.New("Failed to detect time field. Please use time_range(fieldName) or str_time_range(fieldName) functions in WHERE clause of your query.")
		return response
	}

	log.DefaultLogger.Info("Unmarshalled json", "query_string", query_string)

	log.DefaultLogger.Info("Querying couchbase", "query_string", query_string)
	var res cbResult
	var e error
	if query_data.Analytics {
		res, e = d.Cluster.AnalyticsQuery(query_string, nil)
	} else {
		res, e = d.Cluster.Query(query_string, nil)
	}

	if e != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Query failed: %+v", e))
		response.Error = e
	} else {
		log.DefaultLogger.Info("Query ok")

		var row map[string]interface{}
		frame := data.NewFrame("response")
		frame.SetMeta(&data.FrameMeta{
			ExecutedQueryString: query_string,
		})

		query_data.Range.To = query_data.Range.From // for streaming queries -- will force time period to be re-queried if no data fetched
		keys := []string{}
		var vals [][]interface{}
		for res.Next() {
			res.Row(&row)
			if row["data"] == nil {
				continue
			}

			d := row["data"].(map[string]interface{})

			if len(keys) == 0 {
				for key := range d {
					keys = append(keys, key)
					vals = append(vals, nil)
				}
			}

			for i, key := range keys {
				val := d[key]
				vals[i] = append(vals[i], val)
				if key == *timeField {
					if to, e := time.Parse(time.RFC3339, val.(string)); e == nil {
						query_data.Range.To = to
					}
				}
			}

		}

		frame.Fields = make(data.Fields, len(keys))
		for i, key := range keys {
			frame.Fields[i] = createField(key, vals[i])
		}

		if channel != nil {
			frame.Meta.Channel = *channel
		}

		response.Frames = append(response.Frames, frame)
	}

	return response
}

func normalizeFieldData(name string, values []interface{}) (string, []interface{}) {
	result := make([]interface{}, len(values))
	if strings.EqualFold(name, "time") {
		for i, v := range values {
			if time, err := time.Parse(time.RFC3339, v.(string)); err == nil {
				result[i] = time
			} else {
				panic(err)
			}
		}
		return "Time", result
	}
	return name, values
}

func createField(name string, values []interface{}) *data.Field {
	vlen := len(values)
	if vlen == 0 {
		return data.NewField(name, nil, []bool{})
	}

	name, values = normalizeFieldData(name, values)

	log.DefaultLogger.Debug(fmt.Sprintf("field %s: %d values", name, vlen))
	switch v := values[0].(type) {
	case int8:
		r := make([]int8, vlen)
		for i, v := range values {
			r[i] = v.(int8)
		}
		return data.NewField(name, nil, r)
	case *int8:
		r := make([]*int8, vlen)
		for i, v := range values {
			r[i] = v.(*int8)
		}
		return data.NewField(name, nil, r)
	case int16:
		r := make([]int16, vlen)
		for i, v := range values {
			r[i] = v.(int16)
		}
		return data.NewField(name, nil, r)
	case *int16:
		r := make([]*int16, vlen)
		for i, v := range values {
			r[i] = v.(*int16)
		}
		return data.NewField(name, nil, r)
	case int32:
		r := make([]int32, vlen)
		for i, v := range values {
			r[i] = v.(int32)
		}
		return data.NewField(name, nil, r)
	case *int32:
		r := make([]*int32, vlen)
		for i, v := range values {
			r[i] = v.(*int32)
		}
		return data.NewField(name, nil, r)
	case int64:
		r := make([]int64, vlen)
		for i, v := range values {
			r[i] = v.(int64)
		}
		return data.NewField(name, nil, r)
	case *int64:
		r := make([]*int64, vlen)
		for i, v := range values {
			r[i] = v.(*int64)
		}
		return data.NewField(name, nil, r)
	case uint8:
		r := make([]uint8, vlen)
		for i, v := range values {
			r[i] = v.(uint8)
		}
		return data.NewField(name, nil, r)
	case *uint8:
		r := make([]*uint8, vlen)
		for i, v := range values {
			r[i] = v.(*uint8)
		}
		return data.NewField(name, nil, r)
	case uint16:
		r := make([]uint16, vlen)
		for i, v := range values {
			r[i] = v.(uint16)
		}
		return data.NewField(name, nil, r)
	case *uint16:
		r := make([]*uint16, vlen)
		for i, v := range values {
			r[i] = v.(*uint16)
		}
		return data.NewField(name, nil, r)
	case uint32:
		r := make([]uint32, vlen)
		for i, v := range values {
			r[i] = v.(uint32)
		}
		return data.NewField(name, nil, r)
	case *uint32:
		r := make([]*uint32, vlen)
		for i, v := range values {
			r[i] = v.(*uint32)
		}
		return data.NewField(name, nil, r)
	case uint64:
		r := make([]uint64, vlen)
		for i, v := range values {
			r[i] = v.(uint64)
		}
		return data.NewField(name, nil, r)
	case *uint64:
		r := make([]*uint64, vlen)
		for i, v := range values {
			r[i] = v.(*uint64)
		}
		return data.NewField(name, nil, r)
	case float32:
		r := make([]float32, vlen)
		for i, v := range values {
			r[i] = v.(float32)
		}
		return data.NewField(name, nil, r)
	case *float32:
		r := make([]*float32, vlen)
		for i, v := range values {
			r[i] = v.(*float32)
		}
		return data.NewField(name, nil, r)
	case float64:
		r := make([]float64, vlen)
		for i, v := range values {
			r[i] = v.(float64)
		}
		return data.NewField(name, nil, r)
	case *float64:
		r := make([]*float64, vlen)
		for i, v := range values {
			r[i] = v.(*float64)
		}
		return data.NewField(name, nil, r)
	case string:
		r := make([]string, vlen)
		for i, v := range values {
			r[i] = v.(string)
		}
		return data.NewField(name, nil, r)
	case *string:
		r := make([]*string, vlen)
		for i, v := range values {
			r[i] = v.(*string)
		}
		return data.NewField(name, nil, r)
	case bool:
		r := make([]bool, vlen)
		for i, v := range values {
			r[i] = v.(bool)
		}
		return data.NewField(name, nil, r)
	case *bool:
		r := make([]*bool, vlen)
		for i, v := range values {
			r[i] = v.(*bool)
		}
		return data.NewField(name, nil, r)
	case time.Time:
		r := make([]time.Time, vlen)
		for i, v := range values {
			r[i] = v.(time.Time)
		}
		return data.NewField(name, nil, r)
	case *time.Time:
		r := make([]*time.Time, vlen)
		for i, v := range values {
			r[i] = v.(*time.Time)
		}
		return data.NewField(name, nil, r)
	case []interface{}:
		raw := make([]json.RawMessage, len(values))
		for i, item := range values {
			b, err := json.Marshal(item)
			if err != nil {
				panic(fmt.Errorf("failed to marshal providers: %w", err))
			}
			raw[i] = b
		}
		return data.NewField(name, nil, raw)
	default:
		panic(fmt.Errorf("unsupported type %T", v))
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *CouchbaseDatasource) CheckHealth(context context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if pings, be := d.Cluster.Ping(&gocb.PingOptions{
		ReportID:     "grafana-test",
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeQuery},
		Context:      context,
	}); be != nil {
		log.DefaultLogger.Error("Failed to ping the cluster", "error", be.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: be.Error(),
		}, be
	} else {
		for service, responses := range pings.Services {
			for _, response := range responses {
				if response.State != gocb.PingStateOk {
					return &backend.CheckHealthResult{
						Status:  backend.HealthStatusError,
						Message: fmt.Sprintf("Node %s service '%d' ping response was '%d', which is not ok. Error: %s", response.ID, service, response.State, response.Error),
					}, nil
				}
			}
		}
	}

	log.DefaultLogger.Info("Verified cluster connectivity")
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Ping OK",
	}, nil
}

// SubscribeStream is called when a client wants to connect to a stream. This callback
// allows sending the first message.
func (d *CouchbaseDatasource) SubscribeStream(c context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Info("SubscribeStream called", "request", req)

	status := backend.SubscribeStreamStatusOK

	return &backend.SubscribeStreamResponse{
		Status: status,
	}, nil
}

// RunStream is called once for any open channel.  Results are shared with everyone
// subscribed to the same channel.
func (d *CouchbaseDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Panic occured: %v", err))
			debug.PrintStack()
			panic(err)
		}
	}()
	log.DefaultLogger.Info("RunStream called", "request", req)

	channel := "ds/" + d.Instance.UID + "/" + req.Path
	if query_data, present := channels[channel]; !present {
		log.DefaultLogger.Error("Failed to restore stream query")
		panic(errors.New("Failed to restore query data for channel " + channel))
	} else {
		log.DefaultLogger.Info("restored query data", query_data, "for channel", channel)

		for {
			// Stream data frames periodically till stream closed by Grafana.  for {
			select {
			case <-ctx.Done():
				log.DefaultLogger.Info("Context done, finish streaming", "path", req.Path)
				delete(channels, channel)
				return nil
			case <-time.After(time.Second):

				// Send new data periodically.
				query_data.Range.From = query_data.Range.To
				query_data.Range.To = time.Now()

				resp := d.query(&channel, query_data)

				err := sender.SendFrame(resp.Frames[0], data.IncludeAll)
				if err != nil {
					log.DefaultLogger.Error("Error sending frame", "error", err)
					continue
				}
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
