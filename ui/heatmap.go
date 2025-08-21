package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/sahaj-b/wakafetch/types"
)

func heatmap(days []types.DayData) ([]string, int) {
	const heatmapChar = "■"               // █ ❐ ▪ ◼ 🙩 🙫 ⛝ ⏹ 🞕 🞔 🞖
	const highlight = "\x1b[38;2;0;%v;0m" // \x1b[38;2;R;G;Bm
	if len(days) == 0 {
		return []string{}, 0
	}
	maxSecs := 0
	startDay, err := time.Parse("2006-01-02", strings.Split(days[0].Range.Start, "T")[0])
	if err != nil {
		return []string{}, 0
	}
	endDay, err := time.Parse("2006-01-02", strings.Split(days[len(days)-1].Range.Start, "T")[0])
	if err != nil {
		return []string{}, 0
	}

	height := 4
	numOfDays := int(endDay.Sub(startDay).Hours()/24) + 1
	cols := getTerminalCols()
	width := 2*int((numOfDays+height-1)/height) - 1 // 2*ciel -1
	for width+4 > cols {
		height++
		width = 2*int((numOfDays+height-1)/height) - 1
	}

	for _, day := range days {
		if int(day.GrandTotal.TotalSeconds) > maxSecs {
			maxSecs = int(day.GrandTotal.TotalSeconds)
		}
	}
	output := make([]string, height)
	dataIndex := 0
	i := 0
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		strength255 := 0
		if dataIndex < len(days) && strings.Split(days[dataIndex].Range.Start, "T")[0] == d.Format("2006-01-02") {
			day := days[dataIndex]
			dataIndex++
			strength255 = int(255 * day.GrandTotal.TotalSeconds / float64(maxSecs))
		}
		char := fmt.Sprintf(highlight, strength255) + heatmapChar + "\x1b[0m"
		output[i%height] += char + " "
		i++
	}
	// trim trailing spaces
	for j := range output {
		output[j] = strings.TrimRight(output[j], " ")
	}

	// ensure same width for all lines
	for i%height != 0 {
		if i/height < 1 {
			output[i%height] += " "
		} else {
			output[i%height] += "  "
		}
		i++
	}
	return output, width
}

func heatmap_custom(days []types.DayData) ([]string, int) {
	const heatmapChar = "■"
	const highlight = "\x1b[38;2;0;%v;0m"

	if len(days) == 0 {
		return []string{}, 0
	}

	startDay, err := time.Parse("2006-01-02", days[0].Range.Date)
	if err != nil {
		return []string{}, 0
	}
	endDay, err := time.Parse("2006-01-02", days[len(days)-1].Range.Date)
	if err != nil {
		return []string{}, 0
	}

	height := 4
	numOfDays := int(endDay.Sub(startDay).Hours()/24) + 1
	cols := getTerminalCols()
	width := 2*int((numOfDays+height-1)/height) - 1
	for width+4 > cols {
		height++
		width = 2*int((numOfDays+height-1)/height) - 1
	}

	maxSecs := 0
	for _, day := range days {
		if int(day.GrandTotal.TotalSeconds) > maxSecs {
			maxSecs = int(day.GrandTotal.TotalSeconds)
		}
	}

	output := make([]string, height)
	dataIndex := 0
	i := 0

	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		strength255 := 0
		if dataIndex < len(days) && days[dataIndex].Range.Date == d.Format("2006-01-02") {
			day := days[dataIndex]
			dataIndex++
			strength255 = int(255 * float64(day.GrandTotal.TotalSeconds) / float64(maxSecs))
		}
		char := fmt.Sprintf(highlight, strength255) + heatmapChar + "\x1b[0m"
		output[i%height] += char + " "
		i++
	}

	for j := range output {
		output[j] = strings.TrimRight(output[j], " ")
	}

	for i%height != 0 {
		if i/height < 1 {
			output[i%height] += " "
		} else {
			output[i%height] += "  "
		}
		i++
	}
	return output, width
}
