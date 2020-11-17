package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	docker "docker.io/go-docker"
)

type Image struct {
	Name   string `csv:"name,emitempty"`
	Tag    string `csv:"tag,emitempty"`
	Digest string `csv:"digest,emitempty"`
}

func (i Image) getArr() []string {
	return []string{
		i.Name,
		i.Tag,
		i.Digest,
	}
}

func (i *Image) setDigest(digest string) {
	i.Digest = digest
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("usage %s {CSV File name}\n", os.Args[0])
		os.Exit(1)
	}

	fileName := os.Args[1]

	f, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	reader := csv.NewReader(f)
	lines, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = f.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	images := make([]*Image, 0, len(lines))
	for _, line := range lines {
		image := &Image{
			Name: line[0],
			Tag:  line[1],
		}
		if len(line) > 2 {
			image.Digest = line[2]
		}
		images = append(images, image)
	}

	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	changed := false
	for _, image := range images {
		fullName := fmt.Sprintf("%s:%s", image.Name, image.Tag)
		fmt.Printf("Reading digest for %s\n", fullName)
		inspect, err := cli.DistributionInspect(context.Background(), fullName, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		digest := inspect.Descriptor.Digest.Hex()
		if image.Digest != digest {
			changed = true
			fmt.Printf("New digest for %s - %s\n", fullName, digest)
			image.setDigest(digest)
		}
	}

	if changed {
		fmt.Println("Found new digests. Updating the file")
		f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()

		writer := csv.NewWriter(f)

		lines := make([][]string, 0, len(images))
		for _, image := range images {
			lines = append(lines, image.getArr())
		}

		err = writer.WriteAll(lines)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("The images file is up to date")
	}

}
