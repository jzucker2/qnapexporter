package prometheus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mackerelio/go-osstat/loadavg"
	"github.com/mackerelio/go-osstat/uptime"
	"gitlab.com/pedropombeiro/qnapexporter/lib/utils"
)

func getUptimeMetrics() ([]metric, error) {
	u, err := uptime.Get()
	if err != nil {
		return nil, err
	}

	return []metric{
		{
			name:       "node_time_seconds",
			value:      u.Seconds(),
			help:       "System uptime measured in seconds",
			metricType: "counter",
		},
	}, err
}

func getLoadAvgMetrics() ([]metric, error) {
	s, err := loadavg.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{name: "node_load1", value: s.Loadavg1},
		{name: "node_load5", value: s.Loadavg5},
		{name: "node_load15", value: s.Loadavg15},
	}
	return metrics, nil
}

func (e *promExporter) getSysInfoTempMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, 2)

	for _, dev := range []string{"cputmp", "systmp"} {
		output, err := utils.ExecCommand(e.getsysinfo, dev)
		if err != nil {
			return nil, err
		}

		tokens := strings.SplitN(output, " ", 2)
		value, err := strconv.ParseFloat(tokens[0], 64)
		if err != nil {
			continue
		}

		metrics = append(metrics, metric{
			name:  fmt.Sprintf("node_%s_C", dev),
			value: value,
		})
	}

	return metrics, nil
}

func (e *promExporter) getSysInfoFanMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, e.sysfannum)

	for fannum := 1; fannum <= e.sysfannum; fannum++ {
		fannumStr := strconv.Itoa(fannum)

		fanStr, err := utils.ExecCommand(e.getsysinfo, "sysfan", fannumStr)
		if err != nil {
			return nil, err
		}

		fan, err := strconv.ParseFloat(strings.SplitN(fanStr, " ", 2)[0], 64)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric{
			name:  "node_sysfan_RPM",
			attr:  fmt.Sprintf(`fan=%q`, fannumStr),
			value: fan,
		})
	}

	return metrics, nil
}
