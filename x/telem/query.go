package telem

import (
	"github.com/synnaxlabs/x/query"
)

const (
	rateOptKey   query.OptionKey = "telem.rate"
	timeRangeKey query.OptionKey = "telem.timeRange"
	densityKey   query.OptionKey = "telem.density"
	dataTypeKey  query.OptionKey = "telem.dataType"
)

func SetRate(q query.Query, dr Rate) { q.Set(rateOptKey, dr) }

func SetTimeRange(q query.Query, tr TimeRange) { q.Set(timeRangeKey, tr) }

func SetDensity(q query.Query, dt Density) { q.Set(densityKey, dt) }

func SetDataType(q query.Query, dt DataType) { q.Set(dataTypeKey, dt) }

func GetRate(q query.Query) (Rate, error) {
	if v, ok := q.Get(rateOptKey); ok {
		return v.(Rate), nil
	}
	return 0, InvalidRate
}

func GetTimeRange(q query.Query) (TimeRange, error) {
	if v, ok := q.Get(timeRangeKey); ok {
		return v.(TimeRange), nil
	}
	return TimeRangeZero, InvalidTimeRange
}

func GetDensity(q query.Query) (Density, error) {
	if v, ok := q.Get(densityKey); ok {
		return v.(Density), nil
	}
	return 0, InvalidDensity
}

func GetDataType(q query.Query) (DataType, error) {
	if v, ok := q.Get(dataTypeKey); ok {
		return v.(DataType), nil
	}
	return DataTypeUnknown, InvalidDataType
}
