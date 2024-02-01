package dailycountchart

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"sort"
	"time"
)

const (
	defaultHue = 127
)

type Configuration[E any] struct {
	EmptyDayColor template.CSS
	ColorFunc     func(min, max, n int) template.CSS

	ChartHeadingTitle  func(year int) string
	DataValueAttribute func(day Day[E]) string
	TitleAttribute     func(day Day[E]) string
	Time               func(E) time.Time
}

var (
	//go:embed *.gohtml
	templatesFS embed.FS

	templates = template.Must(template.ParseFS(templatesFS, "*.gohtml"))
)

type Chart struct {
	Year int
	HTML template.HTML
}

func New[E any, List ~[]E](configuration *Configuration[E], elements List, elementTime func(E) time.Time) ([]Chart, error) {
	var (
		result []Chart
		buf    bytes.Buffer
	)
	if configuration == nil {
		configuration = &Configuration[E]{}
	}
	if configuration.ColorFunc == nil {
		configuration.ColorFunc = ColorFunc(defaultHue)
	}
	if configuration.EmptyDayColor == "" {
		configuration.EmptyDayColor = "#EAEAEA"
	}
	for _, year := range years(elements, elementTime) {
		buf.Reset()
		days, count := newYear(year, elements, configuration, elementTime)
		err := templates.ExecuteTemplate(&buf, "daily-count-chart", struct {
			Year int
			*Configuration[E]
			Days  []Day[E]
			Total int
		}{
			Year:          year,
			Configuration: configuration,
			Days:          days,
			Total:         count,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, Chart{
			Year: year,
			HTML: template.HTML(buf.String()),
		})
	}
	return result, nil
}

func newYear[E any](year int, elements []E, configuration *Configuration[E], elementTime func(E) time.Time) ([]Day[E], int) {
	days := make([]Day[E], 0, 366)

	janFirst := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	lastTime := lastTimeInYear(year, elements, elementTime)

	wn := 1
	for t := janFirst; t.Year() < year+1 && !t.After(lastTime); t = t.AddDate(0, 0, 1) {
		days = append(days, Day[E]{
			time:          t,
			weekNumber:    wn,
			configuration: configuration,
		})
		if t.Weekday() == time.Saturday {
			wn++
		}
	}
	count := 0
	for _, el := range elements {
		t := elementTime(el)
		if t.Year() != year {
			continue
		}
		count++
		days[t.YearDay()-1].elements = append(days[t.YearDay()-1].elements, el)
	}
	setColors(days, configuration)

	return days, count
}

type Day[E any] struct {
	configuration *Configuration[E]
	time          time.Time
	elements      []E
	weekNumber    int
	color         template.CSS
}

func (day Day[E]) GridColumn() int {
	return day.weekNumber
}

func (day Day[E]) GridRow() int {
	return int(day.time.Weekday()) + 1
}

func (day Day[E]) Color() template.CSS {
	return day.color
}

func (day Day[E]) Timestamp() time.Time {
	return day.time
}

func (day Day[E]) Elements() []E {
	return day.elements
}

func (day Day[E]) DataValueAttribute() string {
	if day.configuration.DataValueAttribute == nil {
		return fmt.Sprintf("%d", len(day.elements))
	}
	return day.configuration.DataValueAttribute(day)
}

func (day Day[E]) TitleAttribute() string {
	if day.configuration.TitleAttribute == nil {
		return fmt.Sprintf("%s [%d]", day.time.Format("2006-01-02"), len(day.elements))
	}
	return day.configuration.TitleAttribute(day)
}

func setColors[E any](days []Day[E], configuration *Configuration[E]) {
	minReleases := math.MaxInt
	maxReleases := 0
	for _, day := range days {
		if len(day.elements) < minReleases {
			minReleases = len(day.elements)
		}
		if len(day.elements) > maxReleases {
			maxReleases = len(day.elements)
		}
	}
	color := configuration.ColorFunc
	if color == nil {
		color = ColorFunc(defaultHue)
	}
	for i, day := range days {
		if len(day.elements) == 0 {
			days[i].color = configuration.EmptyDayColor
			continue
		}
		days[i].color = color(minReleases, maxReleases, len(day.elements))
	}
}

func MapToRange(initialLow, initialHigh, finalLow, finalHigh, n float64) float64 {
	return finalLow + (finalHigh-finalLow)/(initialHigh-initialLow)*(n-initialLow)
}

func ColorFunc(hue int8) func(min, max, n int) template.CSS {
	return func(min, max, n int) template.CSS {
		l := MapToRange(float64(min), float64(max), 80, 20, float64(n))
		return template.CSS(fmt.Sprintf(`hsl(%d, 50%%, %.4f%%)`, hue, l))
	}
}

func years[E any](elements []E, elementTime func(E) time.Time) []int {
	set := make(map[int]struct{})
	for _, e := range elements {
		set[elementTime(e).Year()] = struct{}{}
	}
	result := make([]int, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	sort.Ints(result)
	return result
}

func lastTimeInYear[E any](year int, elements []E, elementTime func(E) time.Time) time.Time {
	var t time.Time
	for _, e := range elements {
		if et := elementTime(e); et.Year() == year && (t.IsZero() || et.After(t)) {
			t = elementTime(e)
		}
	}
	return t
}
