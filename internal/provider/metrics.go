package provider

// Tags describe metric dimensions attached to a measurement.
type Tags map[string]string

// Counter records additive totals.
type Counter interface {
	Add(float64)
}

// Histogram records sampled floating point values.
type Histogram interface {
	Record(float64)
}

// Gauge records the latest instantaneous floating point value.
type Gauge interface {
	Set(float64)
}

// MetricsProvider abstracts application metrics collection.
type MetricsProvider interface {
	Counter(name string, tags Tags) Counter
	Histogram(name string, tags Tags) Histogram
	Gauge(name string, tags Tags) Gauge
}

// NewNoopMetricsProvider returns a provider that drops all measurements.
func NewNoopMetricsProvider() MetricsProvider {
	return noopMetricsProvider{}
}

type noopMetricsProvider struct{}

func (noopMetricsProvider) Counter(string, Tags) Counter {
	return noopCounter{}
}

func (noopMetricsProvider) Histogram(string, Tags) Histogram {
	return noopHistogram{}
}

func (noopMetricsProvider) Gauge(string, Tags) Gauge {
	return noopGauge{}
}

type noopCounter struct{}

func (noopCounter) Add(float64) {}

type noopHistogram struct{}

func (noopHistogram) Record(float64) {}

type noopGauge struct{}

func (noopGauge) Set(float64) {}
