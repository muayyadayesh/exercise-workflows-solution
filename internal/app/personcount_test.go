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

func TestMinPeople(t *testing.T) {
	counts := []int{0, 1, 2, 3, 4, 2, 3, 1} // one frame per second
	for _, test := range []struct {
		minPeople   int
		detected    bool
		first, last float64
	}{
		{0, true, 1, 7},
		{1, true, 1, 7},
		{3, true, 3, 6},
		{4, true, 4, 4},
		{5, false, 0, 0},
	} {
		var detection personDetection
		for second, count := range counts {
			detection.add(count, float64(second), test.minPeople)
		}
		if detection.Detected != test.detected || detection.FirstSeen != test.first || detection.LastSeen != test.last ||
			detection.PeakCount != 4 || detection.PeakAt != 4 {
			t.Fatalf("minPeople %d: %+v", test.minPeople, detection)
		}
	}
}

func TestCrowdMarkerText(t *testing.T) {
	detection := personDetection{Detected: true, FramesAnalyzed: 30, MaxConfidence: 0.9, PeakCount: 4, PeakAt: 26, FirstSeen: 21, LastSeen: 28, MinPeople: 3}
	want := "Up to 4 people at once (peak at 0:26), 3 or more visible 0:21-0:29; max confidence 0.90 over 30 sampled frames"
	if got := personSummary(detection, 21, 29); got != want {
		t.Fatalf("summary = %q", got)
	}
	if markerName(3) != "3 or more people" || markerName(1) != personMarkerName || markerName(0) != personMarkerName {
		t.Fatal("unexpected marker name")
	}
}
