package result

import (
	"github.com/samber/lo"
	"github.com/serverless-aliyun/func-status/client/core"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

type StatusResult struct {
	gorm.Model
	// Key of Endpoint
	Key string
	// HTTPStatus is the HTTP response status code
	HTTPStatus int
	// Hostname extracted from Endpoint.URL
	Hostname string
	// IP resolved from the Endpoint URL
	IP string
	// Connected whether a connection to the host was established successfully
	Connected bool
	// Duration time that the request took
	Duration time.Duration
	// Errors encountered during the evaluation of the Endpoint's health
	// Errors []string
	// ConditionResults results of the Endpoint's conditions
	ConditionResults datatypes.JSONSlice[*core.ConditionResult]
	// Success whether the result signifies a success or not
	Success bool
	// CertificateExpiration is the duration before the certificate expires
	CertificateExpiration time.Duration `json:"-"`
	// Version of Current Release | Package
	Version string
}

var conn *gorm.DB

func ConnectToDB(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&StatusResult{})
	if err != nil {
		return err
	}
	conn = db
	return err
}

func SaveToDB(endpoint *core.Endpoint, result *core.Result, maxDays int) []*core.Result {
	endpoint.Key()
	dbResult := &StatusResult{
		Key:                   endpoint.Key(),
		HTTPStatus:            result.HTTPStatus,
		Hostname:              result.Hostname,
		IP:                    result.IP,
		Connected:             result.Connected,
		Duration:              result.Duration,
		ConditionResults:      datatypes.NewJSONSlice(result.ConditionResults),
		Success:               result.Success,
		CertificateExpiration: result.CertificateExpiration,
		Version:               result.Version,
	}
	// 写入
	conn.Create(&dbResult)
	// 删除
	deleteDate := time.Now().AddDate(0, 0, -maxDays)
	conn.Where("key = ? AND created_at < ?", endpoint.Key(), deleteDate).Delete(&StatusResult{})
	// 查询
	var results []StatusResult
	conn.Where("key = ?", endpoint.Key()).Find(&results)

	return lo.Map(results, func(item StatusResult, index int) *core.Result {
		return &core.Result{
			HTTPStatus:            item.HTTPStatus,
			Hostname:              item.Hostname,
			IP:                    item.IP,
			Connected:             item.Connected,
			Duration:              item.Duration,
			ConditionResults:      item.ConditionResults,
			Success:               item.Success,
			CertificateExpiration: item.CertificateExpiration,
			Version:               item.Version,
		}
	})
}
