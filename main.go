package main

import (
	"os"
	"fmt"
	"strconv"
	"strings"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"os/exec"
	"bufio"
	"math"
	"image/color"
)

var dataSet [][]string
var start, end float64
var topXys plotter.XYs
var binWidth float64
var xysLength int

func getData() {
	//[-h] # this help
	//[-d serial_number] # Serial number of desired HackRF
	//[-a amp_enable] # RX RF amplifier 1=Enable, 0=Disable
	//[-f freq_min:freq_max] # minimum and maximum frequencies in MHz
	//[-p antenna_enable] # Antenna port power, 1=Enable, 0=Disable
	//[-l gain_db] # RX LNA (IF) gain, 0-40dB, 8dB steps
	//[-g gain_db] # RX VGA (baseband) gain, 0-62dB, 2dB steps
	//[-n num_samples] # Number of samples per frequency, 16384-4294967296
	//[-w bin_width] # FFT bin width (frequency resolution) in Hz
	//[-1] # one shot mode
	//[-B] # binary output

	var args []string

	args = append(args, "-a 1")
	args = append(args, fmt.Sprintf("-f %v:%v", start, end))
	args = append(args, fmt.Sprintf("-n %v", 2 * 16384))
	args = append(args, fmt.Sprintf("-w %v", 10 * 1222)) // Sample rate (20Mhz) / 1222 ~= 16368
	args = append(args, "-1")


	cmd := exec.Command("hackrf_sweep", args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		dataSet = append(dataSet, strings.Split(scanner.Text(), ", "))
	}
}

func init() {
	start = 2400
	end = 2500

	// Round to upper 20MHz
	end = 20 * math.Ceil((end - start) / 20) + start
	getData()
	binWidth, _ = strconv.ParseFloat(dataSet[0][4], 64)
	//  xys length = end - start (in hz) / bin width (in hz)
	xysLength = int(((end - start) * 1000 * 1000) / binWidth)
}

func main() {
	var xys = make(plotter.XYs, xysLength)

	for _, data := range dataSet {
		s, _ := strconv.ParseFloat(data[2], 64)

		data = data[6:]
		for i, d := range data {
			freq := s + (float64(i) * binWidth)
			freq = freq / 1000 / 1000

			if freq < start {
				continue
			}

			pos := int(((freq - start) / (end - start)) * float64(xysLength))

			y, _ := strconv.ParseFloat(d, 64)

			xys[pos] = struct {X,Y float64} {
				X: freq,
				Y: y,
			}
		}
	}

	if len(topXys) == 0 {
		topXys = xys
	}

	for i, d := range xys {
		if d.X == 0.0 {
			xys[i] = xys[i - 1]
		} else {
			if d.Y > topXys[i].Y {
				topXys[i] = d
			}
		}
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Y.Max = 0
	p.Y.Min = -180

	p.Y.Label.Text = "dbs"
	p.X.Label.Text = "MHz"

	l, err := plotter.NewLine(xys)
	if err != nil {
		panic(err)
	}
	p.Add(l)

	s, err := plotter.NewLine(topXys)
	if err != nil {
		panic(err)
	}
	s.Color = color.RGBA{0, 0, 0, 64}
	p.Add(s)

	wt, err := p.WriterTo(2048, 256, "png")
	if err != nil {
		panic(err)
	}

	img, err := os.Create("/Users/trevorstarick/output.png")
	if err != nil {
		panic(err)
	}

	wt.WriteTo(img)

	fmt.Println("written")

	img.Close()

	getData()
	main()
}
