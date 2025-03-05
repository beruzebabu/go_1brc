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
	buf := make([]byte, 4096*4096)
	scanner.Buffer(buf, 4096*32768)
	for scanner.Scan() {
		token := scanner.Bytes()
		i := slices.Index(token, 0x3B)

		if i < 0 {
			continue
		}

		station := string(token[:i])
		mant, exp, neg, _, _, _, ok := readFloat(string(token[i+1:]))
		reading, ok := atof64exact(mant, exp, neg) // this could be faster, but would require a different implementation which takes more shortcuts
		if !ok {
			log.Fatalln("Failed to parse to float", string(token[i+1:]))
		}
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

	return nil
}

func sum[T cmp.Ordered](slice []T) T {
	var sum T
	for _, v := range slice {
		sum += v
	}
	return sum
}

// FROM STDLIB BUT UNNECESSARY PARTS REMOVED
func readFloat(s string) (mantissa uint64, exp int, neg, trunc, hex bool, i int, ok bool) {
	// optional sign
	if i >= len(s) {
		return
	}
	switch {
	case s[i] == '+':
		i++
	case s[i] == '-':
		neg = true
		i++
	}

	// digits
	base := uint64(10)
	maxMantDigits := 19 // 10^19 fits in uint64
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
loop:
	for ; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '.':
			if sawdot {
				break loop
			}
			sawdot = true
			dp = nd
			continue

		case '0' <= c && c <= '9':
			sawdigits = true
			if c == '0' && nd == 0 { // ignore leading zeros
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}

	if mantissa != 0 {
		exp = dp - ndMant
	}

	ok = true
	return
}

type floatInfo struct {
	mantbits uint
	expbits  uint
	bias     int
}

var float64info = floatInfo{52, 11, -1023}
var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

func atof64exact(mantissa uint64, exp int, neg bool) (f float64, ok bool) {
	if mantissa>>float64info.mantbits != 0 {
		return
	}
	f = float64(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		// an integer.
		return f, true
	// Exact integers are <= 10^15.
	// Exact powers of ten are <= 10^22.
	case exp > 0 && exp <= 15+22: // int * 10^k
		// If exponent is big but number of digits is not,
		// can move a few zeros into the integer part.
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f > 1e15 || f < -1e15 {
			// the exponent was really too large.
			return
		}
		return f * float64pow10[exp], true
	case exp < 0 && exp >= -22: // int / 10^k
		return f / float64pow10[-exp], true
	}
	return
}

// END STDLIB EDITS

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
