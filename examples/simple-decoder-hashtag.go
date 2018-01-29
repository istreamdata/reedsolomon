//+build ignore

// Copyright 2015, Klaus Post, see LICENSE for details.
//
// Simple decoder example.
//
// The decoder reverses the process of "simple-encoder.go"
//
// To build an executable use:
//
// go build simple-decoder.go
//
// Simple Encoder/Decoder Shortcomings:
// * If the file size of the input isn't diviable by the number of data shards
//   the output will contain extra zeroes
//
// * If the shard numbers isn't the same for the decoder as in the
//   encoder, invalid output will be generated.
//
// * If values have changed in a shard, it cannot be reconstructed.
//
// * If two shards have been swapped, reconstruction will always fail.
//   You need to supply the shards in the same order as they were given to you.
//
// The solution for this is to save a metadata file containing:
//
// * File size.
// * The number of data/parity shards.
// * HASH of each shard.
// * Order of the shards.
//
// If you save these properties, you should abe able to detect file corruption
// in a shard and be able to reconstruct your data if you have the needed number of shards left.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"reedsolomon.git/trunk"
)

var dataShards = flag.Int("data", 5, "Number of shards to split the data into")
var parShards = flag.Int("par", 2, "Number of parity shards")
var outFile = flag.String("out", "", "Alternative output path/file")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  simple-decoder [-flags] basefile.ext\nDo not add the number to the filename.\n")
		fmt.Fprintf(os.Stderr, "Valid flags:\n")
		flag.PrintDefaults()
	}
}

func main() {
	// Parse flags
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: No filenames given\n")
		flag.Usage()
		os.Exit(1)
	}
	fname := args[0]

	// Detect storage node failures
	pIfFailedSN := make([]bool,*dataShards+*parShards)
	numOfFailedNodes := 0
	shardSize := int64(0)
	for i := 0; i<*dataShards+*parShards; i++ {
		infn := fmt.Sprintf("%s.%d", fname, i)
		// check whether file exists or not
		if fi, err := os.Stat(infn); os.IsNotExist(err) {
			fmt.Println("Failure of %d-th storage node has been detected.\n", i)
			pIfFailedSN[i] = true
			numOfFailedNodes++
		} else {
			shardSize = fi.Size()
		}
	}

	// Create HashTagCodec
	encH, err := reedsolomon.NewHashTagCode(*dataShards, *parShards)
	checkErr(err)

	alpha:=encH.GetNumOfSubchunksInChunk()
	subshardSize := shardSize/int64(alpha)

	shards := make([][]byte, (*dataShards+*parShards)*alpha)

	err = encH.Repair(fname, pIfFailedSN, subshardSize, shards)
	checkErr(err)

	// reconstruct file
	// read data from k non-failed storage nodes
	// write reconstructed file

	err = encH.Reconstruct(fname, subshardSize, shards)
	checkErr(err)

	// Join the shards and write them
	outfn := *outFile
	if outfn == "" {
		outfn = CreateOutputFileName(fname)
	}

	fmt.Println("Writing data to", outfn)
	f, err := os.Create(outfn)
	checkErr(err)

	// We don't know the exact filesize.
	err = encH.Join(f, shards, len(shards[0])*(*dataShards*alpha))
	checkErr(err)
	err = f.Close()
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(2)
	}
}

func CreateOutputFileName(fname string) string {
	lines := strings.Split(fname, ".")
	linesNum := len(lines)
	if linesNum > 1 {
		lines[linesNum-2] = fmt.Sprintf("%s_Reconstructed.", lines[linesNum-2])
	}
	outfn :=lines[0]
	for i:=1;i<linesNum;i++ {
		outfn = fmt.Sprintf("%s%s", outfn, lines[i])
	}
	return outfn
}

