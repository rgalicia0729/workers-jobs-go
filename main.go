package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Job struct {
	Name   string
	Delay  time.Duration
	Number int
}

type Worker struct {
	Id         int
	JobQueue   chan Job
	WorkerPool chan chan Job
	QuitChan   chan bool
}

func NewWorker(id int, workerPool chan chan Job) *Worker {
	return &Worker{
		Id:         id,
		JobQueue:   make(chan Job),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool),
	}
}

func (w *Worker) Fibonacci(n int) int {
	if n <= 1 {
		return n
	}

	return w.Fibonacci(n-1) + w.Fibonacci(n-2)
}

func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobQueue

			select {
			case job := <-w.JobQueue:
				fmt.Printf("Worker with id %d started\n", w.Id)
				fib := w.Fibonacci(job.Number)
				time.Sleep(job.Delay)
				fmt.Printf("Worker with id %d finished with result %d\n", w.Id, fib)

			case <-w.QuitChan:
				fmt.Printf("Worker with id %d Stopped\n", w.Id)
			}
		}
	}()
}

func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

type Dispatcher struct {
	WorkerPool chan chan Job
	MaxWorkers int
	JobQueue   chan Job
}

func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		WorkerPool: workerPool,
		JobQueue:   jobQueue,
		MaxWorkers: maxWorkers,
	}
}

func (d *Dispatcher) Dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			go func() {
				workerJobQueue := <-d.WorkerPool
				workerJobQueue <- job
			}()
		}
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(i, d.WorkerPool)
		worker.Start()
	}

	go d.Dispatch()
}

func RequestHandler(res http.ResponseWriter, req *http.Request, jobQueue chan Job) {
	if req.Method != "POST" {
		res.Header().Set("Allow", "POST")
		res.WriteHeader(http.StatusMethodNotAllowed)
	}

	delay, err := time.ParseDuration(req.FormValue("delay"))
	if err != nil {
		http.Error(res, "Invalid Delay", http.StatusBadRequest)
		return
	}

	value, err := strconv.Atoi(req.FormValue("value"))
	if err != nil {
		http.Error(res, "Invalid Value", http.StatusBadRequest)
		return
	}

	name := req.FormValue("name")

	if name == "" {
		http.Error(res, "Invalid Name", http.StatusBadRequest)
		return
	}

	job := Job{
		Name:   name,
		Delay:  delay,
		Number: value,
	}

	jobQueue <- job

	res.WriteHeader(http.StatusCreated)
}

func main() {
	const (
		maxWorkers   = 4
		maxQueueSize = 20
		port         = ":8081"
	)

	jobQueue := make(chan Job, maxQueueSize)
	dispatcher := NewDispatcher(jobQueue, maxWorkers)

	dispatcher.Run()

	http.HandleFunc("/fib", func(res http.ResponseWriter, req *http.Request) {
		RequestHandler(res, req, jobQueue)
	})

	log.Fatal(http.ListenAndServe(port, nil))
}
