package workers

import (
	"log"
	"mvp-clipper/internal/services/ffmpeg"
	"mvp-clipper/internal/services/yt"
)

func StartWorker() {
	go func() {
		for job := range JobQueue {

			log.Println("Processing job:", job.URL)

			video, _ := yt.DownloadVideo(job.URL, "tmp/downloads")

			clip := "tmp/clips/output.mp4"
			ffmpeg.Cut(video, clip, job.Start, job.End)

			if job.Portrait {
				ffmpeg.ToPortrait(clip, "tmp/clips/portrait.mp4")
			}

			log.Println("Clip done!")
		}
	}()
}
