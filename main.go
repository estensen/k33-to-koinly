package main

import (
	"flag"
	"log"
	"os"

	"k33-to-koinly/converter"
)

func main() {
	inPath := flag.String("in", "k33.csv", "K33 export CSV file")
	outPath := flag.String("out", "koinly.csv", "Koinly universal CSV output")
	dryrun := flag.Bool("dryrun", false, "Print mapped rows without writing file")
	flag.Parse()

	in, err := os.Open(*inPath)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer in.Close()

	if *dryrun {
		conv := converter.New()
		if err := conv.ProcessDryRun(in); err != nil {
			log.Fatal(err)
		}
		return
	}

	out, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer out.Close()

	conv := converter.New()
	if err := conv.Process(in, out); err != nil {
		log.Fatal(err)
	}

	log.Printf("Successfully converted %s to %s", *inPath, *outPath)
}