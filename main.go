package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/golang/freetype"

	"github.com/golang/freetype/truetype"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type glyphInfo struct {
	AdvanceWidth  int
	AdvanceHeight int
}

func main() {
	fontPath := flag.String("font-path", "", "The path to the TrueType font to use.")
	fontSize := flag.Int("font-size", 16, "The size, in points, of the font.")
	dpi := flag.Float64("dpi", 72, "The screen resolution to use, in dots per inch.")

	flag.Parse()

	if *fontPath == "" {
		log.Fatalln("Missing font-path flag.")
	}

	fontData, err := ioutil.ReadFile(*fontPath)
	if err != nil {
		panic(err)
	}

	context := freetype.NewContext()

	ttfFont, err := truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}
	fontFace := truetype.NewFace(ttfFont, &truetype.Options{
		Size:    float64(*fontSize),
		DPI:     *dpi,
		Hinting: font.HintingFull,
	})

	context.SetDPI(*dpi)
	context.SetFont(ttfFont)
	context.SetFontSize(float64(*fontSize))
	context.SetHinting(font.HintingFull)
	context.SetSrc(image.Black)

	log.Printf("Creating font atlas for '%s' (%s)", ttfFont.Name(truetype.NameIDFontFamily), ttfFont.Name(truetype.NameIDFontSubfamily))

	atlasImage := image.NewRGBA(image.Rect(0, 0, 512, 256))

	context.SetClip(atlasImage.Bounds())
	context.SetDst(atlasImage)

	startChar := '!'
	endChar := '|'
	charWidth := 32
	charHeight := 32
	rowSize := atlasImage.Bounds().Size().X / charWidth

	glyphBuf := truetype.GlyphBuf{}

	scale := context.PointToFixed(float64(*fontSize))
	point := freetype.Pt(0, *fontSize)
	glyphMetrics := map[rune]glyphInfo{}

	for i := 1; i <= int(endChar-startChar)+1; i++ {
		char := startChar + rune(i-1)

		fontIndex := ttfFont.Index(char)
		glyphBuf.Load(ttfFont, scale, fontIndex, font.HintingFull)

		advanceWidth := ttfFont.HMetric(scale, fontIndex).AdvanceWidth.Ceil()
		advanceHeight := ttfFont.VMetric(scale, fontIndex).AdvanceHeight.Ceil()

		glyphBounds, _, _ := fontFace.GlyphBounds(char)
		glyphWidth := glyphBounds.Max.X.Ceil()
		glyphHeight := advanceHeight

		glyphMetrics[char] = glyphInfo{
			AdvanceWidth:  advanceWidth,
			AdvanceHeight: advanceHeight,
		}

		startY := (point.Y + fontFace.Metrics().Descent).Floor()

		// draw.Draw(atlasImage, image.Rect(point.X.Floor(), startY-glyphHeight, point.X.Floor()+glyphWidth, startY), image.Black, image.ZP, draw.Over)
		log.Println(glyphWidth, glyphHeight, startY)

		context.DrawString(string(char), point)

		if i%rowSize == 0 {
			// new row
			point.X = 0
			point.Y += fixed.I(charHeight)
		} else {
			// new column
			point.X += fixed.I(charWidth)
		}
	}

	// log.Printf("%+v", glyphMetrics)
	glyphs := make([]int, 0)
	for k := range glyphMetrics {
		glyphs = append(glyphs, int(k))
	}
	sort.Ints(glyphs)
	for _, glyph := range glyphs {
		currentGlyphMetrics := glyphMetrics[rune(glyph)]
		fmt.Printf("{%d, CharMetrics{%d, %d}},\n", glyph, currentGlyphMetrics.AdvanceWidth, currentGlyphMetrics.AdvanceHeight)
	}

	outputFile, err := os.Create("atlas.png")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, atlasImage)
	if err != nil {
		panic(err)
	}
}
