package main

import (
	_ "embed"
	"html/template"
	"math/rand"
	"net/http"
	"time"

	"github.com/crhntr/dailycountchart"
)

type Record struct {
	Index int
	Time  time.Time
}

func (r Record) Timestamp() time.Time {
	return r.Time
}

//go:embed index.gohtml
var indexPage string

func makeNRandomRecords(start time.Time, n int) []Record {
	records := make([]Record, 0, n)
	for i := 0; i < cap(records); i++ {
		d := rand.Intn(365 * 2)
		t := start.AddDate(0, 0, d)
		records = append(records, Record{
			Index: i,
			Time:  t,
		})
	}
	return records
}

func main() {
	templates := template.Must(template.New("").Parse(indexPage))
	rand.Seed(time.Now().Unix())
	_ = http.ListenAndServe(":8080", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now().AddDate(-2, 0, 0)

		charts, err := dailycountchart.New(makeNRandomRecords(start, 1000), nil)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
		_ = templates.Execute(res, charts)
	}))
}
