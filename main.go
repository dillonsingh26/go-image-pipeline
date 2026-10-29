package main

import (
	"flag"
	"fmt"
	imageprocessing "goroutines_pipeline/image_processing"
	"image"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Job struct {
	InputPath string
	Image     image.Image
	OutPath   string
	Err       error // records the first thing that went wrong for this image
}

// The photos the pipeline processes.
var imagePaths = []string{
	"images/hawaii-big-island.jpeg",
	"images/melbourne.jpg",
	"images/selfie.jpg",
	"images/yosemite.jpeg",
}

// jobFor sets up a Job's input and output paths without loading the image.
func jobFor(path string) Job {
	return Job{InputPath: path,
		OutPath: strings.Replace(path, "images/", "images/output/", 1)}
}

// newJob sets up a Job and loads the image.
func newJob(path string) Job {
	job := jobFor(path)
	job.Image, job.Err = imageprocessing.ReadImage(path)
	return job
}

// ---------- Concurrent version: one goroutine per stage ----------

func loadImage(paths []string) <-chan Job {
	out := make(chan Job)
	go func() {
		for _, p := range paths {
			out <- newJob(p) // send it along even if it failed, so main can report it
		}
		close(out)
	}()
	return out
}

func resize(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		for job := range input {
			if job.Err == nil { // only process images that loaded successfully
				job.Image = imageprocessing.Resize(job.Image)
			}
			out <- job
		}
		close(out)
	}()
	return out
}

func convertToGrayscale(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		for job := range input {
			if job.Err == nil {
				job.Image = imageprocessing.Grayscale(job.Image)
			}
			out <- job
		}
		close(out)
	}()
	return out
}

func saveImage(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		for job := range input {
			if job.Err == nil {
				job.Err = imageprocessing.WriteImage(job.OutPath, job.Image)
			}
			out <- job
		}
		close(out)
	}()
	return out
}

// runConcurrent chains the four stages together with channels.
func runConcurrent(paths []string) []Job {
	results := saveImage(convertToGrayscale(resize(loadImage(paths))))

	var jobs []Job
	for job := range results {
		jobs = append(jobs, job)
	}
	return jobs
}

// ---------- Fan-out version: several goroutines per stage ----------

// generateJobs sends one Job per path without loading the image yet,
// so the slow decoding step can be fanned out too.
func generateJobs(paths []string) <-chan Job {
	out := make(chan Job)
	go func() {
		for _, p := range paths {
			out <- jobFor(p)
		}
		close(out)
	}()
	return out
}

// decodeImage loads each job's image from disk.
func decodeImage(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		for job := range input {
			job.Image, job.Err = imageprocessing.ReadImage(job.InputPath)
			out <- job
		}
		close(out)
	}()
	return out
}

// fanOut starts several copies of a stage that all pull from the same input channel,
// then merges their outputs back into one channel.
func fanOut(input <-chan Job, workers int, stage func(<-chan Job) <-chan Job) <-chan Job {
	outputs := make([]<-chan Job, workers)
	for i := 0; i < workers; i++ {
		outputs[i] = stage(input)
	}
	return merge(outputs...)
}

// merge combines several channels into one (fan-in).
// It closes the combined channel once every input channel is finished.
func merge(channels ...<-chan Job) <-chan Job {
	out := make(chan Job)
	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(ch <-chan Job) {
			defer wg.Done()
			for job := range ch {
				out <- job
			}
		}(ch)
	}

	go func() {
		wg.Wait() // wait until every worker is done
		close(out)
	}()
	return out
}

// runFanOut runs every stage with several workers.
func runFanOut(paths []string, workers int) []Job {
	if workers < 1 {
		workers = 1
	}

	jobs := generateJobs(paths)
	decoded := fanOut(jobs, workers, decodeImage)
	resized := fanOut(decoded, workers, resize)
	grayed := fanOut(resized, workers, convertToGrayscale)
	results := fanOut(grayed, workers, saveImage)

	var done []Job
	for job := range results {
		done = append(done, job)
	}
	return done
}

// ---------- Sequential version: a plain for loop, no goroutines ----------

// runSequential sends each image through all four steps before starting the next one.
func runSequential(paths []string) []Job {
	var jobs []Job
	for _, p := range paths {
		job := newJob(p)
		if job.Err == nil {
			job.Image = imageprocessing.Resize(job.Image)
			job.Image = imageprocessing.Grayscale(job.Image)
			job.Err = imageprocessing.WriteImage(job.OutPath, job.Image)
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func main() {
	mode := flag.String("mode", "concurrent", "sequential, concurrent, or fanout")
	workers := flag.Int("workers", runtime.NumCPU(), "goroutines per stage in fanout mode")
	flag.Parse()

	var run func([]string) []Job
	var label string

	switch *mode {
	case "sequential":
		run = runSequential
		label = "sequential (no goroutines)"
	case "concurrent":
		run = runConcurrent
		label = "concurrent (1 goroutine per stage)"
	case "fanout":
		run = func(paths []string) []Job { return runFanOut(paths, *workers) }
		label = fmt.Sprintf("fan-out (%d goroutines per stage)", *workers)
	default:
		fmt.Println("Unknown mode:", *mode, "(use sequential, concurrent, or fanout)")
		os.Exit(1)
	}

	start := time.Now()
	jobs := run(imagePaths)
	elapsed := time.Since(start)

	failed := 0
	for _, job := range jobs {
		if job.Err != nil {
			fmt.Println("Failed:", job.Err)
			failed++
		} else {
			fmt.Println("Success:", job.OutPath)
		}
	}
	fmt.Printf("Done: %d succeeded, %d failed\n", len(jobs)-failed, failed)
	fmt.Printf("Mode: %s | Time: %v\n", label, elapsed)
}
