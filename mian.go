package main

import (
	"errors"
	"fmt"
	"github.com/lukeroth/gdal"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
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
	fmt.Println("start open", time.Now())
	ds, err := gdal.Open(infile, gdal.ReadOnly)
	if err != nil {
		log.Fatal("Open Infile Error")
		return err
	}
	fmt.Println("end open", time.Now())
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

	fmt.Println("band num", num)
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

		fmt.Println("start read", time.Now())
		redBuffer := make([]int32, 100*100)
		red.IO(gdal.Read, 0, 0, xSize, ySize, redBuffer, 100, 100, 0, 0)
		greenBuffer := make([]int32, 100*100)
		green.IO(gdal.Read, 0, 0, xSize, ySize, greenBuffer, 100, 100, 0, 0)
		blueBuffer := make([]int32, 100*100)
		blue.IO(gdal.Read, 0, 0, xSize, ySize, blueBuffer, 100, 100, 0, 0)
		fmt.Println("end read", time.Now())
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

func ReadShapefile(infile string) error {
	driver := gdal.OGRDriverByName("ESRI Shapefile")
	ds, ok := driver.Open(infile, 0)
	if !ok {
		log.Fatal("Open Shapefile Error")
		return errors.New("Open Shapefile Error")
	}
	defer ds.Destroy()

	layer := ds.LayerByIndex(0)

	for {
		if feature := layer.NextFeature(); feature != nil {
			fmt.Println("FID: ", feature.FID())
			df := feature.Definition()
			for i := 0; i < df.FieldCount(); i++ {
				attr := df.FieldDefinition(i).Name() // 属性字段名称
				switch df.FieldDefinition(i).Type() {
				case gdal.FT_String:
					fmt.Printf("%s : %s", attr, feature.FieldAsString(i))
				case gdal.FT_Integer:
					fmt.Printf("%s : %d", attr, feature.FieldAsInteger(i))
				case gdal.FT_Real:
					fmt.Printf("%s : %f", attr, feature.FieldAsFloat64(i))
				}
			}

			geometry, _ := feature.Geometry().ToWKT()
			fmt.Println("Geometry: ", geometry)
		} else {
			break
		}
	}

	return nil
}

func WriteShapefile(outfile string) error {
	driver := gdal.OGRDriverByName("ESRI Shapefile")

	ds, ok := driver.Create(outfile, nil)
	if !ok {
		log.Fatal("Create Shapefile Error")
		return errors.New("Create Shapefile Error")
	}
	defer ds.Destroy()
	ref := gdal.CreateSpatialReference("")
	ref.FromEPSG(4326)

	layer := ds.CreateLayer("", ref, gdal.GT_Unknown, nil)

	// fileds := []string{"OBJECTID", "YBBQ"}

	ObjectID := gdal.CreateFieldDefinition("OBJECTID", gdal.FT_Integer64)
	layer.CreateField(ObjectID, false)

	YBBQ := gdal.CreateFieldDefinition("YBBQ", gdal.FT_String)
	YBBQ.SetWidth(64)
	layer.CreateField(YBBQ, false)

	for i := 0; i < 100; i++ {
		feature := layer.Definition().Create()
		feature.SetFID(int64(i))
		feature.SetFieldInteger64(0, int64(i*i))
		feature.SetFieldString(1, "小麦")
		geometry, _ := gdal.CreateFromWKT("POLYGON((119.226 35.7242,119.607 35.6541,119.521 35.343,119.141 35.413,119.226 35.7242))", ref)
		feature.SetGeometry(geometry)
		layer.Create(feature)
		feature.Destroy()
	}

	return nil

}

func main() {
	// todo...
}
