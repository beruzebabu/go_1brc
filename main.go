package main

import (
	"bufio"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type CliFlags struct {
	File string
}

type StationResult struct {
	Station  string
	Min      float64
	Max      float64
	Mean     float64
	Readings int
}

func parseFlags() (CliFlags, error) {
	file := flag.String("file", "", "specify the file to process")
	flag.Parse()

	if *file == "" {
		return CliFlags{}, errors.New("no file specified")
	}

	return CliFlags{*file}, nil
}

func processFile(filepath string) error {
	log.Println("starting to process", filepath)
	start := time.Now()

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("opening file for reading failed: %w", err)
	}
	defer file.Close()

	stations := map[string]*StationResult{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		token := scanner.Bytes()
		i := slices.Index(token, 0x3B)

		if i < 0 {
			continue
		}

		station := string(token[:i])
		reading, _ := strconv.ParseFloat(string(token[i+1:]), 64) // this could be faster, but would require a different implementation which takes more shortcuts
		v, ok := stations[station]
		if !ok {
			stations[station] = &StationResult{Station: station, Min: reading, Max: reading, Mean: reading, Readings: 1}
			continue
		}

		if v.Min > reading {
			v.Min = reading
		} else if v.Max < reading {
			v.Max = reading
		}
		v.Mean += reading
		v.Readings += 1
	}

	log.Println("all readings read from file", time.Since(start))

	stationsSlice := []*StationResult{}
	for s, r := range stations {
		min := r.Min
		max := r.Max
		mean := r.Mean / float64(r.Readings)

		result := &StationResult{s, min, max, mean, 0}
		stationsSlice = append(stationsSlice, result)
	}

	log.Println("calculated min/max/mean", time.Since(start))

	slices.SortFunc(stationsSlice, func(a *StationResult, b *StationResult) int {
		return strings.Compare(a.Station, b.Station)
	})

	log.Println("sorted", time.Since(start))

	// for _, sr := range stationsSlice {
	// 	fmt.Println(*sr)
	// }

	return nil
}

func sum[T cmp.Ordered](slice []T) T {
	var sum T
	for _, v := range slice {
		sum += v
	}
	return sum
}

func main() {
	flags, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("started with args", flags)
	start := time.Now()

	err = processFile(filepath.Clean(flags.File))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("finished in", time.Since(start))
}
