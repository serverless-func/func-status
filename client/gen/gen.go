package gen

import (
	"encoding/json"
	"fmt"
	lo "github.com/samber/lo"
	"github.com/serverless-aliyun/func-status/client/core"
	"math"
	"os"
	"time"
)

// EndpointGen info
type EndpointGen struct {
	// Key of the endpoint. Auto generated.
	Key string `yaml:"key"`

	// Name of the endpoint. Can be anything.
	Name string `yaml:"name"`

	// URL to send the request to
	URL string `yaml:"url"`

	// SLA of all results
	SLA float32 `yaml:"sla"`

	// Results of health check
	Results []*core.Result `json:"results"`
}

// EndpointReport from results
type EndpointReport struct {
	// Key of the endpoint. Auto generated.
	Key string `json:"key"`

	// Name of the endpoint. Can be anything.
	Name string `json:"name"`

	// URL to send the request to
	URL string `json:"url"`

	// SLA of all results
	SLA float64 `json:"sla"`

	// Status of latest (nodata, success, failure, partial)
	Status string `json:"status"`

	// Reports of health check per day
	Reports []EndpointDayReport `json:"reports"`
}

// EndpointDayReport from day result
type EndpointDayReport struct {
	// Day of check heal
	Day string `json:"day"`

	// Status of result by day (nodata, success, failure, partial)
	Status string `json:"status"`

	// ConditionResults results of the Endpoint's conditions
	ConditionResults []*core.ConditionResult `json:"conditionResults"`

	// SLA of result by day
	SLA float64 `yaml:"sla"`
}

// Gen endpoints report by results
func Gen(endpoints []EndpointGen, maxDays int) {
	var reports []EndpointReport
	for _, endpoint := range endpoints {
		report := EndpointReport{
			Key:  endpoint.Key,
			Name: endpoint.Name,
			URL:  endpoint.URL,
		}
		// 按日分组
		dayGrouped := lo.GroupBy(endpoint.Results, func(item *core.Result) string {
			return item.Timestamp.Format("2006-01-02")
		})
		// 时间范围
		start := time.Now()
		end := start.AddDate(0, 0, -maxDays)
		// 汇总计算 sla
		var total float64
		// 计算每日状态
		var dayReports []EndpointDayReport
		for d := start; d.After(end); d = d.AddDate(0, 0, -1) {
			day := d.Format("2006-01-02")
			if results, ok := dayGrouped[day]; ok {
				dayReport := calcDaySLA(results)
				dayReport.Day = day
				total += dayReport.SLA
				dayReports = append(dayReports, dayReport)
				report.Status = latestStatus(results)
			} else {
				// 数据填充
				dayReports = append(dayReports, EndpointDayReport{
					Day:              day,
					Status:           "nodata",
					ConditionResults: nil,
					SLA:              0,
				})
			}
		}
		report.Reports = dayReports
		// 计算总SLA
		report.SLA = math.Round(total / float64(len(dayGrouped)))

		reports = append(reports, report)
	}

	rb, _ := json.Marshal(reports)
	genFile := "web/endpoints.js"
	_ = os.WriteFile(genFile, []byte(fmt.Sprintf("const endpoints = %s;", string(rb))), 0644)
}

// calcDaySLA 每日状态计算: 全部成功 success (sla: 100)/全部失败 failure (sla: 0)/部分成功失败 partial (sla: 失败 condition / condition 总数)
func calcDaySLA(results []*core.Result) EndpointDayReport {
	total := 0
	success := 0
	successConditions := make([]*core.ConditionResult, 0)
	failureConditions := make([]*core.ConditionResult, 0)
	for _, r := range results {
		for _, cr := range r.ConditionResults {
			if cr.Success {
				success += 1
			}
			total += 1
		}
		if r.Success {
			successConditions = r.ConditionResults
		} else {
			failureConditions = r.ConditionResults
		}
	}
	if success == 0 {
		return EndpointDayReport{
			Status:           "failure",
			ConditionResults: failureConditions,
			SLA:              0,
		}
	}
	if success == total {
		return EndpointDayReport{
			Status:           "success",
			ConditionResults: successConditions,
			SLA:              100,
		}
	}
	return EndpointDayReport{
		Status:           "partial",
		ConditionResults: failureConditions,
		SLA:              math.Round(float64(success) * 100 / float64(total)),
	}
}

func latestStatus(results []*core.Result) string {
	if len(results) == 1 {
		return "nodata"
	}
	last := results[len(results)-1]
	if last.Success {
		return "success"
	}
	partialSuccess := lo.Filter(last.ConditionResults, func(item *core.ConditionResult, index int) bool {
		return item.Success
	})
	if len(partialSuccess) == 0 {
		return "failure"
	}
	return "partial"
}
