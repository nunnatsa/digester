package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"

	docker "docker.io/go-docker"
)

const (
	csvFile = "deploy/images.csv"
	envFile = "deploy/images.env"
)

type Image struct {
	EnvVar string `csv:"envVar,omitempty"`
	Name   string `csv:"name,omitempty"`
	Tag    string `csv:"tag,omitempty"`
	Digest string `csv:"digest,omitempty"`
}

func (i Image) getArr() []string {
	return []string{
		i.EnvVar,
		i.Name,
		i.Tag,
		i.Digest,
	}
}

func (i *Image) setDigest(digest string) {
	i.Digest = digest
}

func main() {
	fmt.Println("Checking image digests")
	f, err := os.Open(csvFile)
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
			EnvVar: line[0],
			Name:   line[1],
			Tag:    line[2],
		}
		if len(line) > 3 {
			image.Digest = line[3]
		}
		images = append(images, image)
	}

	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	changed := false
	for _, image := range images[1:] {
		fullName := fmt.Sprintf("%s:%s", image.Name, os.Getenv(image.Tag))
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
		if err = writeCsv(images); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err = writeEnvFile(images); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("The images file is up to date")
	}

}

func writeCsv(images []*Image) error {
	f, err := os.OpenFile(csvFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	lines := make([][]string, 0, len(images))
	for _, image := range images {
		lines = append(lines, image.getArr())
	}

	err = writer.WriteAll(lines)
	if err != nil {
		return err
	}
	return nil
}

func writeEnvFile(images []*Image) error {
	f, err := os.OpenFile(envFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	for _, image := range images[1:] {
		_, err = writer.WriteString(fmt.Sprintf("%s=%s@sha256:%s\n", image.EnvVar, image.Name, image.Digest))
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}
