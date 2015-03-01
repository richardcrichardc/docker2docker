package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type imageInfo struct {
	Id, Parent string
	Size       int64
}

func main() {
	// Parse command line arguments

	var srcAddr, dstAddr string
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-s address] [-d address] image [...] \n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Source and destination addresses are in the usual format, i.e. unix://socket or tcp://host:port.\n")
		fmt.Fprintf(os.Stderr, "SSL connections are not supported.\n\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&srcAddr, "s", "unix:///var/run/docker.sock", "Source docker daemon")
	flag.StringVar(&dstAddr, "d", "unix:///var/run/docker.sock", "Destination docker daemon")
	flag.Parse()

	if srcAddr == dstAddr {
		fmt.Fprintf(os.Stderr, "Source and destination docker instances should be different, make sure at least one is specified.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Please specify at least one image to transfer.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set up src and dst connections
	srcClient := NewRemoteClient(srcAddr)
	dstClient := NewRemoteClient(dstAddr)

	// Loop over and transfer images specified on command line
	for _, image := range flag.Args() {
		var layers []imageInfo

		fmt.Printf("Retrieving layer info of %s from %s...\n", image, srcAddr)

		for id := image; id != ""; {

			var info imageInfo
			err := srcClient.GetJSON("/images/"+id+"/json", &info)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			layers = append(layers, info)
			id = info.Parent
		}

		fmt.Printf("Transfering layers from %s to %s...\n", srcAddr, dstAddr)
		ticker := time.Tick(500 * time.Millisecond)
		for i := 0; i < len(layers); i++ {
			// transfer layers bottom upto avoid missing image errors from dest
			j := len(layers) - 1 - i
			layer := layers[j]

			shortId := layer.Id[:10]

			exists, err := dstClient.Exists("/images/" + layer.Id + "/json")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if exists {
				fmt.Printf("\r%d/%d %s... Already exists\n", i+1, len(layers), shortId)
			} else {

				var transfered int64
				done := make(chan bool)

				go func() {
					srcReader, _, err := srcClient.Get("/images/" + layer.Id + "/get?noparents=1")
					if err != nil {
						fmt.Fprintln(os.Stderr) // move past current progress line
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
					defer srcReader.Close()

					meteredReader := NewMeteredReader(srcReader, &transfered)
					err = dstClient.Post("/images/load", "application/x-tar", meteredReader)
					if err != nil {
						fmt.Fprintln(os.Stderr) // move past current progress line
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}

					done <- true
				}()

			progressLoop:
				for {
					printProgress(i, layers, shortId, transfered, layer.Size)

					select {
					case <-ticker:
						// loop and update progress
					case <-done:
						// break out of progressLoop and move to next layer
						break progressLoop
					}

				}
				printProgress(i, layers, shortId, transfered, layer.Size)
				fmt.Println()
			}
		}
	}
}

func printProgress(i int, layers []imageInfo, shortId string, transfered, size int64) {
	/*
		There is quite a large discrepancy between reported layer size and amount of data
		sent. So for now display progress as KB instead of as a percent.

		progress := float64(transfered) / float64(size) * 100.0

		// Some images have a size of 0. I assume these images only contain metadata.
		// Should transfer pretty quickly so always show progress as 100%
		if math.IsNaN(progress) || progress > 100.0 {
		  progress = 100.0
		}
	*/

	sizeMB := float64(size) / 1024.0 / 1024.0
	transferedMB := float64(transfered) / 1024.0 / 1024.0

	fmt.Printf("\r%d/%d %s... %.1fMB / ~ %.1fMB", i+1, len(layers), shortId, transferedMB, sizeMB)
}
