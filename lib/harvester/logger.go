// Social Harvest is a social media analytics platform.
//     Copyright (C) 2014 Tom Maiaroto, Shift8Creative, LLC (http://www.socialharvest.io)
//
//     This program is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     This program is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <http://www.gnu.org/licenses/>.

package harvester

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

// This file contains all the code necessary to log to multiple files at once in order to avoid file locks and allow concurreny in the logging.
// The upside is speed. The downside is multiple files (not a huge deal).
// So folders for "messages" and "hashtags" and "mentions" must be made.
// This code is a modified version of: http://openmymind.net/Concurrent-Friendly-Logging-With-Go/

// This may be configurable? It's a more technical configuration though. Maybe automatically set it based on things like number of territories and/or criteria within.
// We could then get a sense for the amount of volume that might be coming through and make a reasonable worker count here. Playing with this is going to greatly
// affect performance and efficiency.
const workerCount = 4

// This represents a channel for each series.
// TODO: Maybe keep this in sync with the series listed in config/series.go or maybe make it be passed to NewLogggers() (if the latter, we could turn this into its own package)
var logChannels = map[string]chan []byte{
	"messages":           make(chan []byte, 1024),
	"mentions":           make(chan []byte, 1024),
	"hashtags":           make(chan []byte, 1024),
	"shared_links":       make(chan []byte, 1024),
	"contributor_growth": make(chan []byte, 1024),
}
var logWorkers = map[string][]*Worker{}
var logRootDir string

// This also has an effect on performance and efficiency. It changes the buffer size (and ultimately the amount of data in each log file).
// One risk that occurs when making this larger is that should the server crash during the middle of things buffering, more data would be lost.
// If keeping this small then less data would be lost if something were to crash.
// TODO: Maybe also make this configurable. WAS: 32768 - but I saw a lot of over capacity warnings...
const capacity = 65536

type Worker struct {
	fileRoot string
	buffer   []byte
	position int
}

// Creates and configures new workers on each of the logging channels and sets the directory path to store the log files.
func NewLoggers(dir string) {
	// If not configured, this could be empty. We need a directory to write to. So just return if an empty string was passed.
	if dir == "" {
		return
	}
	logRootDir = dir
	// Get the workers for each channel.
	for k, _ := range logChannels {
		logWorkers[k] = make([]*Worker, workerCount)
		for i := 0; i < workerCount; i++ {
			logWorkers[k][i] = NewWorker(i, k)
			// Work it baby!
			go logWorkers[k][i].Work(logChannels[k])
		}
	}
}

// Converts the various things to JSON first before sending those bytes to Log()
func LogJson(message interface{}, channelName string) {
	// If NewLoggers() was not called, there would be no root directory and thus no where to write to and no workers. Just return.
	if logRootDir == "" {
		return
	}
	jsonMsg, err := json.Marshal(message)
	if err == nil {
		Log(jsonMsg, channelName)
	}
}

// Sends to the buffered channel to eventually flush to disk.
func Log(event []byte, channelName string) {
	// If NewLoggers() was not called, there would be no root directory and thus no where to write to and no workers. Just return.
	if logRootDir == "" {
		return
	}
	select {
	case logChannels[channelName] <- event:
	case <-time.After(5 * time.Second):
		return // do we need this? i feel like while memory stops climbing after a harvest...it isn't being released. would this force GC?
		// or would i need to close all the logChannels and open them back up again on next harvest?
		// throw away the message, so sad
	default:
		// need this? if there's a timeout?
	}
}

// Each worker gets an id and a series name which get combined for a file name and directory within the root directory defined in the Social Harvest configuration.
func NewWorker(id int, series string) (w *Worker) {
	os.Mkdir(logRootDir+"/"+series, 0755)
	return &Worker{
		//move the root path to some config or something
		fileRoot: logRootDir + "/" + series + "/" + strconv.Itoa(id) + "_",
		buffer:   make([]byte, capacity),
	}
}

// Assigns a worker to work on the given channel.
func (w *Worker) Work(channelName chan []byte) {
	for {
		event := <-channelName
		length := len(event)
		// we run with nginx's client_max_body_size set to 2K which makes this unlikely to happen, but, just in case...
		if length > capacity {
			log.Println("message received was too large")
			continue
		}
		if (length + w.position) > capacity {
			w.Save()
		}
		copy(w.buffer[w.position:], event)
		w.position += length
	}
}

// Writes the buffer to a temporary file that gets moved to it's final location. Logging is split up among multiple files by worker id and time.
func (w *Worker) Save() {
	if w.position == 0 {
		return
	}
	f, _ := ioutil.TempFile("", "logs_")
	f.Write(w.buffer[0:w.position])
	f.Close()
	os.Rename(f.Name(), w.fileRoot+strconv.FormatInt(time.Now().UnixNano(), 10)+".log")
	w.position = 0
}
