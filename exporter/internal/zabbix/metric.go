package zabbix

import (
	"fmt"
	"strconv"
	"time"
)

type Value interface {
	string | int64 | uint64 | float64 | bool
}

type Metric struct {
	Host      string
	Key       string
	Value     any
	Timestamp time.Time
}

func FormatValue(value any) (string, error) {
	switch value := value.(type) {
	case string:
		return value, nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case uint64:
		return strconv.FormatUint(value, 10), nil
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64), nil
	case bool:
		return strconv.FormatBool(value), nil
	default:
		return "", fmt.Errorf("неподдерживаемый тип значения Zabbix %T", value)
	}
}
