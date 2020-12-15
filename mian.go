package main

import (
	"github.com/lukeroth/gdal"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// 栅格数据转WGS84
func Raster2WGS84(infile, outfile string) error {
	sds, err := gdal.Open(infile, gdal.ReadOnly)
	if err != nil {
		log.Fatal("Open Infile Error")
		return err
	}
	defer sds.Close()

	projection := gdal.CreateSpatialReference(sds.Projection())
	space, _ := projection.AttrValue("AUTHORITY", 1)
	if !strings.EqualFold(space, "4326") {
		ref := gdal.CreateSpatialReference("")
		ref.FromEPSG(4326)

		wgs84, _ := ref.ToWKT()
		log.Println(wgs84)

		vds, err := sds.AutoCreateWarpedVRT(sds.Projection(), wgs84, gdal.GRA_Bilinear)
		if err != nil {
			log.Fatal("AutoCreateWarpedVRT Error")
			return err
		}
		defer vds.Close()

		driver, err := gdal.GetDriverByName("GTiff")
		if err != nil {
			log.Fatal("GetDriverByName GTiff Error")
			return err
		}

		dds := driver.CreateCopy(outfile, vds, 1, nil, gdal.TermProgress, nil)

		defer dds.Close()
	}

	return nil
}

// 矢量数据转WGS84
func Vector2WGS84(infile, outfile string) error {
	b, err := ioutil.ReadFile(strings.Replace(infile, ".shp", ".prj", 1))
	if err != nil {
		log.Fatal("Open Prj Error")
		return err
	}

	projection := gdal.CreateSpatialReference(string(b))
	space, _ := projection.AttrValue("AUTHORITY", 1)
	if !strings.EqualFold(space, "4326") {
		os.Setenv("SHAPE_ENCODING", "UTF-8")

		sds, err := gdal.OpenEx(infile, gdal.OFVector, nil, nil, nil)
		if err != nil {
			log.Fatal("Open Infile Error")
			return err
		}
		defer sds.Close()

		dds, err := gdal.VectorTranslate(outfile, []gdal.Dataset{sds}, []string{"-t_srs", "epsg:4326"})
		if err != nil {
			log.Fatal("VectorTranslate Error")
			return err
		}

		defer dds.Close()
	}

	return nil
}

// 生成影像的缩略图
func Thumb(infile, outimg string) error {
	ds, err := gdal.Open(infile, gdal.ReadOnly)
	if err != nil {
		log.Fatal("Open Infile Error")
		return err
	}

	defer ds.Close()

	f, err := os.Create(outimg)
	if err != nil {
		log.Fatal("os.Create Error")
		return nil
	}
	defer f.Close()

	num := ds.RasterCount() // 波段数
	xSize := ds.RasterXSize()
	ySize := ds.RasterYSize()

	if num < 3 { // 单波段
		img := image.NewGray(image.Rect(0, 0, 100, 100))

		band := ds.RasterBand(1)
		min, max := band.ComputeMinMax(2)

		buffer := make([]int32, 100*100)
		band.IO(gdal.Read, 0, 0, xSize, ySize, buffer, 100, 100, 0, 0)

		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				g := buffer[y*100+x]
				t := (float64(g) - min) / (max - min) * 255
				g = int32(t)

				img.SetGray(x, y, color.Gray{Y: uint8(g)})
			}
		}

		if err := png.Encode(f, img); err != nil {
			log.Fatal("png.Encode Error")
			return err
		}

	} else { // 多波段
		// 默认生成 长 宽 为100的图片
		img := image.NewNRGBA(image.Rect(0, 0, 100, 100))

		// 红
		red := ds.RasterBand(1)
		redMin, redMax := red.ComputeMinMax(2)

		// 绿
		green := ds.RasterBand(2)
		greenMin, greenMax := green.ComputeMinMax(2)

		// 蓝
		blue := ds.RasterBand(3)
		blueMin, blueMax := blue.ComputeMinMax(2)

		redBuffer := make([]int32, 100*100)
		red.IO(gdal.Read, 0, 0, xSize, ySize, redBuffer, 100, 100, 0, 0)
		greenBuffer := make([]int32, 100*100)
		green.IO(gdal.Read, 0, 0, xSize, ySize, greenBuffer, 100, 100, 0, 0)
		blueBuffer := make([]int32, 100*100)
		blue.IO(gdal.Read, 0, 0, xSize, ySize, blueBuffer, 100, 100, 0, 0)

		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				r := redBuffer[y*100+x]
				t := (float64(r) - redMin) / (redMax - redMin) * 255
				r = int32(t)
				g := greenBuffer[y*100+x]
				t = (float64(g) - greenMin) / (greenMax - greenMin) * 255
				g = int32(t)
				b := blueBuffer[y*100+x]
				t = (float64(b) - blueMin) / (blueMax - blueMin) * 255
				b = int32(t)

				img.Set(x, y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
			}
		}

		if err := png.Encode(f, img); err != nil {
			log.Fatal("png.Encode Error")
			return err
		}
	}

	return nil
}

func main() {
	//...
}
