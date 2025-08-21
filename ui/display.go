package ui

import (
	"fmt"
	"sort"

	"github.com/sahaj-b/wakafetch/types"
)

type Field struct {
	Key string
	Val string
}

type DisplayPayload struct {
	Heading          string
	Stats            []Field
	Languages        []types.StatItem
	Editors          []types.StatItem
	Projects         []types.StatItem
	OperatingSystems []types.StatItem
	Categories       []types.StatItem
	Machines         []types.StatItem
	Entities         []types.StatItem
	Full             bool
}

func DisplayStats(data *types.StatsResponse, full bool, rangeStr string) {
	if data == nil || (data.Data.TotalSeconds == 0 && len(data.Data.Languages) == 0 && len(data.Data.Projects) == 0) {
		Warnln("No data available for the selected period: '%s'", rangeStr)
		return
	}

	stats := data.Data
	var heading string
	if rangeStr == "all_time" {
		heading = formatRangeHeading(rangeStr)
	} else {
		heading = formatRangeHeading(rangeStr) + " (" + formatDateRange(stats.Start, stats.End) + ")"
	}

	totalTime := timeFmt(stats.TotalSeconds)
	dailyAvg := timeFmt(stats.DailyAverage)

	topProject := topItemName(stats.Projects, true)
	topEditor := topItemName(stats.Editors, false)
	topOS := topItemName(stats.OperatingSystems, false)

	numLangs := fmt.Sprintf("%d", len(stats.Languages))
	numProjects := fmt.Sprintf("%d", len(stats.Projects))

	var statsMap []Field
	statsMap = []Field{
		{"Total Time", totalTime},
	}

	if stats.DaysIncludingHolidays > 1 {
		statsMap = append(statsMap,
			Field{"Daily Avg", dailyAvg},
		)
	}

	statsMap = append(statsMap,
		Field{"Top Project", topProject},
		Field{"Top Editor", topEditor},
		Field{"Top OS", topOS},
		Field{"Languages", numLangs},
		Field{"Projects", numProjects},
	)

	payload := DisplayPayload{
		Heading:          heading,
		Stats:            statsMap,
		Languages:        stats.Languages,
		Editors:          stats.Editors,
		Projects:         stats.Projects,
		OperatingSystems: stats.OperatingSystems,
		Categories:       stats.Categories,
		Machines:         stats.Machines,
		Entities:         nil, // stats response doesn't have entities
		Full:             full,
	}
	render(&payload)
}

type job struct {
	targetMap map[string]float64
	getter    func(types.DayData) []types.StatItem
}

