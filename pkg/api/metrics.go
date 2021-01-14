package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/grafana/grafana/pkg/expr"
	"github.com/grafana/grafana/pkg/models"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/tsdb"
	"github.com/grafana/grafana/pkg/tsdb/testdatasource"
	"github.com/grafana/grafana/pkg/util"
)

// QueryMetricsV2 returns query metrics.
// POST /api/ds/query   DataSource query w/ expressions
func (hs *HTTPServer) QueryMetricsV2(c *models.ReqContext, reqDTO dtos.MetricRequest) Response {
	if len(reqDTO.Queries) == 0 {
		return Error(400, "No queries found in query", nil)
	}

	start := time.Now()

	request := &tsdb.TsdbQuery{
		TimeRange: tsdb.NewTimeRange(reqDTO.From, reqDTO.To),
		Debug:     reqDTO.Debug,
		User:      c.SignedInUser,
	}

	hasExpr := false
	var ds *models.DataSource
	for i, query := range reqDTO.Queries {
		hs.log.Debug("Processing metrics query", "query", query)
		name := query.Get("datasource").MustString("")
		if name == expr.DatasourceName {
			hasExpr = true
		}

		datasourceID, err := query.Get("datasourceId").Int64()
		if err != nil {
			hs.log.Debug("Can't process query since it's missing data source ID")
			return Error(400, "Query missing data source ID", nil)
		}

		if i == 0 && !hasExpr {
			ds, err = hs.DatasourceCache.GetDatasource(datasourceID, c.SignedInUser, c.SkipCache)
			if err != nil {
				hs.log.Debug("Encountered error getting data source", "err", err, "id", datasourceID)
				if errors.Is(err, models.ErrDataSourceAccessDenied) {
					return Error(403, "Access denied to data source", err)
				}
				if errors.Is(err, models.ErrDataSourceNotFound) {
					return Error(400, "Invalid data source ID", err)
				}
				return Error(500, "Unable to load data source metadata", err)
			}
		}

		request.Queries = append(request.Queries, &tsdb.Query{
			RefId:         query.Get("refId").MustString("A"),
			MaxDataPoints: query.Get("maxDataPoints").MustInt64(100),
			IntervalMs:    query.Get("intervalMs").MustInt64(1000),
			QueryType:     query.Get("queryType").MustString(""),
			Model:         query,
			DataSource:    ds,
		})
	}
	spent := time.Since(start)
	fmt.Printf("\nTime spent pre-processing queries: %d\n\n", spent.Milliseconds())
	start = time.Now()

	var resp *tsdb.Response
	var err error
	if !hasExpr {
		resp, err = tsdb.HandleRequest(c.Req.Context(), ds, request)
		if err != nil {
			return Error(500, "Metric request error", err)
		}
		spent := time.Since(start)
		fmt.Printf("\nTime spent handling request: %d\n\n", spent.Milliseconds())
	} else {
		if !hs.Cfg.IsExpressionsEnabled() {
			return Error(404, "Expressions feature toggle is not enabled", nil)
		}

		resp, err = expr.WrapTransformData(c.Req.Context(), request)
		if err != nil {
			return Error(500, "Transform request error", err)
		}
	}

	statusCode := 200
	for _, res := range resp.Results {
		if res.Error != nil {
			res.ErrorString = res.Error.Error()
			resp.Message = res.ErrorString
			statusCode = 400
		}
	}

	return jsonStreaming(statusCode, resp)
}

// QueryMetrics returns query metrics
// POST /api/tsdb/query
func (hs *HTTPServer) QueryMetrics(c *models.ReqContext, reqDto dtos.MetricRequest) Response {
	timeRange := tsdb.NewTimeRange(reqDto.From, reqDto.To)

	if len(reqDto.Queries) == 0 {
		return Error(400, "No queries found in query", nil)
	}

	datasourceId, err := reqDto.Queries[0].Get("datasourceId").Int64()
	if err != nil {
		return Error(400, "Query missing datasourceId", nil)
	}

	ds, err := hs.DatasourceCache.GetDatasource(datasourceId, c.SignedInUser, c.SkipCache)
	if err != nil {
		if errors.Is(err, models.ErrDataSourceAccessDenied) {
			return Error(403, "Access denied to datasource", err)
		}
		return Error(500, "Unable to load datasource meta data", err)
	}

	request := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Debug:     reqDto.Debug,
		User:      c.SignedInUser,
	}

	for _, query := range reqDto.Queries {
		request.Queries = append(request.Queries, &tsdb.Query{
			RefId:         query.Get("refId").MustString("A"),
			MaxDataPoints: query.Get("maxDataPoints").MustInt64(100),
			IntervalMs:    query.Get("intervalMs").MustInt64(1000),
			Model:         query,
			DataSource:    ds,
		})
	}

	resp, err := tsdb.HandleRequest(c.Req.Context(), ds, request)
	if err != nil {
		return Error(500, "Metric request error", err)
	}

	statusCode := 200
	for _, res := range resp.Results {
		if res.Error != nil {
			res.ErrorString = res.Error.Error()
			resp.Message = res.ErrorString
			statusCode = 400
		}
	}

	return JSON(statusCode, &resp)
}

// GET /api/tsdb/testdata/scenarios
func GetTestDataScenarios(c *models.ReqContext) Response {
	result := make([]interface{}, 0)

	scenarioIds := make([]string, 0)
	for id := range testdatasource.ScenarioRegistry {
		scenarioIds = append(scenarioIds, id)
	}
	sort.Strings(scenarioIds)

	for _, scenarioId := range scenarioIds {
		scenario := testdatasource.ScenarioRegistry[scenarioId]
		result = append(result, map[string]interface{}{
			"id":          scenario.Id,
			"name":        scenario.Name,
			"description": scenario.Description,
			"stringInput": scenario.StringInput,
		})
	}

	return JSON(200, &result)
}

// GenerateError generates a index out of range error
func GenerateError(c *models.ReqContext) Response {
	var array []string
	// nolint: govet
	return JSON(200, array[20])
}

// GET /api/tsdb/testdata/gensql
func GenerateSQLTestData(c *models.ReqContext) Response {
	if err := bus.Dispatch(&models.InsertSQLTestDataCommand{}); err != nil {
		return Error(500, "Failed to insert test data", err)
	}

	return JSON(200, &util.DynMap{"message": "OK"})
}

// GET /api/tsdb/testdata/random-walk
func GetTestDataRandomWalk(c *models.ReqContext) Response {
	from := c.Query("from")
	to := c.Query("to")
	intervalMs := c.QueryInt64("intervalMs")

	timeRange := tsdb.NewTimeRange(from, to)
	request := &tsdb.TsdbQuery{TimeRange: timeRange}

	dsInfo := &models.DataSource{Type: "testdata"}
	request.Queries = append(request.Queries, &tsdb.Query{
		RefId:      "A",
		IntervalMs: intervalMs,
		Model: simplejson.NewFromAny(&util.DynMap{
			"scenario": "random_walk",
		}),
		DataSource: dsInfo,
	})

	resp, err := tsdb.HandleRequest(context.Background(), dsInfo, request)
	if err != nil {
		return Error(500, "Metric request error", err)
	}

	return JSON(200, &resp)
}
