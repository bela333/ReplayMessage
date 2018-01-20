package main

import (
	"archive/zip"
	"io"
	"errors"
	"github.com/bela333/replayReader"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
)

type Metadata struct {
	MinecraftVersion string `json:"mcversion"`
}

type Marker struct {
	Timestamp int `json:"realTimestamp"`
}

func main() {

	//Open replay as ZIP file
	mcprFile, err := zip.OpenReader("replay.mcpr")
	if err != nil {
		panic(err)
	}


	var tmcprFile io.ReadCloser
	var markersFile io.ReadCloser
	var metadataFile io.ReadCloser

	//Find required files
	for _, file := range mcprFile.File{
		var err error
		switch file.Name {
		case "recording.tmcpr":
			tmcprFile, err = file.Open()
			break
		case "markers.json":
			markersFile, err = file.Open()
			break
		case "metaData.json":
			metadataFile, err = file.Open()
			break
		}
		if err != nil {
			panic(err)
		}
	}
	if tmcprFile == nil || metadataFile == nil || markersFile == nil {
		panic(errors.New("couldn't find required files. File isn't a replay or it doesn't have any markers"))
	}

	//Determine minecraft version and Chat Message packet ID

	var packetID int8

	var metadata Metadata
	metadataContent, err := ioutil.ReadAll(metadataFile)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(metadataContent, &metadata)
	if err != nil {
		panic(err)
	}
	switch metadata.MinecraftVersion {
	case "1.12.2":
		packetID = 0x0F
		break
	default:
		panic(errors.New("unsupported Minecraft version"))
	}

	//Find markers
	var markers []Marker
	markersContent, err := ioutil.ReadAll(markersFile)
	err = json.Unmarshal(markersContent, &markers)
	if err != nil {
		panic(err)
	}

	if len(markers) < 1 {
		panic(errors.New("couldn't find any markers"))
	}

	var marker Marker
	//Find first marker
	marker.Timestamp = math.MaxInt32
	for _, m := range markers {
		if m.Timestamp < marker.Timestamp {
			marker = m
		}
	}



	//Open recording.tmcpr as replay
	replay := replayReader.NewReplay(tmcprFile)

	result := ""

	var packet replayReader.Packet
	for replay.Next(&packet) {
		//If the timestamp of the current packet is more than the marker's then we can be sure that the last result was before the marker. Exiting loop
		if packet.Time > marker.Timestamp {
			break
		}
		//First byte of the packet is always the packet type
		packetType, err := packet.ReadByte()
		if err != nil {
			panic(err)
		}
		if packetType == packetID {
			//Format of Chat Message event: http://wiki.vg/Protocol#Chat_Message_.28clientbound.29
			//Reading json
			chatData, _, err := packet.ReadString()
			if err != nil {
				panic(errors.New("invalid packet"))
			}
			//Reading "position"
			position, err := packet.ReadByte()
			if err != nil {
				panic(errors.New("invalid packet"))
			}
			//Make sure that it is a chat message
			if position == 0 {
				//Store result
				result = chatData
			}
		}
	}

	//Check for errors that might have come up during the reading of the packets
	err = replay.Error()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", result)

}