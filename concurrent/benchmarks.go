package concurrent

import (
	"fmt"
	"os"
	"time"
)

var (
	OutputFile = "results.txt"
)

func WriteToFile(filename string, data string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// file does not exist
		// create file
		os.Create(filename)
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	_, err = fmt.Fprintf(f, "%s", data)
	return  err
}

func ProcessTimer(start time.Time, filename string, txCount *int64) {
	nanoseconds := time.Since(start).Nanoseconds()
	seconds := float64(nanoseconds) / 1e9
	throughput := float64(*txCount) / seconds

	s := fmt.Sprintf("%d\t%f\n", NumThreads, throughput)
	_ = WriteToFile(filename, s)
}