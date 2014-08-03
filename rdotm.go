// Copyright (c) 2014 Soichiro Kashima
// Licensed under MIT license.

package main

import (
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	OutputHeader = `// DO NOT EDIT.
// This file is automatically generated by rdotm tool.
// https://github.com/ksoichiro/rdotm

`
)

// Command line options
type Options struct {
	ResDir string
	OutDir string
	Class  string
	Clean  bool
}

// Resource model structure
type Resources struct {
	Strings []String `xml:"string"`
	Colors  []Color  `xml:"color"`
}

type String struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type Color struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

func main() {
	// Get command line options
	var (
		resDir = flag.String("res", "", "Resource(res) directory path. Required.")
		outDir = flag.String("out", "", "Output directory path. Required.")
		class  = flag.String("class", "R", "Class name to overwrite default value(R). Optional.")
		clean  = flag.Bool("clean", false, "Clean output directory before execution.")
	)
	flag.Parse()
	if *resDir == "" || *outDir == "" {
		// Exit if the required options are empty
		flag.Usage()
		os.Exit(1)
	}

	// Parse resource XML files and generate source code
	parse(&Options{
		ResDir: *resDir,
		OutDir: *outDir,
		Class:  *class,
		Clean:  *clean})
}

func parse(opt *Options) {
	// Parse all of the files in res/values/*.xml
	valuesDir := filepath.Join(opt.ResDir, "values")
	files, _ := ioutil.ReadDir(valuesDir)
	var res Resources
	for i := range files {
		entry := files[i]
		if matched, _ := regexp.MatchString(".xml$", entry.Name()); !matched {
			continue
		}
		entryPath := filepath.Join(valuesDir, entry.Name())
		r := parseXml(entryPath)
		if 0 < len(r.Strings) {
			res.Strings = append(res.Strings, r.Strings...)
		}
		if 0 < len(r.Colors) {
			res.Colors = append(res.Colors, r.Colors...)
		}
	}
	printAsObjectiveC(&res, opt)
}

func parseXml(filename string) (res Resources) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file", err)
		return
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	err = xml.Unmarshal(b, &res)
	if err != nil {
		fmt.Println("Error unmarshaling XML file", err)
		return
	}

	return res
}

func printAsObjectiveC(res *Resources, opt *Options) {
	// Create output directory
	if opt.Clean {
		// Discard all files in the output directory
		os.RemoveAll(opt.OutDir)
	}
	os.MkdirAll(opt.OutDir, 0777)

	class := opt.Class

	// Print header file(.h)
	filename := filepath.Join(opt.OutDir, class+".h")
	f, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()

	f.WriteString(OutputHeader)
	f.WriteString(fmt.Sprintf(`#import <UIKit/UIKit.h>

@interface %s : NSObject

`, class))

	// String
	for i := range res.Strings {
		s := res.Strings[i]
		// Method definition
		f.WriteString(fmt.Sprintf("+ (NSString *)string_%s;\n", s.Name))
	}

	// Color
	for i := range res.Colors {
		s := res.Colors[i]
		// Method definition
		f.WriteString(fmt.Sprintf("+ (UIColor *)color_%s;\n", s.Name))
	}

	f.WriteString(`
@end
`)
	f.Close()

	// Print implementation file(.m)
	filename = filepath.Join(opt.OutDir, class+".m")
	f, _ = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()

	// Import header file
	f.WriteString(OutputHeader)
	f.WriteString(fmt.Sprintf(`#import "%s.h"

@implementation %s

`, class, class))

	// String
	for i := range res.Strings {
		s := res.Strings[i]
		// Method implementation
		f.WriteString(fmt.Sprintf("+ (NSString *)string_%s { return @\"%s\"; }\n", s.Name, s.Value))
	}

	// Color
	for i := range res.Colors {
		s := res.Colors[i]
		// Method implementation
		a, r, g, b := hexToInt(s.Value)
		f.WriteString(fmt.Sprintf("+ (UIColor *)color_%s { return [UIColor colorWithRed:%d/255.0 green:%d/255.0 blue:%d/255.0 alpha:%d/255.0]; }\n", s.Name, r, g, b, a))
	}

	f.WriteString(`
@end
`)
	f.Close()
}

func hexToInt(hexString string) (a, r, g, b int) {
	raw := hexString
	// Remove prefix '#'
	if strings.HasPrefix(raw, "#") {
		braw := []byte(raw)
		raw = string(braw[1:])
	}

	// Format hex string
	if len(raw) == 8 {
		// AARRGGBB: Do nothing
	} else if len(raw) == 6 {
		// RRGGBB: Insert alpha(FF)
		raw = "FF" + raw
	} else if len(raw) == 4 {
		// ARGB: Duplicate each hex
		braw := []byte(raw)
		sa := string(braw[0:1])
		sr := string(braw[1:2])
		sg := string(braw[2:3])
		sb := string(braw[3:4])
		raw = sa + sa + sr + sr + sg + sg + sb + sb
		fmt.Printf("ARGB: %s", raw)
	} else if len(raw) == 3 {
		// RGB: Insert alpha(F) and duplicate each hex
		raw = "F" + raw
		braw := []byte(raw)
		sa := string(braw[0:1])
		sr := string(braw[1:2])
		sg := string(braw[2:3])
		sb := string(braw[3:4])
		raw = sa + sa + sr + sr + sg + sg + sb + sb
		fmt.Printf("RGB: %s", raw)
	}
	bytes, _ := hex.DecodeString(raw)
	a = int(bytes[0])
	r = int(bytes[1])
	g = int(bytes[2])
	b = int(bytes[3])
	return
}
