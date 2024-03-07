# About

The go-meta utility can be used to convert rendered WAV files to other formats (e.g. mp3, flac, ogg) with tags and cover art defined in a json metadata file. A hierarchy of folders and files is created to contain the converted and tagged versions of the WAV files. ffmpeg (available elsewhere) is used to perform the actual conversions.

# Building

* ```go get```
* ```go build```

# Usage

```
go-meta [-num-threads <n>] <metadata.json>

  -num-threads int
        The number of worker threads to use. (default 64)
```

# Description

There is an example metadata file, ```example.json```. The top of the file describes the location of various things, such as the ffmpeg executable, the location of the WAV files to be converted, the output location for the resulting files, and the types of conversion to perform.

```example.json``` begins with:

```
{
  // The path to the ffmpeg executable
  "ffmpeg_path": "c:/users/mrsplidge/bin/ffmpeg/bin/ffmpeg.exe",
  // The location of the WAV files to be converted
  "input_path": "h:/music/renders",
  // The output location for the folder hierarchy that will be created to contain the converted files
  "output_path": "h:/music/renders/albums",
  "output_extensions": [
    // Create .mp3 files
    "mp3",
    // Create .flac files
    "flac",
    // Create .ogg files
    "ogg"
  ],
  // Whether to perform many conversion operations simultaneously. See also the -num-threads argument to go-meta.
  "parallel": true,
```

There then follows an array of albums. Each album contains a variety of different tag values that will be applied to each track within the album. Most of these can be overridden at the track level, so that different artists or cover art are used for specific tracks.

```
 "albums": [
    {
      // The album title
      "title": "Fruit",
      // Composer and Artist information.
      "composer": "The Composer",
      "artist": "The Composer",
      // The album genre.
      "genre": "Electronic",
      // Album date.
      "date": "2023",
      // A cover art image file. This can be a path relative to the current directory, or an absolute path.
      "cover": "Art/Fruit.png",
      // Copyright.
      "copyright": "Copyright (c) 2023 The Composer",
```

Within each album, there is a list of tracks. One converted file is generated for each track. The track includes the name of the source WAV file and the track name, which can be different. Album-level tags, such as composer, artist, genre, date, cover, and copyright can be included with each track to override album-level settings.

```
      "tracks": [
        {
          // The name of the WAV file (do not include the .WAV extension)
          "rendered_file": "30",
          // The name of the track that is to be generated.
          "title": "Apple"
        },
        ...
      ]
```

The result of running go-meta with the example.json metadata would be as shown below. The ordering of the output will probably be different each time due to distribution of work across processor cores.

```
Processing 21 track(s)
h:\music\renders\32.wav to h:\music\renders\albums\ogg\The Composer\Fruit\The Composer - Fruit - 03 Banana [32].ogg
h:\music\renders\32.wav to h:\music\renders\albums\flac\The Composer\Fruit\The Composer - Fruit - 03 Banana [32].flac
h:\music\renders\31.wav to h:\music\renders\albums\ogg\The Composer\Fruit\The Composer - Fruit - 02 Pear [31].ogg
h:\music\renders\30.wav to h:\music\renders\albums\ogg\The Composer\Fruit\The Composer - Fruit - 01 Apple [30].ogg
h:\music\renders\35.wav to h:\music\renders\albums\ogg\The Composer\Directions\The Composer - Directions - 03 Up [35].ogg
h:\music\renders\35.wav to h:\music\renders\albums\ogg\The Composer\Directions\The Composer - Directions - 04 Down [35].ogg
h:\music\renders\34.wav to h:\music\renders\albums\ogg\The Composer\Directions\The Composer - Directions - 02 Right [34].ogg
h:\music\renders\33.wav to h:\music\renders\albums\ogg\The Composer\Directions\The Composer - Directions - 01 Left [33].ogg
h:\music\renders\32.wav to h:\music\renders\albums\mp3\The Composer\Fruit\The Composer - Fruit - 03 Banana [32].mp3
h:\music\renders\31.wav to h:\music\renders\albums\flac\The Composer\Fruit\The Composer - Fruit - 02 Pear [31].flac
h:\music\renders\30.wav to h:\music\renders\albums\flac\The Composer\Fruit\The Composer - Fruit - 01 Apple [30].flac
h:\music\renders\34.wav to h:\music\renders\albums\flac\The Composer\Directions\The Composer - Directions - 02 Right [34].flac
h:\music\renders\35.wav to h:\music\renders\albums\flac\The Composer\Directions\The Composer - Directions - 04 Down [35].flac
h:\music\renders\35.wav to h:\music\renders\albums\flac\The Composer\Directions\The Composer - Directions - 03 Up [35].flac
h:\music\renders\33.wav to h:\music\renders\albums\flac\The Composer\Directions\The Composer - Directions - 01 Left [33].flac
h:\music\renders\35.wav to h:\music\renders\albums\mp3\The Composer\Directions\The Composer - Directions - 04 Down [35].mp3
h:\music\renders\35.wav to h:\music\renders\albums\mp3\The Composer\Directions\The Composer - Directions - 03 Up [35].mp3
h:\music\renders\31.wav to h:\music\renders\albums\mp3\The Composer\Fruit\The Composer - Fruit - 02 Pear [31].mp3
h:\music\renders\33.wav to h:\music\renders\albums\mp3\The Composer\Directions\The Composer - Directions - 01 Left [33].mp3
h:\music\renders\34.wav to h:\music\renders\albums\mp3\The Composer\Directions\The Composer - Directions - 02 Right [34].mp3
h:\music\renders\30.wav to h:\music\renders\albums\mp3\The Composer\Fruit\The Composer - Fruit - 01 Apple [30].mp3
```

The folder hierarchy can be adjusted by changing the following in ```main.go```:

```
// Construct a target output folder if one does not already exist.
targetFolder := filepath.Join(metadata.OutputPath, extension, album.Artist, album.Title)
```

The file naming scheme can be adjusted by changing the following in ```main.go```:

```
// Work out the name of the target file we're going to create.
targetPath := filepath.Join(
  targetFolder,
  fmt.Sprintf("%s - %s - %02d %s [%s].%s",
    album.Artist, album.Title, trackNumber,
    track.Title, track.RenderedFile, extension))
```

# Licence

Please see the included BSD 3-clause LICENSE file.

This program makes use of the following third-party packages. Please refer to these projects for additional licensing information.

github.com/MrSplidge/go-coutil v0.0.0 (BSD 3-clause)