package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MrSplidge/go-coutil"
)

// Mapping from Json to Go.
type Track struct {
	RenderedFile string `json:"rendered_file"`
	Title        string
	// Optional things that can override album metadata (nil if not present in the json metadata file)
	Composer  *string
	Artist    *string
	Genre     *string
	Date      *string
	Cover     *string
	Copyright *string
}

// Mapping from Json to Go.
type Album struct {
	Title     string
	Composer  string
	Artist    string
	Genre     string
	Date      string
	Cover     string
	Copyright string
	Tracks    []Track
}

// Mapping from Json to Go.
type Metadata struct {
	FfmpegPath       string   `json:"ffmpeg_path"`
	InputPath        string   `json:"input_path"`
	OutputPath       string   `json:"output_path"`
	OutputExtensions []string `json:"output_extensions"`
	Parallel         bool
	Albums           []Album
}

// Captures information about an asynchronous ffmpeg encoding activity.
type WorkItem struct {
	Task  string   // Description of the activity
	Args  []string // Command arguments for ffmpeg
	Error error    // A launch or ffmpeg error description
}

func main() {
	var numThreadsFlag = flag.Int("num-threads", runtime.NumCPU(), "The number of worker threads to use.")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "go-meta [-num-threads] <metadata.json>\n\n")
		flag.PrintDefaults()
		return
	}

	path := flag.Arg(0)

	// Get the last modified time of the metadata. This is used to ensure that encoding takes place even when there
	// is an encoded file more recent than the original rendered file.
	metadataModTime := time.Now()
	stat, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: Looking for %s: %s", path, err)
	} else {
		metadataModTime = stat.ModTime()
	}

	// Read the metadata.
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: Reading the metadata file: %s", err)
		panic(err)
	}

	// Unmarshal the Json metadata into a Go DOM.
	var metadata Metadata
	err = json.Unmarshal(bytes, &metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: Parsing the Json metadata: %s", err)
		return
	}

	// Create the main output folder, if not already present.
	err = os.MkdirAll(metadata.OutputPath, 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: Creating the output path %s: %s", metadata.OutputPath, err)
		return
	}

	var workItems []WorkItem

	// Collect WorkItem instances for all albums.
	for _, album := range metadata.Albums {
		albumWorkItems := processAlbum(metadata, album, metadataModTime)
		workItems = append(workItems, albumWorkItems...)
	}

	// Convert path separators to native type.
	ffmpegCommand := filepath.FromSlash(metadata.FfmpegPath)

	// Process the work items and report completion or errors.
	coutil.WorkPool(
		*numThreadsFlag,
		workItems,
		func(item WorkItem) WorkItem {
			cmd := exec.Command(ffmpegCommand, item.Args...)

			// Collect stderr into a string Builder.
			var stderrStringBuilder strings.Builder
			cmd.Stderr = &stderrStringBuilder

			err = cmd.Run()

			if err != nil {
				// Record launch error
				item.Error = fmt.Errorf("error: %s (file in use?): %s", item.Task, err)
			} else {
				// Check ffmpeg exit code. Record stderr text if we have a non-zero exit code.
				if cmd.ProcessState.ExitCode() != 0 {
					item.Error = fmt.Errorf("error: ffmpeg: %s", stderrStringBuilder.String())
				}
			}
			return item
		},
		func(item WorkItem) {
			if item.Error != nil {
				fmt.Fprintf(os.Stderr, "%s", item.Error)
			} else {
				fmt.Printf("%s\n", item.Task)
			}
		})
}

