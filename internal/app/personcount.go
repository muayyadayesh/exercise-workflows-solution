package app

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// YOLO11 output: 84 values per box (cx, cy, w, h + 80 class scores), person is class 0.
const yoloValuesPerBox = 84
const personScoreIndex = 4

// IoU above this counts as the same person.
const overlapThreshold = 0.45

type personBox struct {
	x1, y1, x2, y2 float32
	score          float32
}

func countPersons(output []float32, rows, columns int, threshold float32) (int, float32, error) {
	boxes, best, err := personBoxes(output, rows, columns, threshold)
	if err != nil {
		return 0, 0, err
	}
	return len(removeOverlaps(boxes)), best, nil
}

// Handles both output layouts: [84 x N] and [N x 84].
func personBoxes(output []float32, rows, columns int, threshold float32) ([]personBox, float32, error) {
	if len(output) != rows*columns {
		return nil, 0, errors.New("YOLO returned inconsistent output data")
	}
	var candidates int
	var value func(candidate, field int) float32
	switch {
	case rows == yoloValuesPerBox:
		candidates = columns
		value = func(candidate, field int) float32 { return output[field*columns+candidate] }
	case columns == yoloValuesPerBox:
		candidates = rows
		value = func(candidate, field int) float32 { return output[candidate*columns+field] }
	default:
		return nil, 0, fmt.Errorf("YOLO returned unsupported output shape %dx%d", rows, columns)
	}

	var boxes []personBox
	var best float32
	for candidate := 0; candidate < candidates; candidate++ {
		score := value(candidate, personScoreIndex)
		if score < 0 || score > 1 || math.IsNaN(float64(score)) {
			return nil, 0, errors.New("YOLO returned an invalid person confidence")
		}
		best = max(best, score)
		if score < threshold {
			continue
		}
		cx, cy := value(candidate, 0), value(candidate, 1)
		w, h := value(candidate, 2), value(candidate, 3)
		boxes = append(boxes, personBox{x1: cx - w/2, y1: cy - h/2, x2: cx + w/2, y2: cy + h/2, score: score})
	}
	return boxes, best, nil
}

// Greedy NMS: strongest box first, drop anything overlapping it.
func removeOverlaps(boxes []personBox) []personBox {
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].score > boxes[j].score })
	var kept []personBox
	for _, box := range boxes {
		duplicate := false
		for _, other := range kept {
			if overlap(box, other) > overlapThreshold {
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, box)
		}
	}
	return kept
}

// IoU of two boxes.
func overlap(a, b personBox) float32 {
	width := min(a.x2, b.x2) - max(a.x1, b.x1)
	height := min(a.y2, b.y2) - max(a.y1, b.y1)
	if width <= 0 || height <= 0 {
		return 0
	}
	intersection := width * height
	union := (a.x2-a.x1)*(a.y2-a.y1) + (b.x2-b.x1)*(b.y2-b.y1) - intersection
	if union <= 0 {
		return 0
	}
	return intersection / union
}