func DisplaySummary(data *types.SummaryResponse, full bool, rangeStr string) {
	if data == nil || len(data.Data) == 0 {
		Warnln("No data available for the selected period: '%s'", rangeStr)
		return
	}

	languages := make(map[string]float64)

	// Only process additional data if full mode is on
	projects, editors, operatingSystems, categories, machines, entities := make(map[string]float64), make(map[string]float64), make(map[string]float64), make(map[string]float64), make(map[string]float64), make(map[string]float64)

	aggregateJobs := []job{
		{languages, func(day types.DayData) []types.StatItem { return day.Languages }},
		{projects, func(day types.DayData) []types.StatItem { return day.Projects }},
		{editors, func(day types.DayData) []types.StatItem { return day.Editors }},
		{operatingSystems, func(day types.DayData) []types.StatItem { return day.OperatingSystems }},
		{categories, func(day types.DayData) []types.StatItem { return day.Categories }},
		{machines, func(day types.DayData) []types.StatItem { return day.Machines }},
		{entities, func(day types.DayData) []types.StatItem { return day.Entities }},
	}

	processJobs(data.Data, aggregateJobs)

	// Find busiest day
	busiestDay := ""
	busiestDaySeconds := 0.0
	for _, dayData := range data.Data {
		if dayData.GrandTotal.TotalSeconds > busiestDaySeconds {
			busiestDaySeconds = dayData.GrandTotal.TotalSeconds
			busiestDay = dayData.Range.Date
		}
	}

	aggregatedLangs := mapToSortedStatItems(languages)
	aggregatedProjs := mapToSortedStatItems(projects)
	aggregatedEditors := mapToSortedStatItems(editors)
	aggregatedOS := mapToSortedStatItems(operatingSystems)
	aggregatedCategories := mapToSortedStatItems(categories)
	aggregatedMachines := mapToSortedStatItems(machines)

	heading := formatRangeHeading(rangeStr) + " (" + formatDateRange(data.Start, data.End) + ")"
	totalTime := timeFmt(data.CumulativeTotal.Seconds)
	dailyAvg := timeFmt(data.DailyAverage.Seconds)
	activeDays := fmt.Sprintf("%d/%d days", data.DailyAverage.DaysMinusHolidays, data.DailyAverage.DaysIncludingHolidays)

	topProject := topItemName(aggregatedProjs, true)
	topEditor := topItemName(aggregatedEditors, false)
	topOS := topItemName(aggregatedOS, false)
	// topCategory := topItemName(aggregatedCategories, false)

	numLangs := fmt.Sprintf("%d", len(aggregatedLangs))
	numProjects := fmt.Sprintf("%d", len(aggregatedProjs))

	var statsMap []Field
	statsMap = []Field{
		{"Total Time", totalTime},
	}

	if len(data.Data) > 1 {
		statsMap = append(statsMap,
			Field{"Daily Avg", dailyAvg},
			Field{"Active Days", activeDays},
			Field{"Best Day", fmt.Sprintf("%s (%s)", formatBestDay(busiestDay), timeFmt(busiestDaySeconds))},
		)
	}

	statsMap = append(statsMap,
		Field{"Top Project", topProject},
		Field{"Top Editor", topEditor},
		Field{"Top OS", topOS},
		Field{"Languages", numLangs},
		Field{"Projects", numProjects},
	)

	payload := &DisplayPayload{
		Heading:          heading,
		Stats:            statsMap,
		Languages:        aggregatedLangs,
		Editors:          aggregatedEditors,
		Projects:         aggregatedProjs,
		OperatingSystems: aggregatedOS,
		Full:             full,
		Categories:       aggregatedCategories,
		Machines:         aggregatedMachines,
	}
	render(payload)
}

func DisplayBreakdown(data []types.DayData, heading string) {
	if len(data) == 0 {
		Warnln("No daily data available")
		return
	}
	dailyTable, tableWidth := dailyBreakdownStr(data)
	cardTable, _ := cardify(dailyTable, heading, tableWidth, 0)
	printStrs(cardTable)
	if Clr.Blue == "" {
		return // dont display heatmap if colors disabled
	}
	heatmapStrs, heatmapWidth := heatmap(data)
	cardHeatmap, _ := cardify(heatmapStrs, "Heatmap", heatmapWidth, 0)
	printStrs(cardHeatmap)
}

func DisplayHeatmap(data []types.DayData, heading string) {
	if len(data) == 0 {
		Warnln("No daily data available")
		return
	}

	heatmapStrs, heatmapWidth := heatmap_custom(data)
	if len(heatmapStrs) == 0 {
		Warnln("No heatmap data available")
		return
	}

	cardHeatmap, _ := cardify(heatmapStrs, heading, heatmapWidth, 0)
	printStrs(cardHeatmap)
}

func processJobs(data []types.DayData, jobs []job) {
	for _, dayData := range data {
		for _, j := range jobs {
			items := j.getter(dayData)
			for _, item := range items {
				j.targetMap[item.Name] += item.TotalSeconds
			}
		}
	}
}

func mapToSortedStatItems(m map[string]float64) []types.StatItem {
	items := make([]types.StatItem, 0, len(m))
	for name, seconds := range m {
		items = append(items, types.StatItem{Name: name, TotalSeconds: seconds})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TotalSeconds > items[j].TotalSeconds
	})
	return items
}
