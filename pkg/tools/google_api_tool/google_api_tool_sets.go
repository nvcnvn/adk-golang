// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googleapitool

import (
	"log"
	"sync"
)

var (
	// Pre-configured toolsets as variables to match Python's style
	// Each is lazily initialized with sync.Once to ensure thread safety

	calendarToolSetOnce sync.Once
	calendarToolSet     *GoogleApiToolSet

	bigqueryToolSetOnce sync.Once
	bigqueryToolSet     *GoogleApiToolSet

	gmailToolSetOnce sync.Once
	gmailToolSet     *GoogleApiToolSet

	youtubeToolSetOnce sync.Once
	youtubeToolSet     *GoogleApiToolSet

	slidesToolSetOnce sync.Once
	slidesToolSet     *GoogleApiToolSet

	sheetsToolSetOnce sync.Once
	sheetsToolSet     *GoogleApiToolSet

	docsToolSetOnce sync.Once
	docsToolSet     *GoogleApiToolSet
)

// GetCalendarToolSet returns a Google Calendar API toolset.
func GetCalendarToolSet() *GoogleApiToolSet {
	calendarToolSetOnce.Do(func() {
		var err error
		calendarToolSet, err = LoadToolSet("calendar", "v3")
		if err != nil {
			log.Printf("Failed to load Calendar toolset: %v", err)
		}
	})
	return calendarToolSet
}

// GetBigQueryToolSet returns a Google BigQuery API toolset.
func GetBigQueryToolSet() *GoogleApiToolSet {
	bigqueryToolSetOnce.Do(func() {
		var err error
		bigqueryToolSet, err = LoadToolSet("bigquery", "v2")
		if err != nil {
			log.Printf("Failed to load BigQuery toolset: %v", err)
		}
	})
	return bigqueryToolSet
}

// GetGmailToolSet returns a Google Gmail API toolset.
func GetGmailToolSet() *GoogleApiToolSet {
	gmailToolSetOnce.Do(func() {
		var err error
		gmailToolSet, err = LoadToolSet("gmail", "v1")
		if err != nil {
			log.Printf("Failed to load Gmail toolset: %v", err)
		}
	})
	return gmailToolSet
}

// GetYouTubeToolSet returns a Google YouTube API toolset.
func GetYouTubeToolSet() *GoogleApiToolSet {
	youtubeToolSetOnce.Do(func() {
		var err error
		youtubeToolSet, err = LoadToolSet("youtube", "v3")
		if err != nil {
			log.Printf("Failed to load YouTube toolset: %v", err)
		}
	})
	return youtubeToolSet
}

// GetSlidesToolSet returns a Google Slides API toolset.
func GetSlidesToolSet() *GoogleApiToolSet {
	slidesToolSetOnce.Do(func() {
		var err error
		slidesToolSet, err = LoadToolSet("slides", "v1")
		if err != nil {
			log.Printf("Failed to load Slides toolset: %v", err)
		}
	})
	return slidesToolSet
}

// GetSheetsToolSet returns a Google Sheets API toolset.
func GetSheetsToolSet() *GoogleApiToolSet {
	sheetsToolSetOnce.Do(func() {
		var err error
		sheetsToolSet, err = LoadToolSet("sheets", "v4")
		if err != nil {
			log.Printf("Failed to load Sheets toolset: %v", err)
		}
	})
	return sheetsToolSet
}

// GetDocsToolSet returns a Google Docs API toolset.
func GetDocsToolSet() *GoogleApiToolSet {
	docsToolSetOnce.Do(func() {
		var err error
		docsToolSet, err = LoadToolSet("docs", "v1")
		if err != nil {
			log.Printf("Failed to load Docs toolset: %v", err)
		}
	})
	return docsToolSet
}