// Creates output directories for an album and returns a slice of WorkItem(s) that contain arguments for ffmpeg
// to perform encoding.
func processAlbum(metadata Metadata, album Album, metadataModTime time.Time) []WorkItem {
	// A slice that contains encoding work items.
	var workItems []WorkItem

	// Try to create a WorkItem for each track in the album.
	for trackIndex, track := range album.Tracks {
		trackNumber := trackIndex + 1

		// Get information about the original rendered file that we're going to encode.
		inputRenderedPath := filepath.Join(metadata.InputPath, track.RenderedFile+".wav")
		inputRenderedStat, err := os.Stat(inputRenderedPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Looking for %s: %s", inputRenderedPath, err)
			continue
		} else {
			if inputRenderedStat.IsDir() {
				fmt.Fprintf(os.Stderr, "error: %s is a directory (skipping)!", inputRenderedPath)
				continue
			}
		}

		// Loop over the file format extensions
		for _, extension := range metadata.OutputExtensions {

			// Construct a target output folder if one does not already exist.
			targetFolder := filepath.Join(metadata.OutputPath, extension, album.Artist, album.Title)
			err = os.MkdirAll(targetFolder, 0777)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: Creating directory %s: %s", targetFolder, err)
				continue
			}

			// Work out the name of the target file we're going to create.
			targetPath := filepath.Join(
				targetFolder,
				fmt.Sprintf("%s - %s - %02d %s [%s].%s",
					album.Artist, album.Title, trackNumber,
					track.Title, track.RenderedFile, extension))

			// Check whether an existing target file is more recent than (a) the metadata, and (b) the input rendered file. If so, it can be skipped.
			if targetStat, err := os.Stat(targetPath); err == nil {
				if targetStat.IsDir() {
					// Check whether the proposed target file already exists as a directory.
					fmt.Fprintf(os.Stderr, "error: %s is a directory (skipping)!", targetPath)
					continue
				}

				// Check whether the existing target file is more recent than the metadata or input rendered file.
				targetModTime := targetStat.ModTime()
				if targetModTime.After(inputRenderedStat.ModTime()) && targetModTime.After(metadataModTime) {
					fmt.Printf("Skipping %s\n", targetPath)
					continue
				}
			}

			// Construct arguments for ffmpeg.exe
			args := []string{"-loglevel", "quiet", "-y", "-i", inputRenderedPath}

			if len(album.Cover) > 0 {
				// No cover art for WAV.
				if extension != "wav" {
					args = append(args, "-i", filepath.FromSlash(override(album.Cover, track.Cover)), "-disposition:v", "attached_pic", "-metadata:s:v", "title=Album Cover", "-metadata:s:v", "comment=Cover (Front)")
				}
				// MP3-specific tags.
				if extension == "mp3" {
					args = append(args, "-map", "0:a", "-map", "1:v", "-id3v2_version", "3")
				}
			}

			// Direct audio stream copy for WAV.
			if extension == "wav" {
				args = append(args, "-acodec", "copy")
			}

			// Track metadata.
			args = append(args,
				"-metadata", "track="+fmt.Sprintf("%d", trackNumber),
				"-metadata", "title="+track.Title,
				"-metadata", "album="+album.Title,
				"-metadata", "genre="+override(album.Genre, track.Genre),
				"-metadata", "date="+override(album.Date, track.Date),
				"-metadata", "artist="+override(album.Artist, track.Artist),
				"-metadata", "album_artist="+override(album.Artist, track.Artist),
				"-metadata", "composer="+override(album.Composer, track.Composer),
				"-metadata", "comment="+override(album.Copyright, track.Copyright),
			)

			// Format-specific compression.
			switch extension {
			case "flac":
				args = append(args, "-compression_level", "12")
			case "mp3":
				args = append(args, "-compression_level", "0", "-abr", "1", "-b:a", "320k")
			case "ogg":
				args = append(args, "-q", "10")
			}

			args = append(args, targetPath)

			item := WorkItem{
				Task: fmt.Sprintf("%s to %s", inputRenderedPath, targetPath),
				Args: args,
			}

			workItems = append(workItems, item)
		}
	}

	return workItems
}

// Overrides a string [basic] with another one [override] if available.
func override(basic string, override *string) string {
	if override != nil {
		return *override
	} else {
		return basic
	}
}
