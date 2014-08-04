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

// This code was originally from here: https://github.com/funkygao/golib/blob/master/observer/observer.go
// I wanted to "vendor" it in a sense, but there may also be need to adjust it.
// Additionally, I wanted to slip it into a package within Social Harvest and it made sense to give the harvester
// an observer since everything uses the harvester.

package harvester

import (
	"errors"
	"sync"
	"time"
)

const E_NOT_FOUND = "Event Not Found"

var (
	events  = make(map[string][]chan interface{})
	rwMutex sync.RWMutex
)

func Subscribe(event string, outputChan chan interface{}) {
	rwMutex.Lock()
	defer rwMutex.Unlock()

	events[event] = append(events[event], outputChan)
}

// Stop observing the specified event on the provided output channel
func UnSubscribe(event string, outputChan chan interface{}) error {
	rwMutex.Lock()
	defer rwMutex.Unlock()

	newArray := make([]chan interface{}, 0)
	outChans, ok := events[event]
	if !ok {
		return errors.New(E_NOT_FOUND)
	}
	for _, ch := range outChans {
		if ch != outputChan {
			newArray = append(newArray, ch)
		} else {
			close(ch)
		}
	}

	events[event] = newArray

	return nil
}

// Stop observing the specified event on all channels
func UnSubscribeAll(event string) error {
	rwMutex.Lock()
	defer rwMutex.Unlock()

	outChans, ok := events[event]
	if !ok {
		return errors.New(E_NOT_FOUND)
	}

	for _, ch := range outChans {
		close(ch)
	}
	delete(events, event)

	return nil
}

func Publish(event string, data interface{}) error {
	rwMutex.RLock()
	defer rwMutex.RUnlock()

	outChans, ok := events[event]
	if !ok {
		return errors.New(E_NOT_FOUND)
	}

	// notify all through chan
	for _, outputChan := range outChans {
		outputChan <- data
	}

	return nil
}

func PublishTimeout(event string, data interface{}, timeout time.Duration) error {
	rwMutex.RLock()
	defer rwMutex.RUnlock()

	outChans, ok := events[event]
	if !ok {
		return errors.New(E_NOT_FOUND)
	}

	for _, outputChan := range outChans {
		select {
		case outputChan <- data:
		case <-time.After(timeout):
		}
	}

	return nil
}
