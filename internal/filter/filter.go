// Package filter provides keyword-based pre-filtering of job listings.
package filter

import (
	"strings"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// includeKeywords — any match causes a job to pass the filter (case-insensitive).
var includeKeywords = []string{
	"golang",
	" go ",
	"go backend",
	"backend engineer",
	"backend developer",
	"platform engineer",
	"microservices",
	"api engineer",
	"distributed systems",
	"grpc",
	"rest api",
	"software engineer backend",
	// broader backend titles that also make sense for the candidate profile
	"software engineer",
	"software developer",
	"server-side",
	"server side",
	"java engineer",
	"java developer",
	"rust engineer",
	"rust developer",
	"full-stack",
	"fullstack",
	"full stack",
}

// excludeKeywords — any match causes a job to be rejected (case-insensitive).
// These are checked against title+tags combined, so "react" rejects jobs whose
// title or tag stack is React-focused, even if another tag says "golang".
var excludeKeywords = []string{
	"frontend",
	"front-end",
	"front end",
	"react",
	"vue",
	"angular",
	"mobile",
	"ios",
	"android",
	"flutter",
	"data scientist",
	"machine learning",
	"devops only",
	"qa engineer",
	"manual tester",
	"ux designer",
	"ui designer",
	"graphic designer",
	"customer support",
	"sales",
	"marketing",
}

// acceptedLocations — partial, case-insensitive matches that are allowed.
var acceptedLocations = []string{
	"remote",
	"worldwide",
	"anywhere",
	"hybrid",
	"work from home",
	"wfh",
	// countries / regions
	"asia",
	"indonesia",
	"japan",
	"united states",
	"singapore",
	"europe",
	"global",
	"international",
	"apac",
	// Indonesian cities (JobStreet / Glints return these)
	"jakarta",
	"bandung",
	"tangerang",
	"bekasi",
	"surabaya",
	"yogyakarta",
	"bali",
	"depok",
	"bogor",
}

// IsRelevant returns true when a job passes the keyword filter rules.
// It checks title, tags, and description (combined) for include/exclude keywords
// and validates the location.
func IsRelevant(j store.Job) bool {
	if j.Title == "" {
		return false
	}

	// Build a combined lower-case search blob from title + tags
	haystack := strings.ToLower(j.Title + " " + strings.Join(j.Tags, " "))

	// Reject if any exclude keyword matches
	for _, kw := range excludeKeywords {
		if strings.Contains(haystack, kw) {
			return false
		}
	}

	// Require at least one include keyword
	matched := false
	for _, kw := range includeKeywords {
		if strings.Contains(haystack, kw) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	// Location filter: if a location is provided, at least one accepted term must match.
	// Empty location is allowed (assume remote).
	if j.Location != "" {
		loc := strings.ToLower(j.Location)
		// Reject explicit onsite outside Jakarta
		if strings.Contains(loc, "onsite") && !strings.Contains(loc, "jakarta") {
			return false
		}
		locationOK := false
		for _, accepted := range acceptedLocations {
			if strings.Contains(loc, accepted) {
				locationOK = true
				break
			}
		}
		if !locationOK {
			return false
		}
	}

	return true
}

// Apply returns the subset of jobs that pass IsRelevant.
func Apply(jobs []store.Job) []store.Job {
	out := make([]store.Job, 0, len(jobs))
	for _, j := range jobs {
		if IsRelevant(j) {
			out = append(out, j)
		}
	}
	return out
}

