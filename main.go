package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/twpayne/go-kml"
)

type geohunt struct {
	uuid string
	name string
}

type point struct {
	latitude  float64
	longitude float64
}

type waypoint struct {
	name      string
	latitude  float64
	longitude float64
}

type findpoint struct {
	name      string
	latitude  float64
	longitude float64
}

var db *sql.DB
var filename = ""
var input string
var output string

func main() {

	flag.StringVar(&input, "i", "", "Input ShareData file (default location: C:\\Users\\[username]\\AppData\\Roaming\\XChange2\\Share\\ShareData)")
	flag.StringVar(&output, "o", "", "Output directory for KML-file")

	flag.Parse()

	if input == "" {
		logrus.Errorln("Please specify input ShareData file with option -i")
		return
	}

	if output == "" {
		logrus.Errorln("Please specify output directory with option -o")
		return
	}

	getFilename()

	logrus.Println("Input:", input)
	logrus.Printf("Output: %s\n\n", filepath.Join(output, filename))
	makeKML()
}

func getFilename() {
	counter := int64(0)
	files, _ := filepath.Glob(filepath.Join(output, "xctransfer-*.kml"))
	for _, f := range files {
		sn := strings.Replace(strings.Replace(f, filepath.Join(output, "xctransfer-"), "", 1), ".kml", "", 1)
		c, err := strconv.ParseInt(sn, 10, 64)
		if err != nil {
			c = 0
		}
		if c > counter {
			counter = c
		}
	}
	filename = fmt.Sprintf("xctransfer-%d.kml", counter+1)
}

func makeKML() {
	var err error
	logrus.Println("Make KML...\n\n")
	db, err = sql.Open("sqlite3", input)
	if err != nil {
		logrus.Fatal(err)
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		logrus.Fatal(err)
	}

	ways := []*kml.CompoundElement{}
	waypoints := []*kml.CompoundElement{}
	findpoints := []*kml.CompoundElement{}

	for _, v := range getHunts() {
		points := getPoints(v.uuid)
		way := generateWay(v.name, points)
		ways = append(ways, way)
	}

	for _, v := range getWayPoints() {
		wp := generateWaypoint(v)
		waypoints = append(waypoints, wp)
	}

	for _, v := range getFindPoints() {
		fp := generateFindpoint(v)
		findpoints = append(findpoints, fp)
	}

	generateKML(ways, waypoints, findpoints)
}

func generateWay(name string, points []point) *kml.CompoundElement {
	logrus.Println("Way generation: " + name)
	kcs := []kml.Coordinate{}

	for _, v := range points {
		kcs = append(kcs, kml.Coordinate{
			Lon: v.longitude,
			Lat: v.latitude,
		})
	}

	retval := kml.Placemark(
		kml.Name(name),
		kml.StyleURL("#orangeLineGreenPoly"),
		kml.LineString(
			kml.Coordinates(kcs...),
		),
	)

	return retval
}

func generateWaypoint(wp waypoint) *kml.CompoundElement {
	logrus.Println("Waypoint generation: " + wp.name)
	kcs := kml.Coordinate{
		Lon: wp.longitude,
		Lat: wp.latitude,
	}

	retval := kml.Placemark(
		kml.Name(wp.name),
		kml.StyleURL("#wayPoint"),
		kml.Point(
			kml.Coordinates(kcs),
		),
	)

	return retval
}

func generateFindpoint(fp findpoint) *kml.CompoundElement {
	logrus.Println("Findpoint generation: " + fp.name)
	kcs := kml.Coordinate{
		Lon: fp.longitude,
		Lat: fp.latitude,
	}

	retval := kml.Placemark(
		kml.Name(fp.name),
		kml.StyleURL("#findPoint"),
		kml.Point(
			kml.Coordinates(kcs),
		),
	)

	return retval
}

func generateKML(ways []*kml.CompoundElement, waypoints []*kml.CompoundElement, findpoints []*kml.CompoundElement) {
	document := kml.Document(
		kml.SharedStyle(
			"orangeLineGreenPoly",
			kml.LineStyle(
				kml.Color(color.RGBA{R: 237, G: 100, B: 0, A: 255}),
				kml.Width(4),
			),
			kml.PolyStyle(
				kml.Color(color.RGBA{R: 0, G: 255, B: 0, A: 127}),
			),
		),
		kml.SharedStyle(
			"findPoint",
			kml.IconStyle(
				kml.Color(color.RGBA{R: 255, G: 0, B: 0, A: 255}),
			),
		),
		kml.SharedStyle(
			"wayPoint",
			kml.IconStyle(
				kml.Color(color.RGBA{R: 255, G: 163, B: 0, A: 255}),
			),
		),
	)

	for _, v := range ways {
		v := v
		document = document.Add(v)
	}

	for _, v := range waypoints {
		v := v
		document = document.Add(v)
	}

	for _, v := range findpoints {
		v := v
		document = document.Add(v)
	}

	k := kml.KML(
		document,
	)

	f, err := os.Create(filepath.Join(output, filename))

	if err != nil {
		logrus.Fatal(err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	if err := k.WriteIndent(w, "", "  "); err != nil {
		logrus.Fatal(err)
	}

	logrus.Println()
	logrus.Println("KML file saved in:", filepath.Join(output, filename))

}

func getWayPoints() []waypoint {
	rows, err := db.Query("select name, latitude, longitude from waypoint")
	if err != nil {
		logrus.Fatal(err)
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	retval := []waypoint{}

	for rows.Next() {
		g := waypoint{}
		err := rows.Scan(&g.name, &g.latitude, &g.longitude)
		if err != nil {
			logrus.Fatal(err)
		}
		retval = append(retval, g)

	}
	return retval
}

func getFindPoints() []findpoint {
	rows, err := db.Query("select name, latitude, longitude from findpoint")
	if err != nil {
		logrus.Fatal(err)
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	retval := []findpoint{}

	for rows.Next() {
		g := findpoint{}
		err := rows.Scan(&g.name, &g.latitude, &g.longitude)
		if err != nil {
			logrus.Fatal(err)
		}
		retval = append(retval, g)

	}
	return retval
}

func getPoints(gh string) []point {
	rows, err := db.Query("select latitude, longitude from point where geohunt_fk='" + gh + "'")
	if err != nil {
		logrus.Fatal(err)
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	retval := []point{}

	for rows.Next() {
		g := point{}
		err := rows.Scan(&g.latitude, &g.longitude)
		if err != nil {
			logrus.Fatal(err)
		}
		retval = append(retval, g)

	}
	return retval
}

func getHunts() []geohunt {
	rows, err := db.Query("select uuid, name from geohunt")
	if err != nil {
		logrus.Fatal(err)
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	retval := []geohunt{}

	for rows.Next() {
		g := geohunt{}
		err := rows.Scan(&g.uuid, &g.name)
		if err != nil {
			logrus.Fatal(err)
		}
		retval = append(retval, g)

	}
	return retval
}
