# Concurrent Image Processing Pipeline in Go

This project replicates and extends Amrit Singh's image processing pipeline from [CODEHEIM](https://www.codeheim.io/), *Episode #21: Concurrency in Go: Pipeline Pattern* ([original repository](https://github.com/code-heim/go_21_goroutines_pipeline)).

The program loads JPEG photos, resizes them, converts them to grayscale, and saves the results. It can run three ways, so their speed can be compared:

- **Sequential:** a plain for loop with no goroutines
- **Concurrent:** the original pipeline, with one goroutine per stage connected by channels
- **Fan-out:** several goroutines per stage, with their results merged back together

## Requirements

- Go 1.20 or higher (developed with Go 1.27 on macOS, Apple M4, 10 cores)
- [`github.com/nfnt/resize`](https://github.com/nfnt/resize), which installs automatically with `go mod tidy`

## Project Structure

```
go_21_goroutines_pipeline/
├── main.go                          # pipeline logic and all three run modes
├── main_test.go                     # pipeline tests and benchmarks
├── image_processing/
│   ├── image_processing.go          # read, write, resize, and grayscale helpers
│   └── image_processing_test.go     # unit tests for the helpers
├── images/                          # input photos
│   └── output/                      # processed photos are written here
└── docs/                            # before/after images for this README
```

## How to Run

```bash
git clone https://github.com/dillonsingh26/go_21_goroutines_pipeline.git
cd go_21_goroutines_pipeline
go mod tidy
```

Run the pipeline in any of the three modes:

```bash
go run . -mode=sequential
go run . -mode=concurrent        # default
go run . -mode=fanout
go run . -mode=fanout -workers=4 # choose goroutines per stage (default: number of CPU cores)
```

Each run prints the result for every photo, a summary, and the total processing time:

```
Success: images/output/hawaii-big-island.jpeg
Success: images/output/melbourne.jpg
Success: images/output/selfie.jpg
Success: images/output/yosemite.jpeg
Done: 4 succeeded, 0 failed
Mode: fan-out (10 goroutines per stage) | Time: 205ms
```

## Testing

```bash
go test ./... -v
```

There are nine tests in total.

**Image helpers (`image_processing_test.go`)**

| Test | What it checks |
|---|---|
| `TestWriteThenReadImage` | An image can be saved and read back with its size intact |
| `TestReadImageMissingFile` | A missing file returns an error instead of crashing |
| `TestReadImageNotAnImage` | A file that isn't a real image returns an error |
| `TestWriteImageCreatesFolder` | Saving creates the output folder if needed |
| `TestWriteImageNilImage` | Saving an empty image returns an error |
| `TestResize` | Landscape, portrait, and square images keep their aspect ratio |
| `TestGrayscale` | Output is a grayscale image of the same size |

The helper tests build small images in memory and use temporary folders, so they don't depend on the photos in `images/`.

**Pipeline (`main_test.go`)**

| Test | What it checks |
|---|---|
| `TestAllModesProcessAllImages` | All three modes process every photo without errors |
| `TestAllModesReportMissingFile` | All three modes report a missing file instead of crashing |

## Benchmarks

```bash
go test -bench=. -benchtime=10x -run='^$'
```

Each mode processed all four photos 10 times. Results on an Apple M4 (10 cores):

| Mode | Time per run | Throughput | Speedup vs. sequential |
|---|---|---|---|
| Sequential (no goroutines) | 319.7 ms | 12.51 images/sec | 1.00× |
| Concurrent (1 goroutine per stage) | 270.8 ms | 14.77 images/sec | 1.18× |
| Fan-out (10 goroutines per stage) | 205.3 ms | 19.48 images/sec | 1.56× |

### Discussion

Both concurrent versions beat the sequential loop, but by less than the core count alone would suggest.

The original pipeline uses one goroutine per stage, so it can use at most four cores at once, and it runs only as fast as its slowest stage. Decoding a full-size photo and resizing it with the Lanczos filter take far longer than grayscaling a 500-pixel image or saving a small JPEG, so the faster stages spend time waiting. With only four photos, the pipeline also spends a large share of each run filling up and emptying out.

Fan-out addresses the slow stages by giving every stage several workers that pull from the same channel, which lets several photos be decoded and resized at the same time. That raised the speedup from 1.18× to 1.56×. It is still limited because there are only four photos, so at most four workers per stage ever have work, and a single photo can't be split across workers. The run can't finish faster than the time needed to process the largest photo by itself.

Concurrency pays off most when there are many independent items. A larger batch of photos would show a much wider gap between fan-out and sequential processing.

## Changes from the Original

### 1. Error checking for input and output

The original code called `panic` on any file problem, which crashed the whole program.

- `ReadImage` now returns `(image.Image, error)`, and `WriteImage` returns `error`. Error messages name the file that failed.
- `WriteImage` creates the output folder if it doesn't exist, rejects empty images, and checks for errors when closing the file.
- Each `Job` carries an `Err` field through the pipeline. Stages skip jobs that already failed but still pass them along, so every failure is reported at the end while the other photos finish normally.
- This also fixes a bug in the original, where `saveImage` always sent `true`, so the `Failed!` message could never print.

### 2. New input images

The four sample images were replaced with my own photos: Hawaii's Big Island, Melbourne, a selfie, and Yosemite.

### 3. Unit tests

Nine tests were added, covering the image helpers and all three pipeline modes (see [Testing](#testing)).

### 4. Running with and without goroutines

- `runSequential` processes each photo through all four steps in a plain for loop before starting the next one.
- `runConcurrent` is the original pipeline, wrapped in a function.
- `runFanOut` runs each stage with several goroutines.
- A `-mode` flag selects the version, and each run is timed with `time.Now()` and `time.Since()`.
- Benchmarks report a custom `images/sec` throughput metric with `b.ReportMetric`.

### 5. Fan-out and fan-in

`fanOut` starts several copies of a stage, all reading from the same input channel, and `merge` combines their outputs into one channel. A `sync.WaitGroup` makes sure the merged channel only closes once every worker has finished. Loading was split into `generateJobs` and `decodeImage` so that decoding, one of the slowest steps, could be fanned out as well. In fan-out mode, photos can finish in a different order on each run.

### 6. Preserving the aspect ratio

The original `Resize` forced every photo into 500×500, which squished anything that wasn't square. The new version reads each photo's width and height in pixels, sets the longer side to 500, and scales the shorter side by the same factor. A 4000×3000 photo becomes 500×375, and a 3000×4000 photo becomes 375×500.

| Before (500×500) | After (aspect ratio preserved) |
|---|---|
| ![Squished selfie](docs/selfie_squished.jpg) | ![Selfie with correct proportions](docs/selfie_aspect_ratio.jpg) |

## Credits

- Original code and tutorial: Amrit Singh, [CODEHEIM](https://www.codeheim.io/), *Episode #21: Concurrency in Go: Pipeline Pattern*
- Image resizing: [nfnt/resize](https://github.com/nfnt/resize)
- License: Apache 2.0, inherited from the original repository
