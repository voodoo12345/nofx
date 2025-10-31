package market

import "math"

// BollingerData 布林带指标
type BollingerData struct {
	Middle    float64
	Upper     float64
	Lower     float64
	Bandwidth float64 // (上轨-下轨)/中轨，反映波动率，相对百分比
	ZScore    float64 // 当前价格相对中轨的标准差位置
}

// BollingerSeries 布林带序列（用于输出最近数据）
type BollingerSeries struct {
	Upper  []float64
	Middle []float64
	Lower  []float64
}

func calculateOBV(klines []Kline) []float64 {
	if len(klines) == 0 {
		return []float64{}
	}

	obv := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		obv[i] = obv[i-1]
		switch {
		case klines[i].Close > klines[i-1].Close:
			obv[i] += klines[i].Volume
		case klines[i].Close < klines[i-1].Close:
			obv[i] -= klines[i].Volume
		}
	}

	return obv
}

func calculateBollinger(klines []Kline, period int) (*BollingerSeries, *BollingerData) {
	if len(klines) == 0 {
		return &BollingerSeries{}, &BollingerData{}
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	series := &BollingerSeries{
		Upper:  make([]float64, len(klines)),
		Middle: make([]float64, len(klines)),
		Lower:  make([]float64, len(klines)),
	}

	var latest *BollingerData
	if len(klines) < period {
		return series, &BollingerData{}
	}

	for i := period - 1; i < len(closes); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		mean := sum / float64(period)

		varianceSum := 0.0
		for j := i - period + 1; j <= i; j++ {
			diff := closes[j] - mean
			varianceSum += diff * diff
		}

		std := math.Sqrt(varianceSum / float64(period))
		upper := mean + 2*std
		lower := mean - 2*std

		series.Middle[i] = mean
		series.Upper[i] = upper
		series.Lower[i] = lower

		if i == len(closes)-1 {
			bandwidth := 0.0
			if mean != 0 {
				bandwidth = (upper - lower) / mean
			}

			zScore := 0.0
			if std > 0 {
				zScore = (closes[i] - mean) / std
			}

			latest = &BollingerData{
				Middle:    mean,
				Upper:     upper,
				Lower:     lower,
				Bandwidth: bandwidth,
				ZScore:    zScore,
			}
		}
	}

	if latest == nil {
		latest = &BollingerData{}
	}

	return series, latest
}

func populateIntradayVolumeSignals(data *IntradayData, obvSeries []float64, bbSeries *BollingerSeries) {
	if data == nil || len(data.MidPrices) == 0 {
		return
	}

	targetLen := len(data.MidPrices)

	if data.OBVValues == nil {
		data.OBVValues = make([]float64, 0, targetLen)
	} else {
		data.OBVValues = data.OBVValues[:0]
	}

	if data.BollingerUpper == nil {
		data.BollingerUpper = make([]float64, 0, targetLen)
	} else {
		data.BollingerUpper = data.BollingerUpper[:0]
	}

	if data.BollingerMiddle == nil {
		data.BollingerMiddle = make([]float64, 0, targetLen)
	} else {
		data.BollingerMiddle = data.BollingerMiddle[:0]
	}

	if data.BollingerLower == nil {
		data.BollingerLower = make([]float64, 0, targetLen)
	} else {
		data.BollingerLower = data.BollingerLower[:0]
	}

	// Align the indicator windows with the intraday slice by taking the most recent values.
	if len(obvSeries) > 0 {
		start := len(obvSeries) - targetLen
		if start < 0 {
			start = 0
		}
		for i := start; i < len(obvSeries); i++ {
			data.OBVValues = append(data.OBVValues, obvSeries[i])
		}
	}

	if bbSeries == nil {
		return
	}

	// Bollinger calculations might not be available for the earliest candles; append zeros to retain alignment.
	appendRange := func(dst []float64, src []float64) []float64 {
		if len(src) == 0 {
			return dst
		}
		start := len(src) - targetLen
		if start < 0 {
			start = 0
		}
		for i := start; i < len(src); i++ {
			dst = append(dst, src[i])
		}
		return dst
	}

	data.BollingerUpper = appendRange(data.BollingerUpper, bbSeries.Upper)
	data.BollingerMiddle = appendRange(data.BollingerMiddle, bbSeries.Middle)
	data.BollingerLower = appendRange(data.BollingerLower, bbSeries.Lower)
}
