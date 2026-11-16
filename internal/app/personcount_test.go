package app

import "testing"

func yoloOutput(boxes ...[5]float32) []float32 {
	output := make([]float32, yoloValuesPerBox*len(boxes))
	for candidate, box := range boxes {
		for field, value := range box {
			output[field*len(boxes)+candidate] = value
		}
	}
	return output
}

func TestCountPersons(t *testing.T) {
	for _, test := range []struct {
		name  string
		boxes [][5]float32 // cx, cy, w, h, person score
		want  int
	}{
		{"empty frame", [][5]float32{{100, 100, 50, 120, 0.1}}, 0},
		{"one person", [][5]float32{{100, 100, 50, 120, 0.9}}, 1},
		{"one person seen twice", [][5]float32{{100, 100, 50, 120, 0.9}, {104, 102, 50, 120, 0.8}}, 1},
		{"two people apart", [][5]float32{{100, 100, 50, 120, 0.9}, {400, 100, 50, 120, 0.7}}, 2},
		{"two people side by side", [][5]float32{{100, 100, 50, 120, 0.9}, {140, 100, 50, 120, 0.8}}, 2},
		{"weak second box ignored", [][5]float32{{100, 100, 50, 120, 0.9}, {400, 100, 50, 120, 0.3}}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			count, _, err := countPersons(yoloOutput(test.boxes...), yoloValuesPerBox, len(test.boxes), 0.5)
			if err != nil || count != test.want {
				t.Fatalf("count = %d, error = %v, want %d", count, err, test.want)
			}
		})
	}
}

func TestCountPersonsLayouts(t *testing.T) {
	detectionsFirst := make([]float32, 2*yoloValuesPerBox)
	copy(detectionsFirst, []float32{100, 100, 50, 120, 0.9})
	copy(detectionsFirst[yoloValuesPerBox:], []float32{400, 100, 50, 120, 0.76})
	count, best, err := countPersons(detectionsFirst, 2, yoloValuesPerBox, 0.5)
	if err != nil || count != 2 || best != 0.9 {
		t.Fatalf("detections-first count = %d, best = %v, error = %v", count, best, err)
	}
	if _, _, err := countPersons(make([]float32, 4), 2, 2, 0.5); err == nil {
		t.Fatal("accepted unsupported YOLO output")
	}
	if _, _, err := countPersons(yoloOutput([5]float32{1, 1, 1, 1, 1.5}), yoloValuesPerBox, 1, 0.5); err == nil {
		t.Fatal("accepted an invalid person confidence")
	}
}
