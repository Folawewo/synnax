package alamos

import (
	"bytes"
	"encoding/json"
	"io"
)

type Reporter interface {
	Report() Report
}

type Report map[string]interface{}

func AttachReporter(exp Experiment, key string, level Level, report Reporter) {
	if exp == nil {
		return
	}
	exp.attachReporter(key, level, report)
}

// JSON writes the report as JSON as bytes.
func (r Report) JSON() ([]byte, error) {
	b := bytes.NewBuffer([]byte{})
	err := r.WriteJSON(b)
	return b.Bytes(), err
}

// WriteJSON writes the report as JSON to the given writer.
func (r Report) WriteJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "")
	return e.Encode(r)
}

func (r Report) String() string {
	b, err := r.JSON()
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (r Report) LogArgs() []interface{} {
	args := make([]interface{}, 0, len(r))
	for k, v := range r {
		args = append(args, k, v)
	}
	return args
}

// Report implements Experiment.
func (e *experiment) Report() Report {
	report := make(map[string]interface{})
	for k, v := range e.measurements {
		report[k] = v.Report()
	}
	for k, v := range e.children {
		report[k] = v.Report()
	}
	for k, v := range e.reporters {
		report[k] = v.Report()
	}
	return report
}

func (e *experiment) attachReporter(key string, level Level, r Reporter) {
	if e.filterTest(level) {
		e.reporters[key] = r
	}
}
