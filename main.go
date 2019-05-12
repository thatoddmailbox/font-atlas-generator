package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/golang/freetype"

	"github.com/golang/freetype/truetype"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	outputFormatPNG = "png"
	outputFormatC   = "c"
)

type glyphInfo struct {
	AdvanceWidth  int
	AdvanceHeight int
}

func main() {
	fontPath := flag.String("font-path", "", "The path to the TrueType font to use.")
	fontSize := flag.Int("font-size", 16, "The size, in points, of the font.")
	dpi := flag.Float64("dpi", 72, "The screen resolution to use, in dots per inch.")
	outputFormat := flag.String("output-format", "png", "The output format. (either 'png' or 'c')")

	flag.Parse()

	if *fontPath == "" {
		log.Fatalln("Missing font-path flag.")
	}

	if *outputFormat != outputFormatPNG && *outputFormat != outputFormatC {
		log.Fatalf("Invalid output-format flag.")
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

	fontName := ttfFont.Name(truetype.NameIDFontFamily)

	log.Printf("Creating font atlas for '%s' (%s)", fontName, ttfFont.Name(truetype.NameIDFontSubfamily))

	atlasImage := image.NewRGBA(image.Rect(0, 0, 512, 256))

	context.SetClip(atlasImage.Bounds())
	context.SetDst(atlasImage)

	startChar := '!'
	endChar := '~'
	charWidth := 32
	charHeight := 32
	rowSize := atlasImage.Bounds().Size().X / charWidth

	glyphBuf := truetype.GlyphBuf{}

	scale := context.PointToFixed(float64(*fontSize))
	point := freetype.Pt(0, *fontSize)
	glyphMetrics := map[rune]glyphInfo{}
	outputData := ""
	indexData := ""
	currentIndex := 0

	if *outputFormat == outputFormatC {
		outputData += "\t.data = {\n"
	}

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
		log.Println(char, glyphBounds.Max.X.Ceil(), glyphBounds.Max.Y.Ceil())

		var characterImage draw.Image

		if *outputFormat == outputFormatC {
			characterImage = image.NewRGBA(image.Rect(0, 0, advanceWidth, startY))
			context.SetClip(characterImage.Bounds())
			context.SetDst(characterImage)
		}

		context.DrawString(string(char), point)

		if *outputFormat == outputFormatC {
			outputFile, err := os.Create("debug/" + strconv.Itoa(int(char)) + ".png")
			if err != nil {
				panic(err)
			}

			err = png.Encode(outputFile, characterImage)
			if err != nil {
				panic(err)
			}

			outputFile.Close()
		}

		if *outputFormat == outputFormatC {
			bounds := characterImage.Bounds()

			topSpace := 0
			leftSpace := 0
			bitmapWidth := bounds.Dx()
			bitmapHeight := bounds.Dy()

			// calculate top space
			for y := 0; y < bounds.Dy(); y++ {
				foundNotBlank := false
				for x := 0; x < bounds.Dx(); x++ {
					alpha := characterImage.At(x, y).(color.RGBA).A
					if alpha != 0 {
						foundNotBlank = true
						break
					}
				}

				if foundNotBlank {
					break
				} else {
					topSpace++
				}
			}

			// calculate bottom space
			for y := bounds.Dy() - 1; y >= 0; y-- {
				foundNotBlank := false
				for x := 0; x < bounds.Dx(); x++ {
					alpha := characterImage.At(x, y).(color.RGBA).A
					if alpha != 0 {
						foundNotBlank = true
						break
					}
				}

				if foundNotBlank {
					break
				} else {
					bitmapHeight--
				}
			}

			bitmapHeight -= topSpace

			// calculate left space
			for x := 0; x < bounds.Dx(); x++ {
				foundNotBlank := false
				for y := 0; y < bounds.Dy(); y++ {
					alpha := characterImage.At(x, y).(color.RGBA).A
					if alpha != 0 {
						foundNotBlank = true
						break
					}
				}

				if foundNotBlank {
					break
				} else {
					leftSpace++
				}
			}

			// calculate right space
			for x := bounds.Dx() - 1; x >= 0; x-- {
				foundNotBlank := false
				for y := 0; y < bounds.Dy(); y++ {
					alpha := characterImage.At(x, y).(color.RGBA).A
					if alpha != 0 {
						foundNotBlank = true
						break
					}
				}

				if foundNotBlank {
					break
				} else {
					bitmapWidth--
				}
			}

			bitmapWidth -= leftSpace

			charString := string(char)
			if charString == "\\" {
				charString = "backslash"
			}
			outputData += fmt.Sprintf("\t\t// character: %s\n", charString)

			if i != 1 {
				indexData += ", "
			}
			indexData += strconv.Itoa(currentIndex)

			if bitmapWidth > 0 {
				// get the bitmap
				outputData += fmt.Sprintf("\t\t%d, %d, %d, %d, %d, %d,\n", leftSpace, topSpace, bitmapWidth, bitmapHeight, advanceWidth, advanceHeight)
				currentIndex += 6

				for y := topSpace; y < topSpace+bitmapHeight; y++ {
					outputData += "\t\t"
					for x := leftSpace; x < leftSpace+bitmapWidth; x++ {
						color := characterImage.At(x, y).(color.RGBA)
						if x != leftSpace {
							outputData += ", "
						}
						outputData += fmt.Sprintf("0x%x", 255-color.A)
						currentIndex++
					}
					outputData += ",\n"
				}

				log.Printf("%d box: (%d, %d, %d, %d)", char, leftSpace, topSpace, bitmapWidth, bitmapHeight)
			} else {
				log.Printf("%d skip", char)
				outputData += fmt.Sprintf("\t\t0, 0, 0, 0, 0, 0,\n")
				currentIndex += 6
			}

			if i != int(endChar-startChar)+1 {
				outputData += "\n"
			}
		}

		if *outputFormat != outputFormatC {
			if i%rowSize == 0 {
				// new row
				point.X = 0
				point.Y += fixed.I(charHeight)
			} else {
				// new column
				point.X += fixed.I(charWidth)
			}
		}
	}

	// log.Printf("%+v", glyphMetrics)
	glyphs := make([]int, 0)
	for k := range glyphMetrics {
		glyphs = append(glyphs, int(k))
	}
	sort.Ints(glyphs)

	if *outputFormat == outputFormatPNG {
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
	} else if *outputFormat == outputFormatC {
		headerData := "#include <stdint.h>\n"
		headerData += "\n"
		headerData += "#include \"font/font.h\"\n"
		headerData += "\n"
		headerData += fmt.Sprintf("// font: %s\n", fontName)
		headerData += fmt.Sprintf("// size: %d\n", *fontSize)
		headerData += "const font_t font = {\n"
		headerData += "\t.indexes = {"
		headerData += indexData
		headerData += "},\n"

		outputData = headerData + outputData

		outputData += "\t}\n"
		outputData += "};\n"

		err = ioutil.WriteFile("atlas.h", []byte(outputData), 0777)
		if err != nil {
			panic(err)
		}
	}
}
