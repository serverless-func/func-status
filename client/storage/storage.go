package storage

import (
	"github.com/samber/lo"
	"github.com/serverless-aliyun/func-status/client/core"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"math"
	"time"
)

// Endpoint from results
type Endpoint struct {
	gorm.Model

	// Key of the endpoint. Auto generated.
	Key string `gorm:"column:key;uniqueIndex:uidx_key"`

	// Name of the endpoint. Can be anything.
	Name string `gorm:"column:name"`

	// URL to send the request to
	URL string `gorm:"column:url"`

	// Description of the endpoint
	Desc string `gorm:"column:desc"`

	// SLA of all results
	SLA float64 `gorm:"column:sla"`

	// Status of latest (nodata, success, failure, partial)
	Status string `gorm:"column:status"`
}

// Result from day result
type Result struct {
	gorm.Model

	// Key of the endpoint. Reference of the Endpoint.
	Key string `gorm:"column:key;uniqueIndex:uidx_key_day"`

	// Day of check health
	Day string `gorm:"column:day;uniqueIndex:uidx_key_day"`

	// SLA of result by day
	SLA float64 `gorm:"column:sla"`

	// Status of result by day (nodata, success, failure, partial)
	Status string `gorm:"column:status"`

	// Logs of the Endpoint's conditions
	Logs datatypes.JSONSlice[ConditionLog] `gorm:"column:logs"`
}

type ConditionLog struct {
	// Time of check health
	Time string `json:"time"`

	// Conditions result of the Endpoint's conditions
	Conditions []ConditionResult `json:"conditions"`
}

type ConditionResult struct {
	// Condition that was evaluated
	Condition string `json:"condition"`

	// Success whether the condition was met (successful) or not (failed)
	Success bool `json:"success"`
}

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusNoData  = "nodata"
	StatusPartial = "partial"
)

var conn *gorm.DB

func (Endpoint) TableName() string {
	return "endpoint"
}

func (Result) TableName() string {
	return "endpoint_result"
}

func ConnectToDB(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&Endpoint{}, &Result{})
	if err != nil {
		return err
	}
	conn = db
	return err
}

func SaveResult(key string, result *core.Result, maxDays int) {
	// 删除历史数据
	deleteDate := time.Now().AddDate(0, 0, -maxDays)
	conn.Where("key = ? AND created_at < ?", key, deleteDate).Delete(&Result{})
	// 查询当天数据
	day := time.Now().Format("2006-01-02")
	dayResult := &Result{
		Key:    key,
		Day:    day,
		SLA:    0,
		Status: StatusNoData,
		Logs:   nil,
	}
	conn.Where(&Result{Key: key, Day: day}).First(dayResult)
	// 更新当天数据
	nowResult := ConditionLog{
		Time: time.Now().Format("15:04:05"),
		Conditions: lo.Map(result.ConditionResults, func(item *core.ConditionResult, index int) ConditionResult {
			return ConditionResult{
				Condition: item.Condition,
				Success:   item.Success,
			}
		}),
	}
	dayResult.Logs = append([]ConditionLog{nowResult}, dayResult.Logs...)
	dayResult.Logs = dayResult.Logs[:int(math.Min(10, float64(len(dayResult.Logs))))]
	// 计算SLA
	status, sla := calcDaySLA(dayResult.Logs)
	dayResult.Status = status
	dayResult.SLA = sla
	conn.Save(dayResult)
}

func SaveEndpoint(e *core.Endpoint) {
	endpoint := &Endpoint{
		Key:    e.Key(),
		Name:   e.Name,
		URL:    e.URL,
		Status: StatusNoData,
		SLA:    0,
	}
	if e.Version != "" {
		endpoint.Desc = "Running Version: " + e.Version
	}
	conn.Where(&Endpoint{Key: e.Key()}).First(endpoint)
	var results []Result
	conn.Where(&Result{Key: e.Key()}).Find(&results)
	status, sla := calcEndpointSLA(results)
	endpoint.Status = status
	endpoint.SLA = sla
	conn.Save(endpoint)
}

// calcDaySLA 每日状态计算: 全部成功 success (sla: 100)/全部失败 failure (sla: 0)/部分成功失败 partial (sla: 失败 condition / condition 总数)
func calcDaySLA(logs datatypes.JSONSlice[ConditionLog]) (status string, sla float64) {
	total := 0
	success := 0
	for _, r := range logs {
		for _, cr := range r.Conditions {
			if cr.Success {
				success += 1
			}
			total += 1
		}
	}
	if success == 0 {
		status = StatusFailure
		sla = 0
	} else if success == total {
		status = StatusSuccess
		sla = 100
	} else {
		status = StatusPartial
		sla = math.Round(float64(success) * 100 / float64(total))
	}
	return status, sla
}

func calcEndpointSLA(results []Result) (status string, sla float64) {
	total := 0
	success := 0
	for _, r := range results {
		for _, l := range r.Logs {
			for _, cr := range l.Conditions {
				if cr.Success {
					success += 1
				}
				total += 1
			}
		}
	}
	if success == 0 {
		status = StatusFailure
		sla = 0
	} else if success == total {
		status = StatusSuccess
		sla = 100
	} else {
		status = StatusPartial
		sla = math.Round(float64(success) * 100 / float64(total))
	}
	return status, sla
}
