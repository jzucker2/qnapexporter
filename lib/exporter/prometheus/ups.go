package prometheus

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"

	nut "github.com/robbiet480/go.nut"
)

type upsState struct {
	upsLock   sync.Mutex
	upsClient nut.Client

	upsConnErr      error
	upsConnAttempts int
	upsList         *[]nut.UPS
}

func (e *promExporter) getUpsStatsMetrics() (metrics []metric, err error) {
	e.upsState.upsLock.Lock()
	defer e.upsState.upsLock.Unlock()

	defer func() {
		if err != nil {
			var syscallErr *os.SyscallError
			if errors.As(err, &syscallErr) && syscallErr.Err == syscall.ECONNRESET {
				_, _ = e.upsState.upsClient.Disconnect()
			}
		}
	}()

	if e.upsState.upsClient.ProtocolVersion == "" {
		if e.upsState.upsConnAttempts < 10 {
			e.logger.Println("Connecting to UPS daemon")

			e.upsState.upsConnAttempts++
			e.upsState.upsClient, e.upsState.upsConnErr = nut.Connect("127.0.0.1")
		}
		if e.upsState.upsConnErr != nil {
			return nil, fmt.Errorf("%w (attempt %d)", e.upsState.upsConnErr, e.upsState.upsConnAttempts)
		}
	}

	if e.upsState.upsList == nil {
		upsList, err := e.upsState.upsClient.GetUPSList()
		if err != nil {
			return nil, err
		}
		e.upsState.upsList = &upsList
	}

	if len(*e.upsState.upsList) == 0 {
		return nil, nil
	}

	for _, ups := range *e.upsState.upsList {
		vars, err := ups.GetVariables()
		if err != nil {
			return nil, err
		}

		if metrics == nil {
			metrics = make([]metric, 0, len(vars)*len(*e.upsState.upsList)+1)
		}

		attr := fmt.Sprintf("ups=%q", ups.Name)

		var status, statusHelp, firmware string
		for _, v := range vars {
			switch v.Name {
			case "ups.status":
				status = v.Value.(string)
				statusHelp = v.Description
				continue
			case "ups.firmware":
				firmware = v.Value.(string)
				continue
			}

			var value float64
			switch v.Type {
			case "INTEGER":
				value = float64(v.Value.(int64))
			case "FLOAT_64":
				value = v.Value.(float64)
			default:
				continue
			}

			metrics = append(metrics, metric{
				name:  "ups_" + strings.ReplaceAll(v.Name, ".", "_"),
				attr:  attr,
				value: value,
				help:  v.Description,
			})
		}
		metrics = append(metrics, metric{
			name:  "ups_ups_status",
			attr:  fmt.Sprintf(`status=%q,firmware=%q,%s`, status, firmware, attr),
			value: getUpsStatus(status),
			help:  statusHelp,
		})
	}

	return metrics, nil
}

func getUpsStatus(status string) float64 {
	switch status {
	case "OL":
		return 0
	case "OL CHRG":
		return 1
	case "OB", "LB", "HB", "DISCHRG":
		return 2
	case "OFF":
		return 3
	case "RB":
		return 999
	default:
		return 99
	}
}
