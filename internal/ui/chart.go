package ui

import (
	"fmt"
	"strings"
)

// RenderBarChart renders a horizontal bar chart with labels, bars, and formatted values.
func RenderBarChart(labels []string, values []float64, maxVal float64, width int, formatVal func(float64) string) string {
	if len(labels) == 0 {
		return ""
	}

	if maxVal <= 0 {
		for _, v := range values {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	maxLabelLen := 0
	for _, l := range labels {
		if len(l) > maxLabelLen {
			maxLabelLen = len(l)
		}
	}

	var sb strings.Builder
	for i, val := range values {
		label := labels[i]
		barLen := int((val / maxVal) * float64(width))
		if barLen < 0 {
			barLen = 0
		}
		if barLen > width {
			barLen = width
		}

		bar := strings.Repeat("█", barLen)
		empty := strings.Repeat("░", width-barLen)

		valStr := ""
		if formatVal != nil {
			valStr = formatVal(val)
		} else {
			valStr = fmt.Sprintf("%.1f", val)
		}

		paddedLabel := fmt.Sprintf("%*s", maxLabelLen, label)
		sb.WriteString(fmt.Sprintf("%s%s%s │ %s%s%s%s%s%s │ %s\n",
			Dim, paddedLabel, Reset,
			Green, bar, Reset,
			Gray, empty, Reset,
			valStr,
		))
	}

	return sb.String()
}
