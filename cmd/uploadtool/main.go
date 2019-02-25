package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dimdiden/oo"
)

const chunkSizeDefault int = 100

func main() {
	// All variables needed for upload
	var (
		api     string
		secret  string
		file    string
		ecode   string
		name    string
		chunk   int
		verbose bool
	)
	// The root usage
	flag.Usage = func() {
		fmt.Println("usage: uploadtool <command> [<args>]")
		fmt.Println("use image or asset for <command>")
		flag.PrintDefaults()
	}
	// List of flags for image subcommand
	imageCommand := flag.NewFlagSet("image", flag.ExitOnError)
	imageCommand.StringVar(&api, "a", "", "specify api key")
	imageCommand.StringVar(&secret, "s", "", "specify secret key")
	imageCommand.StringVar(&file, "f", "", "specify path to the image file")
	imageCommand.StringVar(&ecode, "e", "", "specify embed code to load the image for")
	imageCommand.BoolVar(&verbose, "v", false, "verbose mode")
	// List of flags for asset subcommand
	assetCommand := flag.NewFlagSet("asset", flag.ExitOnError)
	assetCommand.StringVar(&api, "a", "", "specify api key")
	assetCommand.StringVar(&secret, "s", "", "specify secret key")
	assetCommand.StringVar(&file, "f", "", "specify path to the video file")
	assetCommand.StringVar(&ecode, "e", "", "[optional] specify embed code for the content-replacement procedure")
	assetCommand.StringVar(&name, "n", "", "[optional] specify the asset name")
	assetCommand.IntVar(&chunk, "ch", chunkSizeDefault, "[optional] specify the chunk size. Default 100MB")
	assetCommand.BoolVar(&verbose, "v", false, "verbose mode")
	// Check if subcommands are provided
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	// Switch on the subcommand
	switch os.Args[1] {
	case "image":
		imageCommand.Parse(os.Args[2:])
	case "asset":
		assetCommand.Parse(os.Args[2:])
	default:
		flag.Usage()
		os.Exit(1)
	}

	ooClient, _ := oo.NewClient(secret, api, oo.BacklotDefaultEndpoint, 15)
	if verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	uploader := oo.NewUploader(ooClient)
	bars := newBars()
	uploader.SetStartFunc(bars.Start)
	uploader.SetFilterFunc(bars.filterFunc)
	uploader.SetDeferFunc(bars.deferFunc)

	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Image flags validation
	if imageCommand.Parsed() {
		if file == "" || secret == "" || api == "" || ecode == "" {
			imageCommand.PrintDefaults()
			os.Exit(1)
		}

		if err := uploader.UploadImage(f, ecode); err != nil {
			log.Fatal(err)
		}
		fmt.Println("The image has been uploaded for asset ", ecode)
	}

	// Asset flags validation
	if assetCommand.Parsed() {
		if file == "" || secret == "" || api == "" {
			assetCommand.PrintDefaults()
			os.Exit(1)
		}

		chunksize := chunk * 1024 * 1024
		if ecode == "" {
			asset, err := uploader.CreateUploadAsset(f, name, chunksize)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Video has been uploaded, embed code: ", asset.EmbedCode)
		} else {
			asset, err := uploader.ReplaceUploadAsset(f, chunksize, ecode)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Video has replaced for embed code: ", asset.EmbedCode)
		}
	}
}
