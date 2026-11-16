# Extension: crowd marker

## What it does

The baseline answers one question: was there a person in the recording. It
stops at the first hit and puts a fixed 30 second marker at the start of the
recording. Hub's own classification marks every pedestrian separately. Neither
of them tells you when the shop is busy.

This extension counts the people in every sampled frame. With `MIN_PEOPLE=N`
the worker only creates a marker for the part of the recording where at least
N people are visible at the same time. The marker is named `N or more people`
and its description holds the peak count and when it happened. The same text
is written to the recording description, so it can be searched in Hub.

With the default `MIN_PEOPLE=1` the marker keeps the name `person detected`,
but it runs from the first to the last sighting instead of a fixed 30 seconds.

## How it works

1. Every sampled frame (`YOLO_SAMPLE_FPS`) goes through YOLO11n. Boxes with a
   person score of at least `YOLO_CONFIDENCE` are kept.
2. Overlapping boxes are merged with a greedy NMS (IoU above 0.45 is the same
   person). What is left is the number of people in that frame.
3. A frame is a hit when it has at least `MIN_PEOPLE` people. The marker runs
   from the first hit to one second after the last hit.
4. No hit means the run is resolved without a marker, the same way the
   baseline handles a recording without a person.

The code is in `internal/app/personcount.go`, the frame loop in
`internal/app/yolo.go` and the marker in `internal/app/persondetection.go`.

## Run it

Set `MIN_PEOPLE=5` in `.env` and restart the worker. Everything else is the
same as the baseline.

## Results

All recordings are 29 second clips from the exercise Agent, taken on
16 Nov 2026.

### Count compared with Hub's classification

17 recordings were processed with `MIN_PEOPLE=1`, so every description holds
the worker's peak count. As a reference I used the `Pedestrian` markers that
Hub created for the same recording, read from the Markers table on the
recording page. The reference peak is the highest number of `Pedestrian`
markers that overlap in the same second.

| | |
|---|---|
| Same peak as Hub | 6 of 17 |
| Within one person | 14 of 17 |
| Mean difference (worker - Hub) | -0.24 |
| Mean absolute difference | 0.94 |
| Correlation | 0.80 |

The full table is in `docs/count-comparison.csv`.

Hub is a second system, not ground truth. It sometimes splits one person into
several markers, which pushes its peak up. The three biggest gaps are
18:37:32 (worker 5, Hub 2), 18:42:00 (worker 7, Hub 10) and 18:39:01
(worker 3, Hub 5).

### Choosing MIN_PEOPLE

Worker peak over the same 17 recordings:

| MIN_PEOPLE | Recordings that would get a marker |
|---|---|
| 3 | 13 of 17 |
| 4 | 11 of 17 |
| 5 | 7 of 17 |
| 6 | 4 of 17 |

I picked 5. It marks a bit less than half of the recordings, so a marker
still means something for this camera.

### Run with MIN_PEOPLE=5

| Recording | Worker | Hub peak |
|---|---|---|
| 18:43:29 | marker 0:00-0:29, peak 7 | 6 |
| 18:51:42 | marker 0:22-0:23, peak 5 | 6 |
| 18:52:12 | marker 0:07-0:13, peak 5 | 6 |
| 18:52:42 | marker 0:10-0:11, peak 5 | 3 |
| 18:53:12 | no marker | 5 |
| 18:53:41 | no marker | 5 |
| 18:54:11 | marker 0:00-0:20, peak 6 | 9 |
| 18:54:41 | marker 0:01-0:23, peak 6 | 8 |
| 18:55:11 | marker 0:00-0:20, peak 8 | 7 |

- Positive example: 18:52:12. The marker covers 0:07-0:13 and Hub also sees a
  busy shop in that recording.
- Counterexample: 18:53:12 and 18:53:41 end without a marker and without a
  description. Both are close to the limit: Hub counts 5 for a few seconds,
  the worker stays below 5.
- False alarm: 18:52:42. One sampled frame reaches 5 while Hub never sees
  more than 3.

## Limitations

- The count is the peak number of people at the same moment, not the number
  of different visitors. There is no tracking.
- YOLO11n is a small model and the frame is stretched to 640x640. It misses
  small or half hidden people, for example on the street behind the door.
- One sampled frame is enough to create a marker, see 18:52:42. Requiring a
  few hits in a row would remove most of these.
- There is one marker per recording. If the shop is busy, quiet and busy
  again, the quiet part is inside the marker.
- Time is frame index divided by the frame rate of the file. The marker can
  end up to one second after the recording does (0:30 on a 29 second clip).
- `MARKER_DURATION_SECONDS` is still validated but no longer sets the length
  of the marker.
- The comparison is 17 clips from one camera within one hour.

## Note on the baseline

The worker sends the bearer token to any HTTPS callback URL whose path
contains `/workflows/` and ends in the run ID. Someone who can publish to the queue could point it
at another host. The token only has `workflow-runs.update` and expires, so the
damage is limited, but checking the callback host against an allow list would
close it.
