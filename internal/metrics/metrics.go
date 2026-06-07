package metrics

import "github.com/prometheus/client_golang/prometheus"

var ReadingsTotal *prometheus.CounterVec

var ReadingsLatency prometheus.Histogram

var WSClientsConnected prometheus.Gauge

var NATSPublishErrors prometheus.Counter

func Register() error {
	ReadingsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "warden_gateway_readings_total",
		Help: "Total sensor readings processed",
	}, []string{"sensor_type", "room"})

	err := prometheus.Register(ReadingsTotal)
	if err != nil {
		return err
	}

	ReadingsLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "warden_gateway_readings_latency_seconds",
		Help:    "Latency of sensor readings",
		Buckets: prometheus.DefBuckets,
	})
	err = prometheus.Register(ReadingsLatency)
	if err != nil {
		return err
	}

	WSClientsConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "warden_gateway_ws_clients_connected",
		Help: "Number of active websocket clients connected",
	})
	err = prometheus.Register(WSClientsConnected)
	if err != nil {
		return err
	}
	NATSPublishErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "warden_gateway_nats_publish_errors_total",
		Help: "Number of errors publishing to NATS",
	})
	err = prometheus.Register(NATSPublishErrors)
	if err != nil {
		return err
	}
	return nil
}
