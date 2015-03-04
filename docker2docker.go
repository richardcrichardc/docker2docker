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

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-s address] [-d address] image [...] \n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Efficiently copies images between two Docker daemons. Only layers that
are not already present at the destination are transfered.

Source and destination addresses can be in the following formats:
	unix:///path/to/unix/socket
	tcp://host:port
	sshunix://[user@]host:[/path/to/unix/socket]

Unix and tcp are the usual docker transports.

Sshunix tunnels to a unix domain socket from a remote host over ssh, it
requires the 'socat' command to be installed on the remote host.

TLS/SSL is not supported.
`)
	}
}

func main() {
	// Parse command line arguments

	var srcAddr, dstAddr string

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
	srcClient, err := NewRemoteClient(srcAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err.Error())
		flag.Usage()
		os.Exit(1)
	}
	dstClient, err := NewRemoteClient(dstAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err.Error())
		flag.Usage()
		os.Exit(1)
	}

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
			// transfer layers bottom up to avoid missing image errors from dest
			j := len(layers) - 1 - i
			layer := layers[j]

			shortId := layer.Id[:10]

			// The image export does not include the repository and tag in the archive
			// if a hash is used to identify the image. So use the image name from the
			// command line in the export of the top layer.
			var layerId string
			if j == 0 {
				layerId = image
			} else {
				layerId = layer.Id
			}

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
					srcReader, _, err := srcClient.Get("/images/" + layerId + "/get?noparents=1")
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
