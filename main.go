package main

// This tool maintain the deploy/images.csv and the deploy/images.env file, to be used to generate the CSV

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"sync"

	docker "docker.io/go-docker"
)

const (
	csvFile = "deploy/images.csv"
	envFile = "deploy/images.env"
)

type Image struct {
	EnvVar string
	Name   string
	Tag    string
	Digest string
}

func (i Image) getArr() []string {
	return []string{
		i.EnvVar,
		i.Name,
		i.Tag,
		i.Digest,
	}
}

func NewImage(fields []string) *Image {
	image := &Image{
		EnvVar: fields[0],
		Name:   fields[1],
		Tag:    fields[2],
	}

	if len(fields) > 3 {
		image.Digest = fields[3]
	}

	return image
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
		images = append(images, NewImage(line))
	}

	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(images) - 1) // the first "image" is the CSV title

	type message struct {
		index    int
		digest   string
		fullName string
	}

	ch := make(chan message, len(images) - 1)

	go func() {
		wg.Wait()
		close(ch)
	}()

	for i, image := range images[1:] {
		go func(image *Image, index int) {
			fullName := fmt.Sprintf("%s:%s", image.Name, os.Getenv(image.Tag))
			fmt.Printf("Reading digest for %s\n", fullName)
			inspect, err := cli.DistributionInspect(context.Background(), fullName, "")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			digest := inspect.Descriptor.Digest.Hex()
			ch <- message{index: index, digest: digest, fullName: fullName}
			wg.Done()
		}(image, i + 1)

	}

	changed := false
	for msg := range ch {
		if images[msg.index].Digest != msg.digest {
			changed = true
			fmt.Printf("New digest for %s - %s\n", msg.fullName, msg.digest)
			images[msg.index].setDigest(msg.digest)
		}
	}

	if changed {
		fmt.Println("Found new digests. Updating the file")
		if err = writeCsv(images); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("The images file is up to date")
	}

	if err = writeEnvFile(images); err != nil {
		fmt.Println(err)
		os.Exit(1)
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
	imageList := make([]string, len(images)-1, len(images)-1)
	for i, image := range images[1:] {
		imageDigest := fmt.Sprintf("%s@sha256:%s", image.Name, image.Digest)
		_, err = writer.WriteString(fmt.Sprintf("%s=%s\n", image.EnvVar, imageDigest))
		if err != nil {
			return err
		}
		imageList[i] = imageDigest
	}
	_, err = writer.WriteString(fmt.Sprintf("DIGEST_LIST=%s\n", strings.Join(imageList, ",")))
	if err != nil {
		return err
	}

	return writer.Flush()
}
