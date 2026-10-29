package main

import (
	"runtime"
	"testing"
)

// All three versions of the pipeline, so each test can check them side by side.
var pipelines = map[string]func([]string) []Job{
	"sequential": runSequential,
	"concurrent": runConcurrent,
	"fanout": func(paths []string) []Job {
		return runFanOut(paths, runtime.NumCPU())
	},
}

func TestAllModesProcessAllImages(t *testing.T) {
	for name, run := range pipelines {
		jobs := run(imagePaths)

		if len(jobs) != len(imagePaths) {
			t.Errorf("%s: expected %d results, got %d", name, len(imagePaths), len(jobs))
		}
		for _, job := range jobs {
			if job.Err != nil {
				t.Errorf("%s: unexpected error: %v", name, job.Err)
			}
		}
	}
}

func TestAllModesReportMissingFile(t *testing.T) {
	paths := []string{"images/does_not_exist.jpeg"}

	for name, run := range pipelines {
		jobs := run(paths)

		if len(jobs) != 1 {
			t.Fatalf("%s: expected 1 result, got %d", name, len(jobs))
		}
		if jobs[0].Err == nil {
			t.Errorf("%s: expected an error for a missing file, got nil", name)
		}
	}
}

// benchmarkPipeline runs a pipeline over and over and reports throughput.
func benchmarkPipeline(b *testing.B, run func([]string) []Job) {
	for i := 0; i < b.N; i++ {
		jobs := run(imagePaths)
		for _, job := range jobs {
			if job.Err != nil {
				b.Fatal(job.Err)
			}
		}
	}
	imagesProcessed := float64(b.N * len(imagePaths))
	b.ReportMetric(imagesProcessed/b.Elapsed().Seconds(), "images/sec")
}

func BenchmarkSequential(b *testing.B) {
	benchmarkPipeline(b, runSequential)
}

func BenchmarkConcurrent(b *testing.B) {
	benchmarkPipeline(b, runConcurrent)
}

func BenchmarkFanOut(b *testing.B) {
	benchmarkPipeline(b, pipelines["fanout"])
}
