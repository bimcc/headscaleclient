package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"
)

const (
	outputSize = 1024
	scale      = 4
)

func main() {
	output := flag.String("output", "build/appicon.png", "output PNG path")
	flag.Parse()

	canvas := image.NewRGBA(image.Rect(0, 0, outputSize*scale, outputSize*scale))
	paintIcon(canvas)

	resized := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), canvas, canvas.Bounds(), xdraw.Over, nil)

	file, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, resized); err != nil {
		log.Fatal(err)
	}
}

func paintIcon(img *image.RGBA) {
	background := color.RGBA{R: 22, G: 31, B: 36, A: 255}
	ink := color.RGBA{R: 244, G: 247, B: 244, A: 255}
	mint := color.RGBA{R: 70, G: 211, B: 164, A: 255}
	coral := color.RGBA{R: 255, G: 126, B: 103, A: 255}

	drawRoundedRect(img, 48, 48, 976, 976, 208, background)

	leftTop := point{300, 282}
	leftBottom := point{300, 742}
	rightTop := point{724, 282}
	rightBottom := point{724, 742}
	center := point{512, 512}

	drawLine(img, leftTop, leftBottom, 76, ink)
	drawLine(img, rightTop, rightBottom, 76, ink)
	drawLine(img, point{300, 512}, point{724, 512}, 76, ink)

	for _, node := range []point{leftTop, leftBottom, rightTop, rightBottom} {
		drawCircle(img, node, 78, mint)
		drawCircle(img, node, 31, background)
	}
	drawCircle(img, center, 92, coral)
	drawCircle(img, center, 38, ink)
}

type point struct {
	x float64
	y float64
}

func drawRoundedRect(img *image.RGBA, left, top, right, bottom, radius float64, fill color.RGBA) {
	for y := int(top * scale); y < int(bottom*scale); y++ {
		for x := int(left * scale); x < int(right*scale); x++ {
			px := float64(x) / scale
			py := float64(y) / scale
			cx := math.Max(left+radius, math.Min(px, right-radius))
			cy := math.Max(top+radius, math.Min(py, bottom-radius))
			if math.Hypot(px-cx, py-cy) <= radius {
				img.SetRGBA(x, y, fill)
			}
		}
	}
}

func drawLine(img *image.RGBA, from, to point, width float64, fill color.RGBA) {
	minX := int((math.Min(from.x, to.x) - width) * scale)
	maxX := int((math.Max(from.x, to.x) + width) * scale)
	minY := int((math.Min(from.y, to.y) - width) * scale)
	maxY := int((math.Max(from.y, to.y) + width) * scale)
	lengthSquared := math.Pow(to.x-from.x, 2) + math.Pow(to.y-from.y, 2)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := point{float64(x) / scale, float64(y) / scale}
			t := ((p.x-from.x)*(to.x-from.x) + (p.y-from.y)*(to.y-from.y)) / lengthSquared
			t = math.Max(0, math.Min(1, t))
			nearest := point{from.x + t*(to.x-from.x), from.y + t*(to.y-from.y)}
			if math.Hypot(p.x-nearest.x, p.y-nearest.y) <= width/2 {
				img.SetRGBA(x, y, fill)
			}
		}
	}
}

func drawCircle(img *image.RGBA, center point, radius float64, fill color.RGBA) {
	minX := int((center.x - radius) * scale)
	maxX := int((center.x + radius) * scale)
	minY := int((center.y - radius) * scale)
	maxY := int((center.y + radius) * scale)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if math.Hypot(float64(x)/scale-center.x, float64(y)/scale-center.y) <= radius {
				img.SetRGBA(x, y, fill)
			}
		}
	}
}
