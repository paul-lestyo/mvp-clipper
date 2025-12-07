package workers

var JobQueue = make(chan ClipJob, 100)

func Enqueue(job ClipJob) {
	JobQueue <- job
}
